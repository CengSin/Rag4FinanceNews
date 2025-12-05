package activity

import (
	"context"
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
	"go.temporal.io/sdk/activity"
	"rag4financenew/client"
)

type MCPActivities struct {
}

func (m *MCPActivities) ListTools(ctx context.Context) ([]mcp.Tool, error) {
	tools, err := client.McpClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, err
	}

	return tools.Tools, nil
}

func (m *MCPActivities) Call(ctx context.Context, params CallToolParams) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info(fmt.Sprintf("🔧 Activity 正在调用 MCP 工具: %s\n", params.ToolName))

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      params.ToolName,
			Arguments: params.Arguments,
		},
	}

	result, err := client.McpClient.CallTool(ctx, req)
	if err != nil {
		// Temporal 会自动捕获这个 error 并根据重试策略重试
		return "", fmt.Errorf("调用 MCP 失败: %w", err)
	}

	// 3. 处理结果
	if result.IsError {
		return "", fmt.Errorf("MCP 工具返回错误: %v", result.Content)
	}

	if len(result.Content) == 0 {
		return "未找到任何资料", nil
	}

	// 提取文本结果返回
	// 这里简化处理，只提取第一个文本内容
	return result.Content[0].(mcp.TextContent).Text, nil
}
