package client

import (
	"context"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sashabaranov/go-openai"
	"log"
)

var (
	McpClient *client.Client
	Tools     []openai.Tool
)

func InitMcpClient(server string) {
	createStreamableHTTPClient(server)
}

func createStreamableHTTPClient(server string) {
	// Create StreamableHTTP client
	c, err := client.NewStreamableHttpClient(server)
	if err != nil {
		log.Fatalln("create streamable http client failed, err ", err)
	}

	ctx := context.Background()

	// 3. 握手 (Initialize)
	// 就像 USB 插入时的握手，交换能力信息
	initializeRequest := mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			Capabilities:    mcp.ClientCapabilities{},
			ClientInfo: mcp.Implementation{
				Name:    "rag_finance_news_tools",
				Version: "1.0.0",
			},
		},
	}

	// Initialize
	if _, err = c.Initialize(ctx, initializeRequest); err != nil {
		log.Fatal(err)
	}

	// Use client
	tools, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Available tools: %d", len(tools.Tools))
	McpClient = c
}

func InitTools() {
	resp, err := McpClient.ListTools(context.Background(), mcp.ListToolsRequest{})
	if err != nil {
		panic(err)
	}

	for _, t := range resp.Tools {
		Tools = append(Tools, openai.Tool{
			Type: "function",
			Function: &openai.FunctionDefinition{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema, // MCP 的 Schema 和 OpenAI 是完全兼容的！
			},
		})
	}
}
