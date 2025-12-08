package api

import (
	"fmt"
	"github.com/labstack/echo/v4"
	enumspb "go.temporal.io/api/enums/v1"
	temporalClient "go.temporal.io/sdk/client"
	"net/http"
	"rag4financenew/client"
	rWorkflow "rag4financenew/workflow"
)

func ChatWithLLM(c echo.Context) error {
	ctx := c.Request().Context()
	var req rWorkflow.ChatReq
	if err := c.Bind(&req); err != nil {
		return err
	}

	startWorkflowOp := client.Temporal.NewWithStartWorkflowOperation(temporalClient.StartWorkflowOptions{
		ID:                       req.GetWorkflowID(),
		TaskQueue:                TaskQueueName,
		WorkflowIDConflictPolicy: enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
	}, rWorkflow.ChatSessionWorkflow, req)

	updateOptions := temporalClient.UpdateWorkflowOptions{
		UpdateName:   "chat_message",
		WaitForStage: temporalClient.WorkflowUpdateStageCompleted,
		Args:         []interface{}{req},
	}

	handle, err := client.Temporal.UpdateWithStartWorkflow(ctx, temporalClient.UpdateWithStartWorkflowOptions{
		StartWorkflowOperation: startWorkflowOp,
		UpdateOptions:          updateOptions,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("UpdateWithStartWorkflow err %s", err.Error()))
	}

	var resp rWorkflow.ChatResp
	if err = handle.Get(ctx, &resp); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Get err %s", err.Error()))
	}

	return c.JSON(http.StatusOK, resp)
}
