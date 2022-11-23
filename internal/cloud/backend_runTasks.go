package cloud

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/internal/terraform"
)

type taskResultSummary struct {
	unreachable     bool
	pending         int
	failed          int
	failedMandatory int
	passed          int
}

type policyEvaluationSummary struct {
	unreachable bool
	pending     int
	failed      int
	passed      int
}

type Symbol rune

const (
	Tick          Symbol = '\u2713'
	Cross         Symbol = '\u00d7'
	Warning       Symbol = '\u24be'
	Arrow         Symbol = '\u2192'
	DownwardArrow Symbol = '\u21b3'
)

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

func summarizePolicyEvaluationResults(policyEvaluations []*tfe.PolicyEvaluation) *policyEvaluationSummary {
	var pendingCount, errCount, passedCount int
	for _, policyEvaluation := range policyEvaluations {
		if policyEvaluation.Status == "unreachable" {
			return &policyEvaluationSummary{
				unreachable: true,
			}
		} else if policyEvaluation.Status == "running" || policyEvaluation.Status == "pending" || policyEvaluation.Status == "queued" {
			pendingCount++
		} else if policyEvaluation.Status == "passed" {
			passedCount++
		} else {
			// Everything else is a failure
			errCount++
		}
	}

	return &policyEvaluationSummary{
		unreachable: false,
		pending:     pendingCount,
		failed:      errCount,
		passed:      passedCount,
	}
}

