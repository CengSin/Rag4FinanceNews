package activity

import (
	"context"
	"encoding/json"
	"fmt"
)

type EventActivities struct {
}

// ProcessEvent 使用 CdcEnvelope 以支持多类型事件的扩展，同时避免接口反序列化问题。
func (e EventActivities) ProcessEvent(ctx context.Context, env CdcEnvelope) (map[string]any, error) {
	switch env.Type {
	case News:
		var event ContentMessageEvent
		if err := json.Unmarshal(env.Payload, &event); err != nil {
			return nil, err
		}
		return event.ToPayload()
	case Articles:
		var event ArticleEntryEvent
		if err := json.Unmarshal(env.Payload, &event); err != nil {
			return nil, err
		}
		return event.ToPayload()
	default:
		return nil, fmt.Errorf("unsupported cdc type: %s", env.Type)
	}
}
