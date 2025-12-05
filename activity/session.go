package activity

import (
	"context"
	"github.com/sashabaranov/go-openai"
	"rag4financenew/dao"
	"rag4financenew/util"
)

type SessionActivities struct {
}

func (s *SessionActivities) GetStartMessages(ctx context.Context, req MessagesReq) ([]openai.ChatCompletionMessage, error) {
	messages, err := dao.QueryMessagesBySessionId(ctx, req.SessionId)
	if err != nil {
		return nil, err
	}

	if len(messages) == 0 {
		messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleSystem, Content: util.SystemPrompt})
	}

	messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: req.Question})
	return messages, nil
}

func (s *SessionActivities) UpdateMessages(ctx context.Context, req UpdateMessagesReq) error {
	return dao.UpdateMessagesBySessionId(ctx, req.SessionId, req.Messages)
}
