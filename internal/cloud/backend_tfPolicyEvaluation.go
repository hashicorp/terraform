// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
	"context"
	"fmt"
	"strings"

	tfe "github.com/hashicorp/go-tfe"
)

// renderTFPolicyEvaluations fetches the Terraform policy evaluation outcomes for
// a finished run and prints a per-stage summary. A run can hold one evaluation
// per stage, so stages limits rendering to the given stages (empty means all).
// When no outcomes are available it falls back to the previous success message.
func (b *Cloud) renderTFPolicyEvaluations(stopCtx context.Context, r *tfe.Run, localPoliciesConfigured bool, stages ...tfe.TFPolicyEvaluationStageType) error {
	if b.renderer == nil {
		return nil
	}

	stageFilter := make(map[tfe.TFPolicyEvaluationStageType]bool, len(stages))
	for _, s := range stages {
		stageFilter[s] = true
	}

	// Re-read the run with the policy evaluations included.
	run, err := b.client.Runs.ReadWithOptions(stopCtx, r.ID, &tfe.RunReadOptions{
		Include: []tfe.RunIncludeOpt{tfe.RunTFPolicyEvaluation},
	})
	if err != nil {
		// Older TFE versions don't know this include; nothing to render.
		if strings.HasSuffix(err.Error(), "Invalid include parameter") {
			b.renderTFPolicyEvalFallback(localPoliciesConfigured)
			return nil
		}
		return b.generalError("Failed to retrieve Terraform policy evaluations", err)
	}

	if len(run.TFPolicyEvaluations) == 0 {
		b.renderTFPolicyEvalFallback(localPoliciesConfigured)
		return nil
	}

	// Fetch the policy-set outcomes for each evaluation we want to show.
	var rendered []tfPolicyStageOutcomes
	for _, eval := range run.TFPolicyEvaluations {
		if eval.Status == tfe.TFPolicyEvaluationStatusUnreachable {
			continue
		}
		if len(stageFilter) > 0 && !stageFilter[eval.StageType] {
			continue
		}

		list, err := b.client.TFPolicyEvaluationOutcomes.List(stopCtx, eval.ID, nil)
		if err != nil {
			return b.generalError("Failed to retrieve Terraform policy outcomes", err)
		}
		rendered = append(rendered, tfPolicyStageOutcomes{eval: eval, sets: list.Items})
	}

	if len(rendered) == 0 {
		b.renderTFPolicyEvalFallback(localPoliciesConfigured)
		return nil
	}

	b.writeTFPolicyEvaluations(rendered)
	return nil
}

// tfPolicyStageOutcomes pairs one stage's evaluation with its policy-set outcomes.
type tfPolicyStageOutcomes struct {
	eval *tfe.TFPolicyEvaluation
	sets []*tfe.TFPolicySetOutcome
}

// writeTFPolicyEvaluations renders the fetched per-stage outcomes. It is split
// from the fetch logic so the formatting can be tested without a live API.
func (b *Cloud) writeTFPolicyEvaluations(rendered []tfPolicyStageOutcomes) {
	overallFailed := false
	total := 0
	for _, stage := range rendered {
		if stage.eval.Status != tfe.TFPolicyEvaluationStatusPassed {
			overallFailed = true
		}
		total += tfPolicyEvaluationCount(stage.eval.ResultCount)
	}

	// If the API didn't populate counts, count the outcomes ourselves.
	if total == 0 {
		for _, stage := range rendered {
			for _, set := range stage.sets {
				total += len(set.Outcomes)
			}
		}
	}

	print := func(format string, args ...any) {
		b.renderer.Streams.Println(b.Colorize().Color(fmt.Sprintf(format, args...)))
	}

	print("\n------------------------------------------------------------------------\n")
	print("[bold]Terraform policy Evaluations\n")

	if overallFailed {
		print("[bold]%c%c Overall result : [red]FAILED", Arrow, Arrow)
		print("[dim]This means that one or more Terraform policies failed.")
	} else {
		print("[bold]%c%c Overall result : [green]PASSED", Arrow, Arrow)
		print("[dim]This means that all Terraform policies passed.")
	}
	print("\n%d policies evaluated\n", total)

	for _, stage := range rendered {
		stageResult := "[green]PASSED"
		if stage.eval.Status != tfe.TFPolicyEvaluationStatusPassed {
			stageResult = "[red]" + strings.ToUpper(string(stage.eval.Status))
		}
		print("[bold]%s stage: %s", tfPolicyStageLabel(stage.eval.StageType), stageResult)
		print("[dim]%s", tfPolicyStageCounts(stage.sets))

		for i, set := range stage.sets {
			print("  %c Policy set %d: [bold]%s", Arrow, i+1, set.PolicySetName)
			for _, outcome := range set.Outcomes {
				if outcome.FileName != "" {
					print("    %c Policy name: [bold]%q[reset] in %s", Arrow, outcome.PolicyName, outcome.FileName)
				} else {
					print("    %c Policy name: [bold]%q", Arrow, outcome.PolicyName)
				}
				print("      %c Result: %s", Arrow, tfPolicyOutcomeResult(outcome))
			}
		}
	}
}

