package workflow

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"rag4financenew/activity"
	"time"
)

func HandleCdcEvent(ctx workflow.Context, env activity.CdcEnvelope) error {
	// 设置重试策略
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval: time.Second * 1,
			MaximumInterval: time.Second * 10,
			MaximumAttempts: 3, // 最多重试3次（生产环境可以设大点）
		},
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	// 2. 初始化 Activities
	syncAct := activity.SyncActivities{}
	llmAct := activity.LLMActivities{}
	eventAct := activity.EventActivities{}

	var payload map[string]any
	if err := workflow.ExecuteActivity(ctx, eventAct.ProcessEvent, env).Get(ctx, &payload); err != nil {
		return err
	}

	if len(payload) == 0 {
		return nil
	}

	// 2. 判断是否为删除操作
	// 来源有二：Binlog 是 delete，或者 业务字段 is_deleted == 1
	isDeleted := false
	if val, ok := payload["action"]; ok && val == "delete" {
		isDeleted = true
	}
	if val, ok := payload["is_deleted"]; ok {
		// JSON 数字转 map[string]any 后通常是 float64
		if v, ok := val.(int8); ok && v == 1 {
			isDeleted = true
		}
	}

	// 3. 分支执行
	if isDeleted {
		// --- 删除分支 ---
		articleId := fmt.Sprint(payload["id"])
		return workflow.ExecuteActivity(ctx, syncAct.QdrantDelete, articleId).Get(ctx, nil)
	}

	textToIndex := fmt.Sprint(payload["textToIndex"])
	if len(textToIndex) == 0 {
		return nil
	}

	// 执行embedding
	var vector []float32
	if err := workflow.ExecuteActivity(ctx, llmAct.Embedding, textToIndex).Get(ctx, &vector); err != nil {
		return err
	}

	var rowId *qdrant.PointId
	if err := workflow.ExecuteActivity(ctx, syncAct.QdrantQueryOrConstruct, vector).Get(ctx, &rowId); err != nil {
		return err
	}

	if rowId == nil {
		rowId = qdrant.NewID(uuid.New().String())
	}

	row := qdrant.PointStruct{
		Id:      rowId,
		Vectors: qdrant.NewVectors(vector...),
		Payload: qdrant.NewValueMap(payload),
	}

	// 3. 执行 Activity
	// Temporal 会记录这一步的状态
	if err := workflow.ExecuteActivity(ctx, syncAct.QdrantUpsert, &row).Get(ctx, nil); err != nil {
		return err
	}
	return nil
}
