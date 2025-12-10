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
	Messages  []openai.ChatCompletionMessage
	Tools     []openai.Tool
	ModelName string
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
