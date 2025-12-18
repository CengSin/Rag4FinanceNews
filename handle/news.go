package handle

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-mysql-org/go-mysql/canal"
	temporalClient "go.temporal.io/sdk/client"
	"log"
	"rag4financenew/activity"
	"rag4financenew/client"
	"rag4financenew/workflow"
	"time"
)

type NewsHandler struct {
	canal.DummyEventHandler // 继承默认的空实现，我们要重写 OnRow
}

func (n *NewsHandler) OnRow(e *canal.RowsEvent) error {
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancelFunc()

	// ---------------------------------------------------------
	// 知识点：go-mysql 的 RowsEvent 结构
	// e.Action: insert, update, delete
	// e.Rows: [][]interface{} 一个二维数组，表示受影响的行
	// ---------------------------------------------------------

	// 遍历所有变动行
	for i, row := range e.Rows {
		// 1. 针对 UPDATE 事件的特殊处理
		// Update 事件在 e.Rows 里是成对出现的：[旧值, 新值, 旧值, 新值...]
		// 偶数索引 (0, 2...) 是旧值，奇数索引 (1, 3...) 是新值。
		// 我们通常只关心"新值"（用于更新向量），但在极少数情况下，如果主键变了，可能需要用旧值删、新值加。
		// 这里假设主键不变更，直接跳过旧值。
		if e.Action == canal.UpdateAction && i%2 == 0 {
			continue
		}

		// 1. 数据组装：将 Slice 转为 Map
		rowData := map[string]any{}
		for colIndex, col := range e.Table.Columns {
			// e.Table.Columns[colIndex].Name 是列名
			// row[colIndex] 是对应的值
			rowData[col.Name] = row[colIndex]
		}

		msgEvent := buildEvent(e, rowData)
		if msgEvent == nil {
			continue
		}

		raw, err := json.Marshal(msgEvent)
		if err != nil {
			log.Printf("序列化 CDC 事件失败: %v", err)
			continue
		}

		env := activity.CdcEnvelope{
			Type:    getType(e.Table.Name),
			Payload: raw,
			Table:   msgEvent.Table(),
			Action:  e.Action, // 显式传递 Action，供 Workflow 判断
		}

		// 3. 触发 Temporal Workflow
		// 这里的 ID 用 TableName + PrimaryKey 组合最好，保证防抖和唯一性
		// 假设第一个字段是主键，简单起见我们先用随机或表名
		workflowId := fmt.Sprintf("sync-%s-%v-%s", msgEvent.Table(), msgEvent.ID(), e.Action)

		options := temporalClient.StartWorkflowOptions{
			ID:        workflowId,
			TaskQueue: SyncQueueName,
		}

		_, err = client.SyncTemporal.ExecuteWorkflow(ctx, options, workflow.HandleCdcEvent, env)
		if err != nil {
			log.Printf("启动 Workflow 失败: %v", err)
			return nil // 不要返回 err，否则 Canal 会停止监听
		}
		log.Printf("CDC 事件触发: Table=%s, Action=%s, ID=%v", e.Table.Name, e.Action, msgEvent.ID())
	}

	return nil
}

func getType(name string) activity.TypeNameEnum {
	switch name {
	case "content_messages":
		return activity.News
	case "article_entries":
		return activity.Articles
	default:
		return ""
	}
}

func buildEvent(e *canal.RowsEvent, data map[string]any) activity.CDCEvent {
	switch e.Table.Name {
	case "content_messages":
		return buildContentMessageEvent(e, data)
	case "article_entries":
		return buildArticleEntryEvent(e, data)
	default:
		return nil
	}
}

func buildArticleEntryEvent(e *canal.RowsEvent, rowData map[string]any) *activity.ArticleEntryEvent {
	isDeleted := rowData["is_deleted"].(int8)
	createdBy := rowData["created_by"].(int64)
	id := rowData["id"].(int64)
	createdAt := parseTime(rowData["created_at"])
	updatedAt := parseTime(rowData["updated_at"])

	return &activity.ArticleEntryEvent{
		TableName:    e.Table.Name,
		Action:       e.Action,
		TypeName:     activity.News,
		Id:           id,
		Title:        toString(rowData["title"]),
		ContentShort: toString(rowData["content_short"]),
		Content:      toString(rowData["content"]),
		CreatedBy:    createdBy,
		CreatedAt:    createdAt.Unix(),
		UpdatedAt:    updatedAt.Unix(),
		IsDeleted:    isDeleted == 1,
	}
}

func buildContentMessageEvent(e *canal.RowsEvent, rowData map[string]interface{}) *activity.ContentMessageEvent {
	score := rowData["score"].(int64)
	isDeleted := rowData["is_deleted"].(int8)
	createdBy := rowData["created_by"].(int64)
	id := rowData["id"].(int64)
	createdAt := parseTime(rowData["created_at"])

	return &activity.ContentMessageEvent{
		TableName: e.Table.Name,
		Action:    e.Action,
		TypeName:  activity.News,
		Id:        id,
		Title:     toString(rowData["title"]),
		Summary:   toString(rowData["summary"]),
		CreatedAt: createdAt.Unix(),
		CreatedBy: createdBy,
		IsDeleted: isDeleted,
		Score:     score,
	}
}

func parseTime(v any) time.Time {
	switch val := v.(type) {
	case time.Time:
		return val
	case string:
		t, _ := time.Parse(time.DateTime, val)
		return t
	case []byte:
		t, _ := time.Parse(time.DateTime, string(val))
		return t
	default:
		return time.Time{}
	}
}

func toString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	default:
		return fmt.Sprint(v)
	}
}
