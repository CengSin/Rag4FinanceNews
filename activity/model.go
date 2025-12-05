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
