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

func ProcessingLongContentWorkflow(ctx workflow.Context, htmlContent string) error {
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

	articleArt := &activity.ProcessingActivities{}
	llmAct := &activity.LLMActivities{}
	syncAtc := &activity.SyncActivities{}

	var chunkResult *activity.ChunkResult
	if err := workflow.ExecuteActivity(ctx, articleArt.CleanAndChunk, htmlContent, 1000, 200).Get(ctx, &chunkResult); err != nil {
		return err
	}

	// 2. 准备 Embedding 的文本 (关键修改点)
	var textsToEmbed []string
	for _, chunk := range chunkResult.Chunks {
		// 【核心技巧】：为每个切片加上“全局上下文”
		// 格式：标题 + 作者 + 正文片段
		// 这样每个切片都变成了“自带主语”的独立微型文章
		enrichedText := fmt.Sprintf("Title: %s\nAuthor: %s\nContent: %s", "【付鹏说13-第六期】2026市场核心主线——关注美债利率曲线与AI应用落地的三种可能", "付鹏", chunk)
		textsToEmbed = append(textsToEmbed, enrichedText)
	}

	// 3. 批量 Embedding (需要 LLM 接口支持 Batch，或者在 Activity 里循环)
	// 假设 llmAct.BatchEmbedding 支持 []string -> [][]float32
	var vectors [][]float32
	if err := workflow.ExecuteActivity(ctx, llmAct.BatchEmbedding, textsToEmbed).Get(ctx, &vectors); err != nil {
		return err
	}

	var points []*activity.UpsertRow
	articleId := "example-article-id" + uuid.New().String()
	for i, vector := range vectors {
		chunkPayload := map[string]interface{}{
			"article_id":  articleId,
			"text":        chunkResult.Chunks[i],
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

	if err := workflow.ExecuteActivity(ctx, syncAtc.QdrantBatchUpsertToCollection, &activity.QdrantUpsertReq{
		Rows:    points,
		ColName: util.CollectionName,
	}).Get(ctx, nil); err != nil {
		logger.Error("向 Qdrant 批量写入向量失败", "错误", err)
		return err
	}

	return nil
}
