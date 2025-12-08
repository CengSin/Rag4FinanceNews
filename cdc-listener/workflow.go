package cdc_listener

import (
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"time"
)

func SyncDataWorkflow(ctx workflow.Context, event CDCEvent) error {
	// 设置重试策略
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval: time.Second * 1,
			MaximumInterval: time.Second * 10,
			MaximumAttempts: 3, // 最多重试3次（生产环境可以设大点）
		},
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	// 2. 初始化 Activities
	var a *QdrantActivities

	// 3. 执行 Activity
	// Temporal 会记录这一步的状态
	err := workflow.ExecuteActivity(ctx, a.SyncToQdrant, event).Get(ctx, nil)
	if err != nil {
		return err
	}
	return nil
}
