package main

import (
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"log"
	cdc_listener "rag4financenew/cdc-listener"
)

func main() {
	c, err := client.Dial(client.Options{
		Namespace: cdc_listener.NameSpace,
	})
	if err != nil {
		log.Fatalln("Unable to connect to temporal server", err)
	}
	defer c.Close()

	// 2. 创建 Worker
	// 注意：QueueName 必须和 Listener 里发任务时的 QueueName 一致
	w := worker.New(c, cdc_listener.QueueName, worker.Options{})

	// 3. 注册 Workflow 和 Activity
	w.RegisterWorkflow(cdc_listener.SyncDataWorkflow)
	w.RegisterActivity(&cdc_listener.QdrantActivities{})

	// 4. 启动 Worker，开始接单！
	log.Println("Worker 已启动，等待任务中...")
	if err = w.Run(worker.InterruptCh()); err != nil {
		log.Fatalln("Worker 停止", err)
	}
}
