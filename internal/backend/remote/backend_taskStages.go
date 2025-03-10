// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package remote

import (
	"context"
	"fmt"

	tfe "github.com/hashicorp/go-tfe"
)

type taskStages map[tfe.Stage]*tfe.TaskStage

const (
	taskStageBackoffMin = 4000.0
	taskStageBackoffMax = 12000.0
)

// waitTaskStage waits for a task stage to complete, only informs the caller if the stage has failed in some way.
func (b *Remote) waitTaskStage(stopCtx, cancelCtx context.Context, r *tfe.Run, stageID string) error {
	ctx := &IntegrationContext{
		StopContext:   stopCtx,
		CancelContext: cancelCtx,
	}
	return ctx.Poll(taskStageBackoffMin, taskStageBackoffMax, func(i int) (bool, error) {
		options := tfe.TaskStageReadOptions{
			Include: []tfe.TaskStageIncludeOpt{tfe.TaskStageTaskResults, tfe.PolicyEvaluationsTaskResults},
		}
		stage, err := b.client.TaskStages.Read(ctx.StopContext, stageID, &options)
		if err != nil {
			return false, generalError("Failed to retrieve task stage", err)
		}

		switch stage.Status {
		case tfe.TaskStagePending:
			// Waiting for it to start
			return true, nil
		case tfe.TaskStageRunning:
			// not a terminal status so we continue to poll
			return true, nil
		case tfe.TaskStagePassed:
			return false, nil
		case tfe.TaskStageCanceled, tfe.TaskStageErrored, tfe.TaskStageFailed:
			return false, fmt.Errorf("Task Stage '%s': %s.", stage.ID, stage.Status)
		case tfe.TaskStageAwaitingOverride:
			return false, fmt.Errorf("Task Stage '%s' awaiting override.", stage.ID)
		case tfe.TaskStageUnreachable:
			return false, nil
		default:
			return false, fmt.Errorf("Task stage '%s' has invalid status: %s", stage.ID, stage.Status)
		}
	})
}
