package activity

import (
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

// CDCEvent 定义了从 MySQL 捕获的一个事件
type CDCEvent struct {
	TableName string                 `json:"table_name"` // 表名
	Action    string                 `json:"action"`     // insert, update
	TypeName  TypeNameEnum           `json:"typename"`   // news 快讯, article 文章
	Data      map[string]interface{} // 关键！我们将数组转为 map: {"id": 1, "content": "hello"}
}

type RouterIntent struct {
	// 让 LLM 只有两个选择：要么选一个具体的工具名，要么选 "general_chat"
	ToolName  string `json:"tool_name" jsonschema_description:"最适合解决用户问题的工具名称。如果不需要使用工具（如纯闲聊），请填 'general_chat'"`
	Reasoning string `json:"reasoning" jsonschema_description:"选择该工具的简短理由"`
	ArgsHint  string `json:"args_hint" jsonschema_description:"(可选) 提取用户问题中可能的关键参数，辅助后续步骤"`
}
