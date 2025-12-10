package activity

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type EventActivities struct {
}

func (e EventActivities) ProcessEvent(ctx context.Context, event CDCEvent) (map[string]any, error) {
	handle := getHandle(event.TypeName)
	return handle.Process(event)
}

func getHandle(typeName TypeNameEnum) eventHandle {
	switch typeName {
	case Articles:
		return articleHandle{}
	default:
		return &newsHandle{}
	}
}

type eventHandle interface {
	Process(event CDCEvent) (map[string]any, error)
}

type defaultHandle struct {
}

func (d *defaultHandle) Process(event CDCEvent) (map[string]any, error) {
	return nil, errors.New("无效的事件类型")
}

type newsHandle struct{}

func (n newsHandle) Process(event CDCEvent) (map[string]any, error) {
	createdAtStr := event.Data["created_at"]
	createAt, _ := time.Parse(time.DateTime, createdAtStr.(string))
	return map[string]any{
		"id":          event.Data["id"],
		"table_name":  event.TableName,
		"title":       event.Data["title"],
		"summary":     event.Data["summary"],
		"type_name":   event.TypeName,
		"created_at":  createAt.Unix(),
		"textToIndex": fmt.Sprintf("%s\n%s", event.Data["title"], event.Data["summary"]),
	}, nil
}

type articleHandle struct{}

func (a articleHandle) Process(event CDCEvent) (map[string]any, error) {
	createdAtStr := event.Data["created_at"]
	createAt, _ := time.Parse(time.DateTime, createdAtStr.(string))
	return map[string]any{
		"id":          event.Data["id"],
		"table_name":  event.TableName,
		"title":       event.Data["title"],
		"content":     event.Data["content"],
		"created_at":  createAt.Unix(),
		"textToIndex": fmt.Sprintf("%s\n%s", event.Data["title"], event.Data["content"]),
	}, nil
}
