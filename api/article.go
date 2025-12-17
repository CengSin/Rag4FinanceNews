package api

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	temporalClient "go.temporal.io/sdk/client"
	"io"
	"os"
	"rag4financenew/client"
	"rag4financenew/workflow"
)

func ParseLocalArticle(e echo.Context) error {
	ctx := e.Request().Context()

	fileName := e.Param("name")
	file, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	// 2. 配置工作流选项
	startWorkflowOptions := temporalClient.StartWorkflowOptions{
		// ID 是去重的关键！
		// 如果使用了相同的 ID 再次提交，Temporal 默认会报错（防止重复处理同一篇文章）
		ID:        "search_article_" + uuid.New().String(),
		TaskQueue: TaskQueueName,
	}

	if _, err = client.Temporal.ExecuteWorkflow(ctx, startWorkflowOptions, workflow.ProcessingLongContentWorkflow, string(content)); err != nil {
		return err
	}

	return nil
}
