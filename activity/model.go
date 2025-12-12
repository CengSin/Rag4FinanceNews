package activity

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sashabaranov/go-openai"
)

type InputArticle struct {
	ID        string
	Title     string
	Content   string
	ColumnID  string   // 栏目ID
	Tags      []string // 原始标签
	PublishTs int64    // 发布时间戳
	URL       string
}

// ProcessedArticle 是处理后的文章，准备入库
type ProcessedArticle struct {
	InputArticle
	Summary string    // LLM 生成的摘要
	Vector  []float32 // Embedding 向量
}

type CallToolParams struct {
	ToolName  string
	Arguments string
}

type ChatWithLLMParams struct {
	Messages       []openai.ChatCompletionMessage
	Tools          []openai.Tool
	ModelName      string
	ResponseFormat *openai.ChatCompletionResponseFormat
}

type MessagesReq struct {
	SessionId string `json:"session_id"`
	Question  string `json:"question"`
	Update    bool   `json:"-"`
}

type UpdateMessagesReq struct {
	SessionId string
	Messages  []openai.ChatCompletionMessage
}

type TypeNameEnum string

const (
	News     TypeNameEnum = "news"
	Articles TypeNameEnum = "articles"
)

type CDCEvent interface {
	ID() int64
	Table() string
	Type() TypeNameEnum
	TextToIndex() string
	ToPayload() (map[string]any, error)
}

// ContentMessageEvent 针对 content_messages 业务表的 CDC 事件
type ContentMessageEvent struct {
	TableName string
	Action    string
	TypeName  TypeNameEnum

	Id        int64
	Title     string
	Summary   string
	CreatedAt int64
	CreatedBy int64
	IsDeleted int8
	Score     int64
}

func (e *ContentMessageEvent) ID() int64 { return e.Id }

func (e *ContentMessageEvent) Table() string      { return e.TableName }
func (e *ContentMessageEvent) Type() TypeNameEnum { return e.TypeName }

func (e *ContentMessageEvent) TextToIndex() string {
	return fmt.Sprintf("%s\n%s", e.Title, e.Summary)
}

func (e *ContentMessageEvent) ToPayload() (map[string]any, error) {
	// 过滤不符合条件的事件
	if e.Score != 2 {
		return nil, nil
	}
	if e.IsDeleted == 1 {
		return nil, nil
	}
	if e.CreatedBy != 1160671 {
		return nil, nil
	}

	text := e.TextToIndex()
	if text == "" {
		return nil, errors.New("text to index is empty")
	}

	return map[string]any{
		"id":          fmt.Sprint(e.Id),
		"table_name":  e.TableName,
		"title":       e.Title,
		"summary":     e.Summary,
		"type_name":   e.TypeName,
		"created_at":  e.CreatedAt,
		"textToIndex": text,
	}, nil
}

// CdcEnvelope 用于跨进程传输 CDC 事件，避免接口无法反序列化的问题。
// Type 指示具体业务类型，Payload 是业务事件的 JSON。
type CdcEnvelope struct {
	Type    TypeNameEnum    `json:"type"`
	Payload json.RawMessage `json:"payload"`
	Table   string          `json:"table,omitempty"`
	Action  string          `json:"action,omitempty"`
}

type RouterIntent struct {
	// 让 LLM 只有两个选择：要么选一个具体的工具名，要么选 "general_chat"
	ToolName  string `json:"tool_name" jsonschema_description:"最适合解决用户问题的工具名称。如果不需要使用工具（如纯闲聊），请填 'general_chat'"`
	Reasoning string `json:"reasoning" jsonschema_description:"选择该工具的简短理由"`
	ArgsHint  string `json:"args_hint" jsonschema_description:"(可选) 提取用户问题中可能的关键参数，辅助后续步骤"`
}
