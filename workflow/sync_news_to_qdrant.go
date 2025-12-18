package workflow

import (
	"fmt"
	"github.com/google/uuid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"rag4financenew/activity"
	"rag4financenew/util"
	"time"
)

func HandleCdcEvent(ctx workflow.Context, env activity.CdcEnvelope) error {
	logger := workflow.GetLogger(ctx)
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
	processAct := activity.ProcessingActivities{}

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

	articleId := fmt.Sprint(payload["id"])
	// 3. 分支执行
	if isDeleted {
		// --- 删除分支 ---
		return workflow.ExecuteActivity(ctx, syncAct.QdrantDelete, articleId).Get(ctx, nil)
	}

	textToIndex := fmt.Sprint(payload["textToIndex"])
	if len(textToIndex) == 0 {
		return nil
	}

	var chunkResult activity.ChunkResult
	if err := workflow.ExecuteActivity(ctx, processAct.CleanAndChunkText, textToIndex, 500, 50).Get(ctx, &chunkResult); err != nil {
		return err
	}

	// 2. 准备 Embedding 的文本 (关键修改点)
	var textsToEmbed []string
	for _, chunk := range chunkResult.Chunks {
		// 【核心技巧】：为每个切片加上“全局上下文”
		// 格式：标题 + 作者 + 正文片段
		// 这样每个切片都变成了“自带主语”的独立微型文章
		enrichedText := fmt.Sprintf("Title: %s\nContent: %s", payload["title"], chunk)
		textsToEmbed = append(textsToEmbed, enrichedText)
	}

	// 执行embedding
	var vectors [][]float32
	if err := workflow.ExecuteActivity(ctx, llmAct.BatchEmbedding, textsToEmbed).Get(ctx, &vectors); err != nil {
		return err
	}

	var points []*activity.UpsertRow
	for i, vector := range vectors {
		chunkPayload := map[string]interface{}{
			"id":          articleId,
			"textToIndex": chunkResult.Chunks[i],
			"summary":     payload["summary"],
			"created_at":  payload["created_at"],
			"title":       payload["title"],
			"chunk_index": i,
		}

		pointId := uuid.New().String()
		point := &activity.UpsertRow{
			ID:      pointId,
			Vector:  vector,
			Payload: chunkPayload,
		}

		points = append(points, point)
	}

	if err := workflow.ExecuteActivity(ctx, syncAct.QdrantBatchUpsertToCollection, &activity.QdrantUpsertReq{
		Rows:    points,
		ColName: util.CollectionFupengshuoName,
	}).Get(ctx, nil); err != nil {
		logger.Error("向 Qdrant 批量写入向量失败", "错误", err)
		return err
	}
	return nil
}
