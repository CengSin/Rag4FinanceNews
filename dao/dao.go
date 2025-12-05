package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"rag4financenew/client"
	"rag4financenew/util"
)

func QueryMessagesBySessionId(ctx context.Context, sessionId string) ([]openai.ChatCompletionMessage, error) {
	s := client.Redis.Get(ctx, fmt.Sprintf(util.ChatHistoryFormat, sessionId)).String()
	if len(s) == 0 {
		return nil, nil
	}

	var result []openai.ChatCompletionMessage
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return nil, err
	}

	return result, nil
}

func UpdateMessagesBySessionId(ctx context.Context, sessionId string, messages []openai.ChatCompletionMessage) error {
	if len(messages) == 0 {
		return nil
	}

	chat, err := json.Marshal(messages)
	if err != nil {
		return err
	}
	return client.Redis.Set(ctx, fmt.Sprintf(util.ChatHistoryFormat, sessionId), string(chat), 0).Err()
}
