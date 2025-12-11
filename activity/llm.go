package activity

import (
	"context"
	"errors"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"go.temporal.io/sdk/activity"
	"rag4financenew/ai"
	"rag4financenew/client"
	"rag4financenew/util"
)

type LLMActivities struct {
}

// Deprecated: 请使用 SessionActivities.GetStartMessages() 方法
func (l *LLMActivities) ConstactParam(ctx context.Context, question string) ([]openai.ChatCompletionMessage, error) {
	return []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: util.SystemPrompt},
		{Role: openai.ChatMessageRoleUser, Content: question},
	}, nil
}

func (l *LLMActivities) ChatWithLLM(ctx context.Context, params ChatWithLLMParams) (*openai.ChatCompletionMessage, error) {
	req := openai.ChatCompletionRequest{
		Model:    params.ModelName,
		Messages: params.Messages,
		Tools:    params.Tools,
	}

	resp, err := client.AI.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, errors.New("无响应")
	}

	return &(resp.Choices[0].Message), nil
}

func (l *LLMActivities) Embedding(ctx context.Context, text string) ([]float32, error) {
	logger := activity.GetLogger(ctx)
	logger.Info(fmt.Sprintf("正在生成向量，文本长度: %d\n", len(text)))

	vec, err := ai.GetEmbedding(ctx, text)
	if err != nil {
		return nil, err
	}
	return vec, nil
}
