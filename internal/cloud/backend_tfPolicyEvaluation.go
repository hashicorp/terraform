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
//
// Terraform-native policy is configured as workspace-attached policy sets and
// evaluated server-side, so HCP returns outcomes whenever a policy set applies,
// regardless of any local --policies paths. When HCP returns no outcomes there
// is nothing to render and the function is silent.
func (b *Cloud) renderTFPolicyEvaluations(stopCtx context.Context, r *tfe.Run, stages ...tfe.TFPolicyEvaluationStageType) error {
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
			return nil
		}
		return b.generalError("Failed to retrieve Terraform policy evaluations", err)
	}

	// No policy set attached to the workspace (or none applied) → nothing to render.
	if len(run.TFPolicyEvaluations) == 0 {
		return nil
	}

	// Fetch the policy-set outcomes for each evaluation we want to show.
	var rendered []tfPolicyStageOutcomes
	for _, eval := range run.TFPolicyEvaluations {
		// "unreachable" means this stage's evaluation never ran because an
		// earlier stage errored or was canceled, so it has no outcomes to show.
		if eval.Status == tfe.TFPolicyEvaluationStatusUnreachable {
			continue
		}
		if len(stageFilter) > 0 && !stageFilter[eval.StageType] {
			continue
		}

		sets, err := b.listTFPolicyOutcomes(stopCtx, eval.ID)
		if err != nil {
			return b.generalError("Failed to retrieve Terraform policy outcomes", err)
		}
		rendered = append(rendered, tfPolicyStageOutcomes{eval: eval, sets: sets})
	}

	if len(rendered) == 0 {
		return nil
	}

	b.writeTFPolicyEvaluations(rendered)
	return nil
}

// listTFPolicyOutcomes fetches all policy-set outcomes for an evaluation,
// following pagination so large policy sets aren't truncated to the first page.
func (b *Cloud) listTFPolicyOutcomes(ctx context.Context, evalID string) ([]*tfe.TFPolicySetOutcome, error) {
	var sets []*tfe.TFPolicySetOutcome
	opts := &tfe.TFPolicyEvaluationListOptions{}
	for {
		page, err := b.client.TFPolicyEvaluationOutcomes.List(ctx, evalID, opts)
		if err != nil {
			return nil, err
		}
		sets = append(sets, page.Items...)
		if page.Pagination == nil || page.CurrentPage >= page.TotalPages {
			break
		}
		opts.PageNumber = page.NextPage
	}
	return sets, nil
}

// tfPolicyStageOutcomes pairs one stage's evaluation with its policy-set outcomes.
type tfPolicyStageOutcomes struct {
	eval *tfe.TFPolicyEvaluation
	sets []*tfe.TFPolicySetOutcome
}

// writeTFPolicyEvaluations renders the fetched per-stage outcomes. It is split
// from the fetch logic so the formatting can be tested without a live API.
func (b *Cloud) writeTFPolicyEvaluations(rendered []tfPolicyStageOutcomes) {
	// Count the evaluated policies from the outcomes we render, so the total
	// always matches the per-stage breakdown (tfPolicyStageCounts) below and
	// stays consistent regardless of the server-side ResultCount aggregate.
	overallFailed := false
	total := 0
	for _, stage := range rendered {
		// TODO: non-failure statuses (overridden, canceled, awaiting_override)
		// are treated as failed for now. Override handling will be added later,
		// mirroring the Sentinel override flow, since overrides only apply when
		// a policy actually failed.
		if stage.eval.Status != tfe.TFPolicyEvaluationStatusPassed {
			overallFailed = true
		}
		for _, set := range stage.sets {
			total += len(set.Outcomes)
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
