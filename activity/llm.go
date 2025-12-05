package activity

import (
	"context"
	"errors"
	"github.com/sashabaranov/go-openai"
	"rag4financenew/client"
	"rag4financenew/util"
)

type LLMActivities struct {
}

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
