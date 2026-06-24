// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
	"strings"
	"testing"

	"github.com/hashicorp/go-tfe"
)

func TestCloud_runTaskStageWithOPAPolicyEvaluation(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	integrationContext, writer := newMockIntegrationContext(b, t)

	cases := map[string]struct {
		taskStage       func() *tfe.TaskStage
		context         *IntegrationContext
		writer          *testIntegrationOutput
		expectedOutputs []string
		isError         bool
	}{
		"all-succeeded": {
			taskStage: func() *tfe.TaskStage {
				ts := &tfe.TaskStage{}
				ts.PolicyEvaluations = []*tfe.PolicyEvaluation{
					{ID: "pol-pass", ResultCount: &tfe.PolicyResultCount{Passed: 1}, Status: "passed", PolicyKind: "opa"},
				}
				return ts
			},
			writer:          writer,
			context:         integrationContext,
			expectedOutputs: []string{"│ [bold]OPA Policy Evaluation\n\n│ [bold]→→ Overall Result: [green]PASSED\n│ [dim] This result means that all OPA policies passed and the protected behavior is allowed\n│ 1 policies evaluated\n\n│ → Policy set 1: [bold]policy-set-that-passes (1)\n│   ↳ Policy name: [bold]policy-pass\n│      | [green][bold]✓ Passed\n│      | [dim]This policy will pass\n"},
			isError:         false,
		},
		"mandatory-failed": {
			taskStage: func() *tfe.TaskStage {
				ts := &tfe.TaskStage{}
				ts.PolicyEvaluations = []*tfe.PolicyEvaluation{
					{ID: "pol-fail", ResultCount: &tfe.PolicyResultCount{MandatoryFailed: 1}, Status: "failed", PolicyKind: "opa"},
				}
				return ts
			},
			writer:          writer,
			context:         integrationContext,
			expectedOutputs: []string{"│ [bold]→→ Overall Result: [red]FAILED\n│ [dim] This result means that one or more OPA policies failed. More than likely, this was due to the discovery of violations by the main rule and other sub rules\n│ 1 policies evaluated\n\n│ → Policy set 1: [bold]policy-set-that-fails (1)\n│   ↳ Policy name: [bold]policy-fail\n│      | [red][bold]× Failed\n│      | [dim]This policy will fail"},
			isError:         true,
		},
		"advisory-failed": {
			taskStage: func() *tfe.TaskStage {
				ts := &tfe.TaskStage{}
				ts.PolicyEvaluations = []*tfe.PolicyEvaluation{
					{ID: "adv-fail", ResultCount: &tfe.PolicyResultCount{AdvisoryFailed: 1}, Status: "failed", PolicyKind: "opa"},
				}
				return ts
			},
			writer:          writer,
			context:         integrationContext,
			expectedOutputs: []string{"│ [bold]OPA Policy Evaluation\n\n│ [bold]→→ Overall Result: [red]FAILED\n│ [dim] This result means that one or more OPA policies failed. More than likely, this was due to the discovery of violations by the main rule and other sub rules\n│ 1 policies evaluated\n\n│ → Policy set 1: [bold]policy-set-that-fails (1)\n│   ↳ Policy name: [bold]policy-fail\n│      | [blue][bold]Ⓘ Advisory\n│      | [dim]This policy will fail"},
			isError:         false,
		},
		"unreachable": {
			taskStage: func() *tfe.TaskStage {
				ts := &tfe.TaskStage{}
				ts.PolicyEvaluations = []*tfe.PolicyEvaluation{
					{ID: "adv-fail", ResultCount: &tfe.PolicyResultCount{Errored: 1}, Status: "unreachable", PolicyKind: "opa"},
				}
				return ts
			},
			writer:          writer,
			context:         integrationContext,
			expectedOutputs: []string{"Skipping policy evaluation."},
			isError:         false,
		},
		"pending-with-failed-task-stage": {
			taskStage: func() *tfe.TaskStage {
				ts := &tfe.TaskStage{Status: tfe.TaskStageFailed}
				ts.PolicyEvaluations = []*tfe.PolicyEvaluation{
					{ID: "pol-pending", ResultCount: &tfe.PolicyResultCount{}, Status: "pending", PolicyKind: "opa"},
				}
				return ts
			},
			writer:          writer,
			context:         integrationContext,
			expectedOutputs: []string{"Skipping policy evaluation."},
			isError:         false,
		},
		"pending-with-canceled-task-stage": {
			taskStage: func() *tfe.TaskStage {
				ts := &tfe.TaskStage{Status: tfe.TaskStageCanceled}
				ts.PolicyEvaluations = []*tfe.PolicyEvaluation{
					{ID: "pol-pending", ResultCount: &tfe.PolicyResultCount{}, Status: "pending", PolicyKind: "opa"},
				}
				return ts
			},
			writer:          writer,
			context:         integrationContext,
			expectedOutputs: []string{"Skipping policy evaluation."},
			isError:         false,
		},
		"pending-with-errored-task-stage": {
			taskStage: func() *tfe.TaskStage {
				ts := &tfe.TaskStage{Status: tfe.TaskStageErrored}
				ts.PolicyEvaluations = []*tfe.PolicyEvaluation{
					{ID: "pol-pending", ResultCount: &tfe.PolicyResultCount{}, Status: "pending", PolicyKind: "opa"},
				}
				return ts
			},
			writer:          writer,
			context:         integrationContext,
			expectedOutputs: []string{"Skipping policy evaluation."},
			isError:         false,
		},
		"mixed-pending-and-completed-with-failed-task-stage": {
			taskStage: func() *tfe.TaskStage {
				ts := &tfe.TaskStage{Status: tfe.TaskStageFailed}
				ts.PolicyEvaluations = []*tfe.PolicyEvaluation{
					{ID: "pol-pass", ResultCount: &tfe.PolicyResultCount{Passed: 1}, Status: "passed", PolicyKind: "opa"},
					{ID: "pol-pending", ResultCount: &tfe.PolicyResultCount{}, Status: "pending", PolicyKind: "opa"},
				}
				return ts
			},
			writer:  writer,
			context: integrationContext,
			expectedOutputs: []string{
				"OPA Policy Evaluation",
				"Overall Result: [green]PASSED",
				"Skipping 1 pending policy evaluation(s) because task stage is failed.",
			},
			isError: false,
		},
		"multiple-mixed-states-with-failed-task-stage": {
			taskStage: func() *tfe.TaskStage {
				ts := &tfe.TaskStage{Status: tfe.TaskStageFailed}
				ts.PolicyEvaluations = []*tfe.PolicyEvaluation{
					{ID: "pol-pass", ResultCount: &tfe.PolicyResultCount{Passed: 1}, Status: "passed", PolicyKind: "opa"},
					{ID: "pol-fail", ResultCount: &tfe.PolicyResultCount{MandatoryFailed: 1}, Status: "failed", PolicyKind: "opa"},
					{ID: "pol-pending", ResultCount: &tfe.PolicyResultCount{}, Status: "pending", PolicyKind: "opa"},
					{ID: "pol-pending-2", ResultCount: &tfe.PolicyResultCount{}, Status: "running", PolicyKind: "opa"},
				}
				return ts
			},
			writer:  writer,
			context: integrationContext,
			expectedOutputs: []string{
				"OPA Policy Evaluation",
				"Overall Result: [green]PASSED",
				"Overall Result: [red]FAILED",
				"Skipping 2 pending policy evaluation(s) because task stage is failed.",
			},
			isError: false,
		},
		"mixed-pending-and-completed-with-canceled-task-stage": {
			taskStage: func() *tfe.TaskStage {
				ts := &tfe.TaskStage{Status: tfe.TaskStageCanceled}
				ts.PolicyEvaluations = []*tfe.PolicyEvaluation{
					{ID: "pol-pass", ResultCount: &tfe.PolicyResultCount{Passed: 1}, Status: "passed", PolicyKind: "opa"},
					{ID: "pol-pending", ResultCount: &tfe.PolicyResultCount{}, Status: "pending", PolicyKind: "opa"},
				}
				return ts
			},
			writer:  writer,
			context: integrationContext,
			expectedOutputs: []string{
				"OPA Policy Evaluation",
				"Overall Result: [green]PASSED",
				"Skipping 1 pending policy evaluation(s) because task stage is canceled.",
			},
			isError: false,
		},
		"mixed-pending-and-completed-with-errored-task-stage": {
			taskStage: func() *tfe.TaskStage {
				ts := &tfe.TaskStage{Status: tfe.TaskStageErrored}
				ts.PolicyEvaluations = []*tfe.PolicyEvaluation{
					{ID: "pol-pass", ResultCount: &tfe.PolicyResultCount{Passed: 1}, Status: "passed", PolicyKind: "opa"},
					{ID: "pol-pending", ResultCount: &tfe.PolicyResultCount{}, Status: "pending", PolicyKind: "opa"},
				}
				return ts
			},
			writer:  writer,
			context: integrationContext,
			expectedOutputs: []string{
				"OPA Policy Evaluation",
				"Overall Result: [green]PASSED",
				"Skipping 1 pending policy evaluation(s) because task stage is errored.",
			},
			isError: false,
		},
	}

	for _, c := range cases {
		c.writer.output.Reset()
		trs := policyEvaluationSummarizer{
			cloud: b,
		}
		c.context.Poll(0, 0, func(i int) (bool, error) {
			cont, _, _ := trs.Summarize(c.context, c.writer, c.taskStage())
			if cont {
				return true, nil
			}

			output := c.writer.output.String()
			for _, expected := range c.expectedOutputs {
				if !strings.Contains(output, expected) {
					t.Fatalf("Expected output to contain '%s' but it was:\n\n%s", expected, output)
				}
			}
			return false, nil
		})
	}
}

