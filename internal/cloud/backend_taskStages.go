package cloud

import (
	"context"
	"fmt"
	"strings"

	tfe "github.com/hashicorp/go-tfe"
)

const (
	taskStageBackoffMin = 4000.0
	taskStageBackoffMax = 12000.0
)

type taskStages map[tfe.Stage]*tfe.TaskStage

type taskStageSummarizer interface {
	// Summarize takes an IntegrationContext, IntegrationOutputWriter for
	// writing output and a pointer to a tfe.TaskStage object as arguments.
	// This function summarizes and outputs the results of the task stage.
	// It returns a boolean which signifies whether we should continue polling
	// for results, an optional message string to print while it is polling
	// and an error if any.
	Summarize(*IntegrationContext, IntegrationOutputWriter, *tfe.TaskStage) (bool, *string, error)
}

func (b *Cloud) runTaskStages(ctx context.Context, client *tfe.Client, runId string) (taskStages, error) {
	taskStages := make(taskStages, 0)
	result, err := client.Runs.ReadWithOptions(ctx, runId, &tfe.RunReadOptions{
		Include: []tfe.RunIncludeOpt{tfe.RunTaskStages},
	})
	if err == nil {
		for _, t := range result.TaskStages {
			if t != nil {
				taskStages[t.Stage] = t
			}
		}
	} else {
		// This error would be expected for older versions of TFE that do not allow
		// fetching task_stages.
		if !strings.HasSuffix(err.Error(), "Invalid include parameter") {
			return taskStages, generalError("Failed to retrieve run", err)
		}
	}

	return taskStages, nil
}

func (b *Cloud) getTaskStageWithAllOptions(ctx *IntegrationContext, stageID string) (*tfe.TaskStage, error) {
	options := tfe.TaskStageReadOptions{
		Include: []tfe.TaskStageIncludeOpt{tfe.TaskStageTaskResults},
	}
	stage, err := b.client.TaskStages.Read(ctx.StopContext, stageID, &options)
	if err != nil {
		return nil, generalError("Failed to retrieve task stage", err)
	} else {
		return stage, nil
	}
}

func (b *Cloud) runTaskStage(ctx *IntegrationContext, output IntegrationOutputWriter, stageID string) error {
	var errs multiErrors

	// Create our summarizers
	summarizers := make([]taskStageSummarizer, 0)
	ts, err := b.getTaskStageWithAllOptions(ctx, stageID)
	if err != nil {
		return err
	}
	if s := newTaskResultSummarizer(b, ts); s != nil {
		summarizers = append(summarizers, s)
	}

	return ctx.Poll(taskStageBackoffMin, taskStageBackoffMax, func(i int) (bool, error) {
		options := tfe.TaskStageReadOptions{
			Include: []tfe.TaskStageIncludeOpt{tfe.TaskStageTaskResults},
		}
		stage, err := b.client.TaskStages.Read(ctx.StopContext, stageID, &options)
		if err != nil {
			return false, generalError("Failed to retrieve task stage", err)
		}

		switch stage.Status {
		case tfe.TaskStagePending:
			// Waiting for it to start
			return true, nil
		// Note: Terminal statuses need to print out one last time just in case
		case tfe.TaskStageRunning, tfe.TaskStagePassed, tfe.TaskStageCanceled, tfe.TaskStageErrored, tfe.TaskStageFailed:
			for _, s := range summarizers {
				cont, msg, err := s.Summarize(ctx, output, stage)
				if err != nil {
					errs.Append(err)
					break
				}

				if !cont {
					continue
				}

				// cont is true and we must continue to poll
				if msg != nil {
					output.OutputElapsed(*msg, len(*msg)) // Up to 2 digits are allowed by the max message allocation
				}
				return true, nil
			}
		case tfe.TaskStageUnreachable:
			return false, nil
		default:
			return false, fmt.Errorf("Invalid Task stage status: %s ", stage.Status)
		}

		if len(errs) > 0 {
			return false, errs.Err()
		}
		return false, nil
	})
}
