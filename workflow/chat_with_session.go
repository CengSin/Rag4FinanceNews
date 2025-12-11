package workflow

import (
	"github.com/sashabaranov/go-openai"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"rag4financenew/activity"
	"rag4financenew/client"
	"rag4financenew/util"
	"time"
)

// 定义对话历史结构
type ChatHistory struct {
	Messages []openai.ChatCompletionMessage // Message 包含 Role 和 Content
}

// 你的 Workflow 定义
func ChatSessionWorkflow(ctx workflow.Context, req ChatReq) error {
	logger := workflow.GetLogger(ctx)
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

	sessionAct := activity.SessionActivities{}
	llmAct := activity.LLMActivities{}
	mcpAct := activity.MCPActivities{}

	history := make([]openai.ChatCompletionMessage, 0)
	historyLoadedFuture, historyLoadSetter := workflow.NewFuture(ctx)

	workflow.Go(ctx, func(ctx workflow.Context) {
		// 1. 初始化对话状态（内存中的上下文）
		if err := workflow.ExecuteActivity(ctx, sessionAct.GetStartMessages,
			activity.MessagesReq{SessionId: req.SessionID, Question: req.Question}).Get(ctx, &history); err != nil {
			// 如果加载失败，必须处理（这里简单设为空，或者让 Future 返回错误）
			workflow.GetLogger(ctx).Error("Failed to load history", "error", err)
			history = []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: util.SystemPrompt},
			}
		}

		historyLoadSetter.Set(true, nil)
	})

	// 设置处理 Update 的 Handler
	err := workflow.SetUpdateHandler(ctx, "chat_message", func(ctx workflow.Context, req ChatReq) (ChatResp, error) {
		ctx = workflow.WithActivityOptions(ctx, ao)

		_ = historyLoadedFuture.Get(ctx, nil)

		var selectedTools []openai.Tool
		// 🔥 第一步：动态路由
		var intent *activity.RouterIntent
		// 调用刚才写的 Activity
		err := workflow.ExecuteActivity(ctx, llmAct.DynamicRouteQuery, req.Question).Get(ctx, &intent)
		if err != nil {
			logger.Info("Router activity failed, defaulting to general_chat", "error", err)
		} else {
			// 🔥 第二步：根据意图筛选工具
			if intent.ToolName == "general_chat" {
				// 闲聊模式：不给任何工具，防止 LLM 瞎调用
				selectedTools = nil
			} else {
				// 工具模式：去 client.Tools 里找这个名字的工具
				// 这样无论你后期加了什么 MCP 工具，这里都不用改代码！
				for _, tool := range client.Tools {
					if tool.Function.Name == intent.ToolName {
						selectedTools = append(selectedTools, tool)
						break // 找到一个就可以停了（或者支持多选）
					}
				}

				// 兜底：如果路由返回了名字但没找到工具（极少见），可以fallback到全量工具
				if len(selectedTools) == 0 {
					// logger.Warn("Router selected unknown tool, falling back to all tools", "tool", intent.ToolName)
					selectedTools = client.Tools
				}
			}
		}

		// A. 将用户消息加入历史
		history = append(history, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: req.Question})

		for i := 0; i < 10; i++ {
			// B. 可以在这里做滑动窗口处理 (Context Window Management)
			// if len(history) > 10 { history = history[2:] }

			// C. 调用 Activity 执行 RAG 和 LLM 推理
			// 注意：这里把整个 history 传给 Activity
			var llmResponse openai.ChatCompletionMessage
			err := workflow.ExecuteActivity(ctx, llmAct.ChatWithLLM, activity.ChatWithLLMParams{
				Messages:  history,
				Tools:     selectedTools,
				ModelName: util.ModelName,
			}).Get(ctx, &llmResponse)
			if err != nil {
				return ChatResp{SessionID: req.SessionID, ReplyMessage: err.Error()}, err
			}

			history = append(history, llmResponse)
			if len(llmResponse.ToolCalls) == 0 {
				// 直接执行并返回结果
				workflow.Go(ctx, func(ctx workflow.Context) {
					workflow.ExecuteActivity(ctx, sessionAct.UpdateMessages, activity.UpdateMessagesReq{
						SessionId: req.SessionID,
						Messages:  history,
					})
				})
				return ChatResp{SessionID: req.SessionID, ReplyMessage: llmResponse.Content}, nil
			}

			for _, tool := range llmResponse.ToolCalls {
				var toolResult string

				callParams := activity.CallToolParams{
					ToolName:  tool.Function.Name,
					Arguments: tool.Function.Arguments,
				}

				if err = workflow.ExecuteActivity(ctx, mcpAct.Call, callParams).Get(ctx, &toolResult); err != nil {
					return ChatResp{}, err
				}

				history = append(history, openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					Content:    toolResult,
					ToolCallID: tool.ID,
				})
			}
		}

		return ChatResp{SessionID: req.SessionID, ReplyMessage: history[len(history)-1].Content}, nil
	})

	if err != nil {
		return err
	}

	// 2. 阻塞 Workflow，防止它结束
	// 我们可以设置一个超时的 Await，如果 30 分钟没有新消息，就结束会话
	// 等待直到 context 被取消或满足特定退出条件
	workflow.AwaitWithTimeout(ctx, 10*time.Minute, func() bool {
		return false // 这里演示简单的一直运行，实际需配合超时逻辑
	})

	return nil
}
