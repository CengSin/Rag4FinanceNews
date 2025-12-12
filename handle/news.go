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
	// e.Rows 里的数据是原始数组，e.Table.Columns 存了列名信息
	// 我们要把它们“拉链”到一起

	// 遍历所有变动行
	for i, row := range e.Rows {
		// 处理 Update 事件的特殊逻辑
		// Update 事件在 e.Rows 里是成对出现的：[旧值, 新值, 旧值, 新值...]
		// 我们只需要同步“新值”，所以跳过偶数索引 (0, 2, 4...)
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
			Type:    activity.News,
			Payload: raw,
			Table:   msgEvent.Table(),
		}

		// 3. 触发 Temporal Workflow
		// 这里的 ID 用 TableName + PrimaryKey 组合最好，保证防抖和唯一性
		// 假设第一个字段是主键，简单起见我们先用随机或表名
		workflowId := fmt.Sprintf("sync-%s-%v", msgEvent.Table(), msgEvent.ID())

		options := temporalClient.StartWorkflowOptions{
			ID:        workflowId,
			TaskQueue: SyncQueueName,
		}

		we, err := client.SyncTemporal.ExecuteWorkflow(ctx, options, workflow.HandleCdcEvent, env)
		if err != nil {
			log.Printf("启动 Workflow 失败: %v", err)
			return nil // 不要返回 err，否则 Canal 会停止监听
		}
		fmt.Printf("已触发 Workflow, RunID: %s, 数据: %v\n", we.GetRunID(), rowData["name"]) // 假设有个 name 字段
	}

	return nil
}

func buildEvent(e *canal.RowsEvent, data map[string]any) activity.CDCEvent {
	switch e.Table.Name {
	case "content_messages":
		return buildContentMessageEvent(e, data)
	default:
		return nil
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
