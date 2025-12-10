package main

import (
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"log"
	"rag4financenew/api"
	"rag4financenew/client"
	"rag4financenew/config"
	"rag4financenew/handle"
	"rag4financenew/util"
)

func main() {
	util.InitSystemPrompt()

	var cfg config.Config
	_ = cleanenv.ReadConfig("./config/config.yaml", &cfg)

	client.InitRedis(cfg.Redis)
	client.InitQdrant(cfg.Qdrant)
	client.InitLLMs(cfg.OpenAI)
	client.InitTemporal(cfg.Temporal, &client.Temporal)
	client.InitTemporal(cfg.SyncTemporal, &client.SyncTemporal)
	client.InitMcpClient(cfg.McpServer)
	client.InitTools()
	defer client.Close()

	runWorker()

	go handle.ChangDataCapture(cfg.Cdc)

	e := echo.New()

	e.Use(middleware.CORS())

	articleGrp := e.Group("/article")
	articleGrp.POST("/process", api.ProcessedArticle)

	sessionGrp := e.Group("/session")
	sessionGrp.GET("/chat/messages", api.NewSession)
	sessionGrp.GET("/history", api.GetSessionHistory)
	sessionGrp.GET("/list", api.ListSessions)
	sessionGrp.DELETE("/:session_id", api.DeleteSession)

	aiGrp := e.Group("/ai")
	aiGrp.POST("/query", api.Question)
	aiGrp.POST("/temporal", api.QuestionOnTemporal)
	aiGrp.POST("/new/session", api.ChatWithLLM)

	if err := e.Start(":8081"); err != nil {
		log.Fatalln(err)
	}
}
