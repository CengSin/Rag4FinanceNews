package activity

import (
	"context"
	"errors"
	"fmt"
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
	return map[string]any{
		"id":          event.Data["id"],
		"table_name":  event.TableName,
		"title":       event.Data["title"],
		"summary":     event.Data["summary"],
		"type_name":   event.TypeName,
		"textToIndex": fmt.Sprintf("%s\n%s", event.Data["title"], event.Data["summary"]),
	}, nil
}

type articleHandle struct{}

func (a articleHandle) Process(event CDCEvent) (map[string]any, error) {
	return map[string]any{
		"id":          event.Data["id"],
		"table_name":  event.TableName,
		"title":       event.Data["title"],
		"content":     event.Data["content"],
		"textToIndex": fmt.Sprintf("%s\n%s", event.Data["title"], event.Data["content"]),
	}, nil
}
