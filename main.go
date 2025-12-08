package main

import (
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.temporal.io/sdk/worker"
	"log"
	"rag4financenew/activity"
	"rag4financenew/api"
	"rag4financenew/client"
	"rag4financenew/config"
	"rag4financenew/util"
	"rag4financenew/workflow"
)

func main() {
	util.InitSystemPrompt()

	var cfg config.Config
	cleanenv.ReadConfig("./config/config.yaml", &cfg)

	client.InitRedis(cfg.Redis)
	client.InitQdrant(cfg.Qdrant)
	client.InitLLMs(cfg.OpenAI)
	client.InitTemporal(cfg.Temporal)
	client.InitMcpClient(cfg.McpServer)
	client.InitTools()
	defer client.Close()

	// 2. 启动 Worker (处理任务的消费者)
	// 在微服务架构中，Worker 通常是独立运行的进程。
	// 为了演示方便，我们在这里和 HTTP Server 一起启动。
	// ------------------------------------------------------
	w := worker.New(client.Temporal, api.TaskQueueName, worker.Options{})
	w.RegisterWorkflow(workflow.ProcessArticleWorkflow)
	w.RegisterWorkflow(workflow.RagChatWorkflow)
	w.RegisterWorkflow(workflow.ChatWorkflow)
	w.RegisterWorkflow(workflow.ChatSessionWorkflow)

	w.RegisterActivity(&activity.Activities{})
	w.RegisterActivity(&activity.LLMActivities{})
	w.RegisterActivity(&activity.MCPActivities{})
	w.RegisterActivity(&activity.SessionActivities{})

	// 异步启动 Worker
	go func() {
		if err := w.Run(worker.InterruptCh()); err != nil {
			log.Fatalln("Worker 启动失败", err)
		}
	}()

	e := echo.New()

	e.Use(middleware.CORS())

	articleGrp := e.Group("/article")
	articleGrp.POST("/process", api.ProcessedArticle)

	sessionGrp := e.Group("/session")
	sessionGrp.GET("/chat/messages", api.NewSession)

	aiGrp := e.Group("/ai")
	aiGrp.POST("/query", api.Question)
	aiGrp.POST("/temporal", api.QuestionOnTemporal)
	aiGrp.POST("/new/session", api.ChatWithLLM)

	if err := e.Start(":8081"); err != nil {
		log.Fatalln(err)
	}
}
