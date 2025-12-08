package api

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sashabaranov/go-openai"
	temporalClient "go.temporal.io/sdk/client"
	"log"
	"net/http"
	"rag4financenew/activity"
	"rag4financenew/client"
	"rag4financenew/util"
	rWorkflow "rag4financenew/workflow"
)

const (
	TaskQueueName = "financial-news-queue"
)

func ProcessedArticle(c echo.Context) error {
	ctx := c.Request().Context()
	var article activity.InputArticle
	if err := c.Bind(&article); err != nil {
		return err
	}

	// 简单校验
	if article.ID == "" || article.Title == "" {
		return fmt.Errorf("ID 和 Title 不能为空")
	}

	// 2. 配置工作流选项
	startWorkflowOptions := temporalClient.StartWorkflowOptions{
		// ID 是去重的关键！
		// 如果使用了相同的 ID 再次提交，Temporal 默认会报错（防止重复处理同一篇文章）
		ID:        "ingest_article_" + article.ID,
		TaskQueue: TaskQueueName,
	}

	res, err := client.Temporal.ExecuteWorkflow(ctx, startWorkflowOptions, rWorkflow.ProcessArticleWorkflow, article)
	if err != nil {
		return err
	}

	response := map[string]string{
		"status":      "accepted",
		"workflow_id": res.GetID(),
		"run_id":      res.GetRunID(),
		"message":     "文章已加入处理队列，稍后可用。",
	}

	return c.JSON(http.StatusOK, response)
}

func Question(c echo.Context) error {
	ctx := c.Request().Context()
	question := c.FormValue("question")
	if len(question) == 0 {
		return nil
	}

	var openAITools []openai.Tool
	mcpTools, err := client.McpClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return err
	}

	for _, t := range mcpTools.Tools {
		openAITools = append(openAITools, openai.Tool{
			Type: "function",
			Function: &openai.FunctionDefinition{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema, // MCP 的 Schema 和 OpenAI 是完全兼容的！
			},
		})
	}

	log.Println("🤖 第一轮：发送用户问题给 LLM...")

	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: util.SystemPrompt},
		{Role: openai.ChatMessageRoleUser, Content: question},
	}
	resp, err := client.AI.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    util.ModelName, // Qwen 对 Tool Use 支持很好
		Messages: messages,
		Tools:    openAITools,
	})
	if err != nil {
		log.Println("chat with llms failed, err ", err)
		return err
	}

	msg := resp.Choices[0].Message

	if len(msg.ToolCalls) > 0 {
		log.Println("⚡ LLM 决定调用工具！")
		messages = append(messages, msg)

		for _, tool := range msg.ToolCalls {
			toolResult, err := client.McpClient.CallTool(ctx, mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      tool.Function.Name,
					Arguments: tool.Function.Arguments,
				},
			})
			if err != nil {
				log.Println("call mcp tools failed, err ", err)
				return err
			}

			messages = append(messages, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    toolResult.Content[0].(mcp.TextContent).Text,
				ToolCallID: tool.ID,
			})
		}
	}

	bytes, _ := json.Marshal(messages)
	fmt.Println(string(bytes))

	res, err := client.AI.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    util.ModelName,
		Messages: messages,
	})
	if err != nil {
		log.Println("llm call failed, err ", err)
		return err
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"answer": res.Choices[0].Message.Content})
}

func QuestionOnTemporal(c echo.Context) error {
	ctx := c.Request().Context()
	sessionId := c.FormValue("session_id")
	question := c.FormValue("question")
	if len(question) == 0 {
		return nil
	}

	if len(sessionId) == 0 {
		sessionId = uuid.New().String()
	}

	// 2. 配置工作流选项
	startWorkflowOptions := temporalClient.StartWorkflowOptions{
		// ID 是去重的关键！
		// 如果使用了相同的 ID 再次提交，Temporal 默认会报错（防止重复处理同一篇文章）
		ID:        "search_article_" + uuid.New().String(),
		TaskQueue: TaskQueueName,
	}

	wf, err := client.Temporal.ExecuteWorkflow(ctx, startWorkflowOptions, rWorkflow.RagChatWorkflow, activity.MessagesReq{
		SessionId: sessionId,
		Question:  question,
	})
	if err != nil {
		return err
	}

	var answer string
	if err = wf.Get(ctx, &answer); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"answer": answer})
}
