package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"rag4financenew/client"
	"rag4financenew/util"
	"sort"
	"strings"
)

func QueryMessagesBySessionId(ctx context.Context, sessionId string) ([]openai.ChatCompletionMessage, error) {
	chatKey := fmt.Sprintf(util.ChatHistoryFormat, sessionId)
	if client.Redis.Exists(ctx, chatKey).Val() > 0 {
		var result []openai.ChatCompletionMessage
		msgs := client.Redis.Get(ctx, chatKey).Val()
		if err := json.Unmarshal([]byte(msgs), &result); err != nil {
			return nil, err
		}
		return result, nil
	}
	return []openai.ChatCompletionMessage{}, nil
}

func UpdateMessages(ctx context.Context, sessionId string, messages []openai.ChatCompletionMessage) error {
	if len(messages) == 0 {
		return nil
	}

	bytes, err := json.Marshal(messages)
	if err != nil {
		return err
	}

	if err = client.Redis.Set(ctx, fmt.Sprintf(util.ChatHistoryFormat, sessionId), string(bytes), 0).Err(); err != nil {
		return err
	}

	return nil
}

func ListSessions(ctx context.Context) ([]string, error) {
	pattern := fmt.Sprintf(util.ChatHistoryFormat, "*")
	iter := client.Redis.Scan(ctx, 0, pattern, 0).Iterator()

	prefix := fmt.Sprintf(util.ChatHistoryFormat, "")
	sessions := make([]string, 0)
	for iter.Next(ctx) {
		key := iter.Val()
		sessionId := strings.TrimPrefix(key, prefix)
		if sessionId == "" {
			continue
		}
		sessions = append(sessions, sessionId)
	}

	if err := iter.Err(); err != nil {
		return nil, err
	}

	sort.Strings(sessions)
	return sessions, nil
}

func DeleteSession(ctx context.Context, sessionId string) error {
	if sessionId == "" {
		return fmt.Errorf("session id is empty")
	}
	return client.Redis.Del(ctx, fmt.Sprintf(util.ChatHistoryFormat, sessionId)).Err()
}