func (b *Cloud) runTasksWithTaskResults(context *IntegrationContext, output IntegrationOutputWriter, fetchTaskStage taskStageReadFunc) error {
	return context.Poll(func(i int) (bool, error) {
		stage, err := fetchTaskStage(b, context.StopContext)

		if err != nil {
			return false, generalError("Failed to retrieve task stage", err)
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

func (b *Cloud) runTaskStage(ctx *IntegrationContext, output IntegrationOutputWriter, stageID string) error {
	err1 := b.runTasksWithTaskResults(ctx, output, func(b *Cloud, stopCtx context.Context) (*tfe.TaskStage, error) {
		options := tfe.TaskStageReadOptions{
			Include: []tfe.TaskStageIncludeOpt{tfe.TaskStageTaskResults, tfe.PolicyEvaluationsTaskResults},
		}

		return b.client.TaskStages.Read(ctx.StopContext, stageID, &options)
	})
	// call to runTaskwithPolicyEvaluation
	err2 := b.runTasksWithPolicyEvaluation(ctx, output, func(b *Cloud, stopCtx context.Context) (*tfe.TaskStage, error) {
		options := tfe.TaskStageReadOptions{
			Include: []tfe.TaskStageIncludeOpt{tfe.TaskStageTaskResults, tfe.PolicyEvaluationsTaskResults},
		}

		return b.client.TaskStages.Read(ctx.StopContext, stageID, &options)
	})
	//TODO: clean this up
	if err1 != nil && err2 != nil {
		return fmt.Errorf("error running task stage: %v %v", err1, err2)
	} else if err1 != nil {
		return err1
	} else if err2 != nil {
		return err2
	}
	return nil

}

func (b *Cloud) runTasksWithPolicyEvaluation(context *IntegrationContext, output IntegrationOutputWriter, fetchTaskStage taskStageReadFunc) error {
	return context.Poll(func(i int) (bool, error) {
		stage, err := fetchTaskStage(b, context.StopContext)

		if err != nil {
			return false, generalError("Failed to retrieve task stage", err)
		}

		if len(stage.PolicyEvaluations) == 0 {
			return false, nil
		}

		summary := summarizePolicyEvaluationResults(stage.PolicyEvaluations)
		if summary.unreachable {
			output.Output("Skipping policy evaluation.")
			output.End()
			return false, nil
		}

		if summary.pending > 0 {
			pendingMessage := "Evaluating ... "

			if i%4 == 0 {
				if i > 0 {
					output.OutputElapsed(pendingMessage, len(pendingMessage))
				}
			}
			return true, nil
		}

		// No more policy evaluations pending/running. Print all the results.
		output.Output("\n------------------------------------------------------------------------\n")
		output.Output("[bold]OPA Policy Evaluation\n")

		var result, message string
		// Currently only one policy evaluation supported : OPA
		if stage.PolicyEvaluations[0].Status == tfe.PolicyEvaluationPassed {
			message = "[dim] This result means that all OPA policies passed and the protected behaviour is allowed"
			result = fmt.Sprintf("[green]%s", strings.ToUpper(string(tfe.PolicyEvaluationPassed)))
			if stage.PolicyEvaluations[0].ResultCount.AdvisoryFailed > 0 {
				result += " (with advisory)"
			}
		} else {
			message = "[dim] This result means that one or more OPA policies failed. More than likely, this was due to the discovery of violations by the main rule and other sub rules"
			result = fmt.Sprintf("[red]%s", strings.ToUpper(string(tfe.PolicyEvaluationFailed)))
		}

		output.Output(fmt.Sprintf("[bold]%c%c Overall Result: %s", Arrow, Arrow, result))

		output.Output(message)

		total := getPolicyCount(stage.PolicyEvaluations[0].ResultCount)

		output.Output(fmt.Sprintf("%d policies evaluated\n", total))

		for _, policyEvaluation := range stage.PolicyEvaluations {
			policyOutcomes, err := b.client.PolicySetOutcomes.List(context.StopContext, policyEvaluation.ID, nil)
			if err != nil {
				return false, err
			}

			for i, out := range policyOutcomes.Items {
				output.Output(fmt.Sprintf("%c Policy set %d: [bold]%s (%d)", Arrow, i+1, out.PolicySetName, len(out.Outcomes)))
				for _, outcome := range out.Outcomes {
					output.Output(fmt.Sprintf("  %c Policy name: [bold]%s", DownwardArrow, outcome.PolicyName))
					switch outcome.Status {
					case "passed":
						output.Output(fmt.Sprintf("     | [green][bold]%c Passed", Tick))
					case "failed":
						if outcome.EnforcementLevel == "advisory" {
							output.Output(fmt.Sprintf("     | [blue][bold]%c Advisory", Warning))
						} else {
							output.Output(fmt.Sprintf("     | [red][bold]%c Failed", Cross))
						}
					}
					if outcome.Description != "" {
						output.Output(fmt.Sprintf("     | [dim]%s", outcome.Description))
					} else {
						output.Output("     | [dim]No description available")
					}
				}
			}
		}

		stage, err = fetchTaskStage(b, context.StopContext)
		if err != nil {
			return false, generalError("Failed to retrieve task stage", err)
		}

		if stage.Status == tfe.TaskStageAwaitingOverride {
			opts := &terraform.InputOpts{
				Id:          fmt.Sprintf("%c%c [bold]Override", Arrow, Arrow),
				Query:       "\nDo you want to override the failed policy check?",
				Description: "Only 'override' will be accepted to override.",
			}
			runUrl := fmt.Sprintf(taskStageHeader, b.hostname, b.organization, context.Op.Workspace, context.Run.ID)
			err = b.confirm(context.StopContext, context.Op, opts, context.Run, "override")
			if err != nil && err != errRunOverridden {
				return false, fmt.Errorf(
					fmt.Sprintf("Failed to override: %s\n%s\n", err.Error(), runUrl),
				)
			}

			if err != errRunOverridden {
				if _, err = b.client.TaskStages.Override(context.StopContext, stage.ID, tfe.TaskStageOverrideOptions{}); err != nil {
					return false, generalError(fmt.Sprintf("Failed to override policy check.\n%s", runUrl), err)
				}
			} else {
				output.Output(fmt.Sprintf("The run needs to be manually overridden or discarded.\n%s\n", runUrl))
			}
		}

		return false, nil
	})
}

func getPolicyCount(resultCount *tfe.PolicyResultCount) int {
	return resultCount.AdvisoryFailed + resultCount.MandatoryFailed + resultCount.Errored + resultCount.Passed
}
