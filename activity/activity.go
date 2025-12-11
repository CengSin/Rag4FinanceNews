package activity

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
	"github.com/sashabaranov/go-openai"
	"go.temporal.io/sdk/activity"
	"rag4financenew/ai"
	"rag4financenew/client"
	"rag4financenew/util"
	"strings"
)

type Activities struct {
}

// ---------------------------------------------------------
// Activity 1: LLM 摘要与增强
// ---------------------------------------------------------
func (a *Activities) SummarizeActivity(ctx context.Context, article *InputArticle) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info(fmt.Sprintf("正在使用LLM分析文章：%s\n", article.Title))
	// 调用LLM生成inputArticle的摘要
	// Prompt 示例: "你是一名金融分析师。请总结以下文章的核心投资逻辑，并提取3个关键实体。"
	res, err := client.AI.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: "openai/gpt-5",
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: "你是一名金融分析师。请总结以下文章的核心投资逻辑，并提取3个关键实体。"},
			{Role: openai.ChatMessageRoleUser, Content: fmt.Sprintf("%s\n\n%s", article.Title, article.Content)},
		},
	})
	if err != nil {
		return "", err
	}

	var summaryByAI strings.Builder
	for _, choice := range res.Choices {
		summaryByAI.WriteString("---\n")
		summaryByAI.WriteString(choice.Message.Content)
		summaryByAI.WriteString("\n")
	}
	summary := summaryByAI.String()
	logger.Info("由AI为【", article.Title, "】生成的摘要如下：", summary)
	return summary, nil
}

// ---------------------------------------------------------
// Activity 2: 向量化 (Embedding)
// ---------------------------------------------------------
func (a *Activities) EmbedActivity(ctx context.Context, text string) ([]float32, error) {
	logger := activity.GetLogger(ctx)
	logger.Info(fmt.Sprintf("正在生成向量，文本长度: %d\n", len(text)))

	vec, err := ai.GetEmbedding(ctx, text)
	if err != nil {
		return nil, err
	}
	return vec, nil
}

// ---------------------------------------------------------
// Activity 3: Qdrant 入库 (使用新版 SDK)
// ---------------------------------------------------------
func (a *Activities) UpsertActivity(ctx context.Context, article ProcessedArticle) error {
	logger := activity.GetLogger(ctx)
	logger.Info(fmt.Sprintf("正在写入 Qdrant: %s\n", article.ID))

	// 1. 构建 Payload (元数据)
	// 新版 SDK 使用 map[string]interface{} 或者构建 Value 类型，
	// 但高层封装通常支持直接传 map，SDK 会自动序列化
	payload := map[string]any{
		"article_id": article.ID,
		"title":      article.Title,
		"column_id":  article.ColumnID,
		"summary":    article.Summary, // 存摘要用于 RAG，不存全文
		"tags":       fmt.Sprintf("%s", article.Tags),
		"publish_ts": article.PublishTs,
		"url":        article.URL,
	}

	// 2. 生成 UUID (Qdrant 推荐使用 UUID 格式作为 Point ID)
	// 如果 article.ID 已经是 UUID 格式则直接用，否则需要转换或哈希
	pointID := uuid.New().String()

	upsertPoints := []*qdrant.PointStruct{
		{
			Id:      qdrant.NewIDUUID(pointID),
			Vectors: qdrant.NewVectors(article.Vector...),
			Payload: qdrant.NewValueMap(payload),
		},
	}

	wait := true
	_, err := client.Qdrant.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: util.CollectionName,
		Points:         upsertPoints,
		Wait:           &wait,
	})
	if err != nil {
		return err
	}

	return nil
}
