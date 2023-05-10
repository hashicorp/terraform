// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-tfe"
)

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

type policyEvaluationSummarizer struct {
	finished bool
	cloud    *Cloud
	counter  int
}

func newPolicyEvaluationSummarizer(b *Cloud, ts *tfe.TaskStage) taskStageSummarizer {
	if len(ts.PolicyEvaluations) == 0 {
		return nil
	}
	return &policyEvaluationSummarizer{
		finished: false,
		cloud:    b,
	}
}

func (pes *policyEvaluationSummarizer) Summarize(context *IntegrationContext, output IntegrationOutputWriter, ts *tfe.TaskStage) (bool, *string, error) {
	if pes.counter == 0 {
		output.Output("[bold]OPA Policy Evaluation\n")
		pes.counter++
	}

	if pes.finished {
		return false, nil, nil
	}

	counts := summarizePolicyEvaluationResults(ts.PolicyEvaluations)

	if counts.pending != 0 {
		pendingMessage := "Evaluating ... "
		return true, &pendingMessage, nil
	}

	if counts.unreachable {
		output.Output("Skipping policy evaluation.")
		output.End()
		return false, nil, nil
	}

	// Print out the summary
	if err := pes.taskStageWithPolicyEvaluation(context, output, ts.PolicyEvaluations); err != nil {
		return false, nil, err
	}
	// Mark as finished
	pes.finished = true

	return false, nil, nil
}

func summarizePolicyEvaluationResults(policyEvaluations []*tfe.PolicyEvaluation) *policyEvaluationSummary {
	var pendingCount, errCount, passedCount int
	for _, policyEvaluation := range policyEvaluations {
		switch policyEvaluation.Status {
		case "unreachable":
			return &policyEvaluationSummary{
				unreachable: true,
			}
		case "running", "pending", "queued":
			pendingCount++
		case "passed":
			passedCount++
		default:
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

func (pes *policyEvaluationSummarizer) taskStageWithPolicyEvaluation(context *IntegrationContext, output IntegrationOutputWriter, policyEvaluation []*tfe.PolicyEvaluation) error {
	var result, message string
	// Currently only one policy evaluation supported : OPA
	for _, polEvaluation := range policyEvaluation {
		if polEvaluation.Status == tfe.PolicyEvaluationPassed {
			message = "[dim] This result means that all OPA policies passed and the protected behavior is allowed"
			result = fmt.Sprintf("[green]%s", strings.ToUpper(string(tfe.PolicyEvaluationPassed)))
			if polEvaluation.ResultCount.AdvisoryFailed > 0 {
				result += " (with advisory)"
			}
		} else {
			message = "[dim] This result means that one or more OPA policies failed. More than likely, this was due to the discovery of violations by the main rule and other sub rules"
			result = fmt.Sprintf("[red]%s", strings.ToUpper(string(tfe.PolicyEvaluationFailed)))
		}

		output.Output(fmt.Sprintf("[bold]%c%c Overall Result: %s", Arrow, Arrow, result))

		output.Output(message)

		total := getPolicyCount(polEvaluation.ResultCount)

		output.Output(fmt.Sprintf("%d policies evaluated\n", total))

		policyOutcomes, err := pes.cloud.client.PolicySetOutcomes.List(context.StopContext, polEvaluation.ID, nil)
		if err != nil {
			return err
		}

		for i, out := range policyOutcomes.Items {
			output.Output(fmt.Sprintf("%c Policy set %d: [bold]%s (%d)", Arrow, i+1, out.PolicySetName, len(out.Outcomes)))
			for _, outcome := range out.Outcomes {
				output.Output(fmt.Sprintf("  %c Policy name: [bold]%s", DownwardArrow, outcome.PolicyName))
				switch outcome.Status {
				case "passed":
					output.Output(fmt.Sprintf("     | [green][bold]%c Passed", Tick))
				case "failed":
					if outcome.EnforcementLevel == tfe.EnforcementAdvisory {
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
	return nil
}

func getPolicyCount(resultCount *tfe.PolicyResultCount) int {
	return resultCount.AdvisoryFailed + resultCount.MandatoryFailed + resultCount.Errored + resultCount.Passed
}