// renderTFPolicyEvalFallback prints the previous success message, but only for
// runs that used local --policies paths so other runs stay silent.
func (b *Cloud) renderTFPolicyEvalFallback(localPoliciesConfigured bool) {
	if b.renderer == nil || !localPoliciesConfigured {
		return
	}
	b.renderer.Streams.Println(b.Colorize().Color(tfpolicyEvalSuccessful))
}

// tfPolicyEvaluationCount returns the total policy count for a result count.
func tfPolicyEvaluationCount(rc *tfe.TFPolicyEvaluationResultCount) int {
	if rc == nil {
		return 0
	}
	return rc.AdvisoryFailed + rc.MandatoryFailed + rc.Errored + rc.Passed + rc.Unknown
}

// tfPolicyStageCounts returns a breakdown line for a stage, e.g.
// "29 total, 18 passed, 2 failed, 7 unknown". Advisory and errored are shown
// only when present.
func tfPolicyStageCounts(sets []*tfe.TFPolicySetOutcome) string {
	var total, passed, failed, advisory, unknown, errored int
	for _, set := range sets {
		for _, outcome := range set.Outcomes {
			total++
			switch strings.ToLower(outcome.Status) {
			case "passed":
				passed++
			case "failed":
				if outcome.EnforcementLevel == tfe.TFPolicyEvaluationOutcomeEnforcementLevelAdvisory {
					advisory++
				} else {
					failed++
				}
			case "unknown":
				unknown++
			case "errored":
				errored++
			}
		}
	}

	parts := []string{
		fmt.Sprintf("%d total", total),
		fmt.Sprintf("%d passed", passed),
		fmt.Sprintf("%d failed", failed),
	}
	if advisory > 0 {
		parts = append(parts, fmt.Sprintf("%d advisory", advisory))
	}
	parts = append(parts, fmt.Sprintf("%d unknown", unknown))
	if errored > 0 {
		parts = append(parts, fmt.Sprintf("%d errored", errored))
	}
	return strings.Join(parts, ", ")
}

// tfPolicyStageLabel returns a human-readable label for a stage.
func tfPolicyStageLabel(stage tfe.TFPolicyEvaluationStageType) string {
	switch stage {
	case tfe.TFPolicyEvaluationStageTypeInit:
		return "Pre-plan"
	case tfe.TFPolicyEvaluationStageTypePlan:
		return "Plan"
	case tfe.TFPolicyEvaluationStageTypeApply:
		return "Apply"
	default:
		return string(stage)
	}
}

// tfPolicyOutcomeResult returns the colorized result label for an outcome.
func tfPolicyOutcomeResult(outcome *tfe.TFPolicySetPolicyOutcome) string {
	switch strings.ToLower(outcome.Status) {
	case "passed":
		return "[green][bold]Passed"
	case "failed":
		if outcome.EnforcementLevel == tfe.TFPolicyEvaluationOutcomeEnforcementLevelAdvisory {
			return "[blue][bold]Advisory"
		}
		return "[red][bold]Failed"
	case "unknown":
		return "[yellow][bold]Unknown"
	case "errored":
		return "[red][bold]Errored"
	default:
		return "[bold]" + outcome.Status
	}
}
