package main

import (
	"go.temporal.io/sdk/worker"
	"log"
	"rag4financenew/activity"
	"rag4financenew/api"
	"rag4financenew/client"
	"rag4financenew/handle"
	"rag4financenew/workflow"
)

func runWorker() {
	// 2. 启动 Worker (处理任务的消费者)
	// 在微服务架构中，Worker 通常是独立运行的进程。
	// 为了演示方便，我们在这里和 HTTP Server 一起启动。
	// ------------------------------------------------------
	w := worker.New(client.Temporal, api.TaskQueueName, worker.Options{})
	w.RegisterWorkflow(workflow.ProcessArticleWorkflow)
	w.RegisterWorkflow(workflow.RagChatWorkflow)
	w.RegisterWorkflow(workflow.ChatWorkflow)
	w.RegisterWorkflow(workflow.ChatSessionWorkflow)
	w.RegisterWorkflow(workflow.ProcessingLongContentWorkflow)

	w.RegisterActivity(&activity.Activities{})
	w.RegisterActivity(&activity.LLMActivities{})
	w.RegisterActivity(&activity.MCPActivities{})
	w.RegisterActivity(&activity.SessionActivities{})
	w.RegisterActivity(&activity.ProcessingActivities{})
	w.RegisterActivity(&activity.SyncActivities{})

	newsSyncWorker := worker.New(client.SyncTemporal, handle.SyncQueueName, worker.Options{})
	newsSyncWorker.RegisterWorkflow(workflow.HandleCdcEvent)
	newsSyncWorker.RegisterActivity(&activity.SyncActivities{})
	newsSyncWorker.RegisterActivity(&activity.EventActivities{})
	newsSyncWorker.RegisterActivity(&activity.LLMActivities{})

	// 异步启动 Worker
	go func() {
		if err := w.Run(worker.InterruptCh()); err != nil {
			log.Fatalln("Worker 启动失败", err)
		}
	}()

	go func() {
		if err := newsSyncWorker.Run(worker.InterruptCh()); err != nil {
			log.Fatalln("News Sync Worker 启动失败", err)
		}
	}()
}
