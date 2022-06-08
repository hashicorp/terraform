package cloud

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/go-tfe"
)

type taskResultSummary struct {
	unreachable     bool
	pending         int
	failed          int
	failedMandatory int
	passed          int
}

type taskStageReadFunc func(b *Cloud, stopCtx context.Context) (*tfe.TaskStage, error)

func summarizeTaskResults(taskResults []*tfe.TaskResult) *taskResultSummary {
	var pendingCount, errCount, errMandatoryCount, passedCount int
	for _, task := range taskResults {
		if task.Status == "unreachable" {
			return &taskResultSummary{
				unreachable: true,
			}
		} else if task.Status == "running" || task.Status == "pending" {
			pendingCount++
		} else if task.Status == "passed" {
			passedCount++
		} else {
			// Everything else is a failure
			errCount++
			if task.WorkspaceTaskEnforcementLevel == "mandatory" {
				errMandatoryCount++
			}
		}
	}

	return &taskResultSummary{
		unreachable:     false,
		pending:         pendingCount,
		failed:          errCount,
		failedMandatory: errMandatoryCount,
		passed:          passedCount,
	}
}

func (b *Cloud) runTasksWithTaskResults(context *IntegrationContext, output IntegrationOutputWriter, fetchTaskStage taskStageReadFunc) error {
	return context.Poll(func(i int) (bool, error) {
		stage, err := fetchTaskStage(b, context.StopContext)

		if err != nil {
			return false, generalError("Failed to retrieve pre-apply task stage", err)
		}

		summary := summarizeTaskResults(stage.TaskResults)

		if summary.unreachable {
			output.Output("Skipping task results.")
			output.End()
			return false, nil
		}

		if summary.pending > 0 {
			pendingMessage := "%d tasks still pending, %d passed, %d failed ... "
			message := fmt.Sprintf(pendingMessage, summary.pending, summary.passed, summary.failed)

			if i%4 == 0 {
				if i > 0 {
					output.OutputElapsed(message, len(pendingMessage)) // Up to 2 digits are allowed by the max message allocation
				}
			}
			return true, nil
		}

		// No more tasks pending/running. Print all the results.

		// Track the first task name that is a mandatory enforcement level breach.
		var firstMandatoryTaskFailed *string = nil

		if i == 0 {
			output.Output(fmt.Sprintf("All tasks completed! %d passed, %d failed", summary.passed, summary.failed))
		} else {
			output.OutputElapsed(fmt.Sprintf("All tasks completed! %d passed, %d failed", summary.passed, summary.failed), 50)
		}

		output.Output("")

		for _, t := range stage.TaskResults {
			capitalizedStatus := string(t.Status)
			capitalizedStatus = strings.ToUpper(capitalizedStatus[:1]) + capitalizedStatus[1:]

			status := "[green]" + capitalizedStatus
			if t.Status != "passed" {
				level := string(t.WorkspaceTaskEnforcementLevel)
				level = strings.ToUpper(level[:1]) + level[1:]
				status = fmt.Sprintf("[red]%s (%s)", capitalizedStatus, level)

				if t.WorkspaceTaskEnforcementLevel == "mandatory" && firstMandatoryTaskFailed == nil {
					firstMandatoryTaskFailed = &t.TaskName
				}
			}

			title := fmt.Sprintf(`%s ⸺   %s`, t.TaskName, status)
			output.SubOutput(title)

			if len(t.Message) > 0 {
				output.SubOutput(fmt.Sprintf("[dim]%s", t.Message))
			}
			if len(t.URL) > 0 {
				output.SubOutput(fmt.Sprintf("[dim]Details: %s", t.URL))
			}
			output.SubOutput("")
		}

		// If a mandatory enforcement level is breached, return an error.
		var taskErr error = nil
		var overall string = "[green]Passed"
		if firstMandatoryTaskFailed != nil {
			overall = "[red]Failed"
			if summary.failedMandatory > 1 {
				taskErr = fmt.Errorf("the run failed because %d mandatory tasks are required to succeed", summary.failedMandatory)
			} else {
				taskErr = fmt.Errorf("the run failed because the run task, %s, is required to succeed", *firstMandatoryTaskFailed)
			}
		} else if summary.failed > 0 { // we have failures but none of them mandatory
			overall = "[green]Passed with advisory failures"
		}

		output.SubOutput("")
		output.SubOutput("[bold]Overall Result: " + overall)

		output.End()

		return false, taskErr
	})
}

func (b *Cloud) runTasks(ctx *IntegrationContext, output IntegrationOutputWriter, stageID string) error {
	return b.runTasksWithTaskResults(ctx, output, func(b *Cloud, stopCtx context.Context) (*tfe.TaskStage, error) {
		options := tfe.TaskStageReadOptions{
			Include: []tfe.TaskStageIncludeOpt{tfe.TaskStageTaskResults},
		}

		return b.client.TaskStages.Read(ctx.StopContext, stageID, &options)
	})
}
