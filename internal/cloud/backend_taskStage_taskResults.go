// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
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

type taskResultSummarizer struct {
	finished bool
	cloud    *Cloud
	counter  int
}

func newTaskResultSummarizer(b *Cloud, ts *tfe.TaskStage) taskStageSummarizer {
	if len(ts.TaskResults) == 0 {
		return nil
	}
	return &taskResultSummarizer{
		finished: false,
		cloud:    b,
	}
}

func (trs *taskResultSummarizer) Summarize(context *IntegrationContext, output IntegrationOutputWriter, ts *tfe.TaskStage) (bool, *string, error) {
	if trs.finished {
		return false, nil, nil
	}
	trs.counter++

	counts := summarizeTaskResults(ts.TaskResults)

	if counts.pending != 0 {
		pendingMessage := "%d tasks still pending, %d passed, %d failed ... "
		message := fmt.Sprintf(pendingMessage, counts.pending, counts.passed, counts.failed)
		return true, &message, nil
	}
	if counts.unreachable {
		output.Output("Skipping task results.")
		output.End()
		return false, nil, nil
	}

	// Print out the summary
	trs.runTasksWithTaskResults(output, ts.TaskResults, counts)

	// Mark as finished
	trs.finished = true

	return false, nil, nil
}

func summarizeTaskResults(taskResults []*tfe.TaskResult) *taskResultSummary {
	var pendingCount, errCount, errMandatoryCount, passedCount int
	for _, task := range taskResults {
		if task.Status == tfe.TaskUnreachable {
			return &taskResultSummary{
				unreachable: true,
			}
		} else if task.Status == tfe.TaskRunning || task.Status == tfe.TaskPending {
			pendingCount++
		} else if task.Status == tfe.TaskPassed {
			passedCount++
		} else {
			// Everything else is a failure
			errCount++
			if task.WorkspaceTaskEnforcementLevel == tfe.Mandatory {
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

func (trs *taskResultSummarizer) runTasksWithTaskResults(output IntegrationOutputWriter, taskResults []*tfe.TaskResult, count *taskResultSummary) {
	// Track the first task name that is a mandatory enforcement level breach.
	var firstMandatoryTaskFailed *string = nil

	if trs.counter == 0 {
		output.Output(fmt.Sprintf("All tasks completed! %d passed, %d failed", count.passed, count.failed))
	} else {
		output.OutputElapsed(fmt.Sprintf("All tasks completed! %d passed, %d failed", count.passed, count.failed), 50)
	}

	output.Output("")

	for _, t := range taskResults {
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
	var overall string = "[green]Passed"
	if firstMandatoryTaskFailed != nil {
		overall = "[red]Failed"
		if count.failedMandatory > 1 {
			output.Output(fmt.Sprintf("[reset][bold][red]Error:[reset][bold]the run failed because %d mandatory tasks are required to succeed", count.failedMandatory))
		} else {
			output.Output(fmt.Sprintf("[reset][bold][red]Error: [reset][bold]the run failed because the run task, %s, is required to succeed", *firstMandatoryTaskFailed))
		}
	} else if count.failed > 0 { // we have failures but none of them mandatory
		overall = "[green]Passed with advisory failures"
	}

	output.SubOutput("")
	output.SubOutput("[bold]Overall Result: " + overall)

	output.End()
}
