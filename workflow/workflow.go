package workflow

import (
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sashabaranov/go-openai"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"rag4financenew/activity"
	"rag4financenew/util"
	"time"
)

// ProcessArticleWorkflow 负责协调文章的清洗、增强、向量化和存储
func ProcessArticleWorkflow(ctx workflow.Context, article activity.InputArticle) error {
	// 1. 设置 Activity 选项
	// LLM 接口通常较慢，且可能限流，因此我们需要较长的超时时间和指数退避的重试策略
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second * 1,  // 首次重试间隔
			BackoffCoefficient: 2.0,              // 指数退避 (1s -> 2s -> 4s...)
			MaximumInterval:    time.Second * 60, // 最大间隔
			MaximumAttempts:    5,                // 最大尝试次数 (防止无限死循环)
		},
	}

	ctx = workflow.WithActivityOptions(ctx, ao)
	logger := workflow.GetLogger(ctx)

	// 为了类型安全，最好定义一个指向 Activities 结构体的指针（虽然在 Workflow 中不直接调用方法）
	var a *activity.Activities

	// -------------------------------------------------------
	// 第一步：调用 LLM 生成摘要
	// -------------------------------------------------------
	logger.Info("开始生成摘要 workflow", "ArticleID", article.ID)
	var summary string
	if err := workflow.ExecuteActivity(ctx, a.SummarizeActivity, article).Get(ctx, &summary); err != nil {
		logger.Error("摘要生成失败", "Error", err)
		return err
	}

	// -------------------------------------------------------
	// 第二步：生成向量 (Embedding)
	// -------------------------------------------------------
	// 策略：我们将 "标题 + 栏目 + 摘要" 组合在一起进行向量化。
	// 这样检索时既能匹配标题关键词，又能匹配摘要中的语义。
	textToEmbed := fmt.Sprintf("标题：%s\n栏目：%s\n摘要：%s", article.Title, article.ColumnID, summary)
	logger.Info("开始生成向量", "TextLength", len(textToEmbed))

	var vector []float32
	if err := workflow.ExecuteActivity(ctx, a.EmbedActivity, textToEmbed).Get(ctx, &vector); err != nil {
		logger.Error("向量生成失败", "Error", err)
		return err
	}

	// -------------------------------------------------------
	// 第三步：组装数据并存入 Qdrant
	// -------------------------------------------------------
	processArticle := activity.ProcessedArticle{
		InputArticle: article,
		Summary:      summary,
		Vector:       vector,
	}
	logger.Info("开始存入数据库")
	if err := workflow.ExecuteActivity(ctx, a.UpsertActivity, processArticle).Get(ctx, nil); err != nil {
		logger.Error("数据库写入失败", "Error", err)
		return err
	}

	logger.Info("文章处理流程完成！", "ArticleID", article.ID)
	return nil
}

func RagChatWorkflow(ctx workflow.Context, question string) (string, error) {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second * 1,  // 首次重试间隔
			BackoffCoefficient: 2.0,              // 指数退避 (1s -> 2s -> 4s...)
			MaximumInterval:    time.Second * 60, // 最大间隔
			MaximumAttempts:    5,                // 最大尝试次数 (防止无限死循环)
		},
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	m := activity.MCPActivities{}
	l := activity.LLMActivities{}
	// 定义最大循环次数，防止 LLM 发疯陷入死循环
	const MaxTurns = 10

	var tools []mcp.Tool
	if err := workflow.ExecuteActivity(ctx, m.ListTools).Get(ctx, &tools); err != nil {
		return "", err
	}

	var openAITools []openai.Tool
	for _, t := range tools {
		openAITools = append(openAITools, openai.Tool{
			Type: "function",
			Function: &openai.FunctionDefinition{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema, // MCP 的 Schema 和 OpenAI 是完全兼容的！
			},
		})
	}

	// Step 1: LLM 思考 (这里简化，假设我们直接构建 Prompt 让 LLM 决定)
	// 在实际代码中，这一步通常是调用 LLM 的 ChatCompletion API (也是一个 Activity)
	// 假设 LLM 返回了一个 "Tool Call" 指令
	var messages []openai.ChatCompletionMessage
	if err := workflow.ExecuteActivity(ctx, l.ConstactParam, question).Get(ctx, &messages); err != nil {
		return "", err
	}

	for i := 0; i < MaxTurns; i++ {
		chatParams := activity.ChatWithLLMParams{
			ModelName: util.ModelName,
			Messages:  messages,
			Tools:     openAITools,
		}
		resp := new(openai.ChatCompletionMessage)
		if err := workflow.ExecuteActivity(ctx, l.ChatWithLLM, chatParams).Get(ctx, &resp); err != nil {
			return "", err
		}

		messages = append(messages, *resp)

		if len(resp.ToolCalls) == 0 {
			return resp.Content, nil
		}

		for _, tool := range resp.ToolCalls {
			var toolResult string

			callParams := activity.CallToolParams{
				ToolName:  tool.Function.Name,
				Arguments: tool.Function.Arguments,
			}

			if err := workflow.ExecuteActivity(ctx, m.Call, callParams).Get(ctx, &toolResult); err != nil {
				return "", err
			}

			messages = append(messages, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    toolResult,
				ToolCallID: tool.ID,
			})
		}
	}

	return "Exceeded maximum conversation turns", nil
}
