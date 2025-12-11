package activity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/invopop/jsonschema"
	"github.com/sashabaranov/go-openai"
	"go.temporal.io/sdk/activity"
	"rag4financenew/ai"
	"rag4financenew/client"
	"rag4financenew/util"
	"strings"
)

type LLMActivities struct {
}

// Deprecated: 请使用 SessionActivities.GetStartMessages() 方法
func (l *LLMActivities) ConstactParam(ctx context.Context, question string) ([]openai.ChatCompletionMessage, error) {
	return []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: util.SystemPrompt},
		{Role: openai.ChatMessageRoleUser, Content: question},
	}, nil
}

func (l *LLMActivities) ChatWithLLM(ctx context.Context, params ChatWithLLMParams) (*openai.ChatCompletionMessage, error) {
	req := openai.ChatCompletionRequest{
		Model:          params.ModelName,
		Messages:       params.Messages,
		Tools:          params.Tools,
		ResponseFormat: params.ResponseFormat,
	}

	resp, err := client.AI.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, errors.New("无响应")
	}

	return &(resp.Choices[0].Message), nil
}

func (l *LLMActivities) Embedding(ctx context.Context, text string) ([]float32, error) {
	logger := activity.GetLogger(ctx)
	logger.Info(fmt.Sprintf("正在生成向量，文本长度: %d\n", len(text)))

	vec, err := ai.GetEmbedding(ctx, text)
	if err != nil {
		return nil, err
	}
	return vec, nil
}

// DynamicRouteQuery 动态分析用户意图 todo 当前只传递问题的话会丢失上下文，导致tools返回错误或者没有返回
func (l *LLMActivities) DynamicRouteQuery(ctx context.Context, query string) (*RouterIntent, error) {
	// 1. 动态构建工具描述清单
	// 我们要把 client.Tools 里的内容变成 LLM 能读懂的菜单
	var toolsMenu strings.Builder
	toolsMenu.WriteString("你是一个智能路由助手。当前系统支持以下工具：\n\n")

	// 遍历全局工具列表
	for _, tool := range client.Tools {
		// 格式：- tool_name: tool_description
		toolsMenu.WriteString(fmt.Sprintf("- %s: %s\n", tool.Function.Name, tool.Function.Description))
	}

	// 添加通用选项
	toolsMenu.WriteString("- general_chat: 当用户只是打招呼、闲聊，或问题与上述工具完全无关时使用。\n")

	// 2. 生成 JSON Schema (用于 Structured Output)
	reflector := jsonschema.Reflector{ExpandedStruct: true}
	schema := reflector.Reflect(&RouterIntent{})

	// 【进阶技巧】如果你想追求极致的 Strict 模式，
	// 你甚至可以在这里动态修改 schema.Properties["tool_name"].Enum
	// 把所有 tool.Function.Name 放进去，强制 LLM 只能选存在的工具。
	// 但通常上面的 Prompt 引导已经足够好用了。

	schemaBytes, _ := json.Marshal(schema)

	// 3. 调用 LLM 进行分类
	resp, err := client.AI.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4o, // 建议用 4o 或 4o-mini，速度快且准
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: toolsMenu.String() + "\n请根据用户问题，从上述列表中选择最合适的一个工具名称 (tool_name)。",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: query,
			},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONSchema,
			JSONSchema: &openai.ChatCompletionResponseFormatJSONSchema{
				Name:   "tool_routing",
				Schema: json.RawMessage(schemaBytes),
				Strict: true,
			},
		},
	})

	if err != nil {
		return nil, err
	}

	var intent RouterIntent
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &intent); err != nil {
		return nil, err
	}

	return &intent, nil
}
