// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
	"strings"
	"testing"

	"github.com/hashicorp/go-tfe"
)

func TestCloud_runTaskStageWithPolicyEvaluation(t *testing.T) {
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
					{ID: "pol-pass", ResultCount: &tfe.PolicyResultCount{Passed: 1}, Status: "passed"},
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
					{ID: "pol-fail", ResultCount: &tfe.PolicyResultCount{MandatoryFailed: 1}, Status: "failed"},
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
					{ID: "adv-fail", ResultCount: &tfe.PolicyResultCount{AdvisoryFailed: 1}, Status: "failed"},
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
					{ID: "adv-fail", ResultCount: &tfe.PolicyResultCount{Errored: 1}, Status: "unreachable"},
				}
				return ts
			},
			writer:          writer,
			context:         integrationContext,
			expectedOutputs: []string{"Skipping policy evaluation."},
			isError:         false,
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
