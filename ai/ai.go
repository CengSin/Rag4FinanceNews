package ai

import (
	"context"
	"errors"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"rag4financenew/client"
)

func GetEmbedding(ctx context.Context, text string) ([]float32, error) {
	res, err := client.AI.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input:          text,
		Model:          "qwen/qwen3-embedding-8b",
		EncodingFormat: openai.EmbeddingEncodingFormatFloat,
		Dimensions:     1536,
	})
	if err != nil {
		return nil, err
	}
	if len(res.Data) == 0 {
		return nil, errors.New("no embedding data")
	}

	// 4. 返回向量 (OpenAI Small 模型通常是 1536 维)
	vec := res.Data[0].Embedding
	fmt.Printf("向量维度检查: %d\n", len(vec)) // <--- 必须确认是 1536

	if len(vec) != 1536 {
		// 如果这里打印出 4096，你需要修改 Qdrant 创建 Collection 时的 Size 为 4096
		return nil, errors.New(fmt.Sprintf("维度不匹配！期望 1536，实际返回 %d", len(vec)))
	}
	return vec, nil
}