func TestCloud_runTaskStageWithSentinelPolicyEvaluation(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	integrationContext, writer := newMockIntegrationContext(b, t)

	cases := map[string]struct {
		taskStage       func() *tfe.TaskStage
		context         *IntegrationContext
		writer          *testIntegrationOutput
		expectedOutputs []string
		isError         bool
	}{
		"all-succeeded": {
			taskStage: func() *tfe.TaskStage {
				ts := &tfe.TaskStage{}
				ts.PolicyEvaluations = []*tfe.PolicyEvaluation{
					{ID: "pol-pass", ResultCount: &tfe.PolicyResultCount{Passed: 1}, Status: "passed", PolicyKind: "sentinel"},
				}
				return ts
			},
			writer:          writer,
			context:         integrationContext,
			expectedOutputs: []string{"│ [bold]Sentinel Policy Evaluation\n\n│ [bold]→→ Overall Result: [green]PASSED\n│ [dim] This result means that all Sentinel policies passed and the protected behavior is allowed\n│ 1 policies evaluated\n\n│ → Policy set 1: [bold]policy-set-that-passes (1)\n│   ↳ Policy name: [bold]policy-pass\n│      | [green][bold]✓ Passed\n│      | [dim]This policy will pass\n"},
			isError:         false,
		},
		"mandatory-failed": {
			taskStage: func() *tfe.TaskStage {
				ts := &tfe.TaskStage{}
				ts.PolicyEvaluations = []*tfe.PolicyEvaluation{
					{ID: "pol-fail", ResultCount: &tfe.PolicyResultCount{MandatoryFailed: 1}, Status: "failed", PolicyKind: "sentinel"},
				}
				return ts
			},
			writer:          writer,
			context:         integrationContext,
			expectedOutputs: []string{"│ [bold]→→ Overall Result: [red]FAILED\n│ [dim] This result means that one or more Sentinel policies failed. More than likely, this was due to the discovery of violations by the main rule and other sub rules\n│ 1 policies evaluated\n\n│ → Policy set 1: [bold]policy-set-that-fails (1)\n│   ↳ Policy name: [bold]policy-fail\n│      | [red][bold]× Failed\n│      | [dim]This policy will fail"},
			isError:         true,
		},
		"advisory-failed": {
			taskStage: func() *tfe.TaskStage {
				ts := &tfe.TaskStage{}
				ts.PolicyEvaluations = []*tfe.PolicyEvaluation{
					{ID: "adv-fail", ResultCount: &tfe.PolicyResultCount{AdvisoryFailed: 1}, Status: "failed", PolicyKind: "sentinel"},
				}
				return ts
			},
			writer:          writer,
			context:         integrationContext,
			expectedOutputs: []string{"│ [bold]Sentinel Policy Evaluation\n\n│ [bold]→→ Overall Result: [red]FAILED\n│ [dim] This result means that one or more Sentinel policies failed. More than likely, this was due to the discovery of violations by the main rule and other sub rules\n│ 1 policies evaluated\n\n│ → Policy set 1: [bold]policy-set-that-fails (1)\n│   ↳ Policy name: [bold]policy-fail\n│      | [blue][bold]Ⓘ Advisory\n│      | [dim]This policy will fail"},
			isError:         false,
		},
		"unreachable": {
			taskStage: func() *tfe.TaskStage {
				ts := &tfe.TaskStage{}
				ts.PolicyEvaluations = []*tfe.PolicyEvaluation{
					{ID: "adv-fail", ResultCount: &tfe.PolicyResultCount{Errored: 1}, Status: "unreachable", PolicyKind: "sentinel"},
				}
				return ts
			},
			writer:          writer,
			context:         integrationContext,
			expectedOutputs: []string{"Skipping policy evaluation."},
			isError:         false,
		},
		"pending-with-failed-task-stage": {
			taskStage: func() *tfe.TaskStage {
				ts := &tfe.TaskStage{Status: tfe.TaskStageFailed}
				ts.PolicyEvaluations = []*tfe.PolicyEvaluation{
					{ID: "pol-pending", ResultCount: &tfe.PolicyResultCount{}, Status: "pending", PolicyKind: "sentinel"},
				}
				return ts
			},
			writer:          writer,
			context:         integrationContext,
			expectedOutputs: []string{"Skipping policy evaluation."},
			isError:         false,
		},
		"pending-with-canceled-task-stage": {
			taskStage: func() *tfe.TaskStage {
				ts := &tfe.TaskStage{Status: tfe.TaskStageCanceled}
				ts.PolicyEvaluations = []*tfe.PolicyEvaluation{
					{ID: "pol-pending", ResultCount: &tfe.PolicyResultCount{}, Status: "pending", PolicyKind: "sentinel"},
				}
				return ts
			},
			writer:          writer,
			context:         integrationContext,
			expectedOutputs: []string{"Skipping policy evaluation."},
			isError:         false,
		},
		"pending-with-errored-task-stage": {
			taskStage: func() *tfe.TaskStage {
				ts := &tfe.TaskStage{Status: tfe.TaskStageErrored}
				ts.PolicyEvaluations = []*tfe.PolicyEvaluation{
					{ID: "pol-pending", ResultCount: &tfe.PolicyResultCount{}, Status: "pending", PolicyKind: "sentinel"},
				}
				return ts
			},
			writer:          writer,
			context:         integrationContext,
			expectedOutputs: []string{"Skipping policy evaluation."},
			isError:         false,
		},
		"mixed-pending-and-completed-with-failed-task-stage": {
			taskStage: func() *tfe.TaskStage {
				ts := &tfe.TaskStage{Status: tfe.TaskStageFailed}
				ts.PolicyEvaluations = []*tfe.PolicyEvaluation{
					{ID: "pol-pass", ResultCount: &tfe.PolicyResultCount{Passed: 1}, Status: "passed", PolicyKind: "sentinel"},
					{ID: "pol-pending", ResultCount: &tfe.PolicyResultCount{}, Status: "pending", PolicyKind: "sentinel"},
				}
				return ts
			},
			writer:  writer,
			context: integrationContext,
			expectedOutputs: []string{
				"Sentinel Policy Evaluation",
				"Overall Result: [green]PASSED",
				"Skipping 1 pending policy evaluation(s) because task stage is failed.",
			},
			isError: false,
		},
		"multiple-mixed-states-with-failed-task-stage": {
			taskStage: func() *tfe.TaskStage {
				ts := &tfe.TaskStage{Status: tfe.TaskStageFailed}
				ts.PolicyEvaluations = []*tfe.PolicyEvaluation{
					{ID: "pol-pass", ResultCount: &tfe.PolicyResultCount{Passed: 1}, Status: "passed", PolicyKind: "sentinel"},
					{ID: "pol-fail", ResultCount: &tfe.PolicyResultCount{MandatoryFailed: 1}, Status: "failed", PolicyKind: "sentinel"},
					{ID: "pol-pending", ResultCount: &tfe.PolicyResultCount{}, Status: "pending", PolicyKind: "sentinel"},
					{ID: "pol-pending-2", ResultCount: &tfe.PolicyResultCount{}, Status: "running", PolicyKind: "sentinel"},
				}
				return ts
			},
			writer:  writer,
			context: integrationContext,
			expectedOutputs: []string{
				"Sentinel Policy Evaluation",
				"Overall Result: [green]PASSED",
				"Overall Result: [red]FAILED",
				"Skipping 2 pending policy evaluation(s) because task stage is failed.",
			},
			isError: false,
		},
		"mixed-pending-and-completed-with-canceled-task-stage": {
			taskStage: func() *tfe.TaskStage {
				ts := &tfe.TaskStage{Status: tfe.TaskStageCanceled}
				ts.PolicyEvaluations = []*tfe.PolicyEvaluation{
					{ID: "pol-pass", ResultCount: &tfe.PolicyResultCount{Passed: 1}, Status: "passed", PolicyKind: "sentinel"},
					{ID: "pol-pending", ResultCount: &tfe.PolicyResultCount{}, Status: "pending", PolicyKind: "sentinel"},
				}
				return ts
			},
			writer:  writer,
			context: integrationContext,
			expectedOutputs: []string{
				"Sentinel Policy Evaluation",
				"Overall Result: [green]PASSED",
				"Skipping 1 pending policy evaluation(s) because task stage is canceled.",
			},
			isError: false,
		},
		"mixed-pending-and-completed-with-errored-task-stage": {
			taskStage: func() *tfe.TaskStage {
				ts := &tfe.TaskStage{Status: tfe.TaskStageErrored}
				ts.PolicyEvaluations = []*tfe.PolicyEvaluation{
					{ID: "pol-pass", ResultCount: &tfe.PolicyResultCount{Passed: 1}, Status: "passed", PolicyKind: "sentinel"},
					{ID: "pol-pending", ResultCount: &tfe.PolicyResultCount{}, Status: "pending", PolicyKind: "sentinel"},
				}
				return ts
			},
			writer:  writer,
			context: integrationContext,
			expectedOutputs: []string{
				"Sentinel Policy Evaluation",
				"Overall Result: [green]PASSED",
				"Skipping 1 pending policy evaluation(s) because task stage is errored.",
			},
			isError: false,
		},
	}

	for _, c := range cases {
		c.writer.output.Reset()
		trs := policyEvaluationSummarizer{
			cloud: b,
		}
		c.context.Poll(0, 0, func(i int) (bool, error) {
			cont, _, _ := trs.Summarize(c.context, c.writer, c.taskStage())
			if cont {
				return true, nil
			}

			output := c.writer.output.String()
			for _, expected := range c.expectedOutputs {
				if !strings.Contains(output, expected) {
					t.Fatalf("Expected output to contain '%s' but it was:\n\n%s", expected, output)
				}
			}
			return false, nil
		})
	}
}
