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
	msgs, err := client.Redis.SMembers(ctx, fmt.Sprintf(util.ChatHistoryFormat, sessionId)).Result()
	if err != nil {
		return nil, err
	}

	var result []openai.ChatCompletionMessage
	for _, msg := range msgs {
		var m openai.ChatCompletionMessage
		if err = m.UnmarshalJSON([]byte(msg)); err != nil {
			return nil, err
		}
		result = append(result, m)
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

func AppendMessagesBySessionId(ctx context.Context, sessionId string, messages []openai.ChatCompletionMessage) error {
	if len(messages) == 0 {
		return nil
	}

	for _, message := range messages {
		bytes, err := message.MarshalJSON()
		if err != nil {
			return err
		}
		if exist, err := client.Redis.SIsMember(ctx, fmt.Sprintf(util.ChatHistoryFormat, sessionId), string(bytes)).Result(); err != nil {
			return err
		} else if !exist {
			if err = client.Redis.SAdd(ctx, fmt.Sprintf(util.ChatHistoryFormat, sessionId), string(bytes), 0).Err(); err != nil {
				return err
			}
		}
	}

	return nil
}
