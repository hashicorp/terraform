// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
	"strings"
	"testing"

	tfe "github.com/hashicorp/go-tfe"

	"github.com/hashicorp/terraform/internal/command/jsonformat"
	"github.com/hashicorp/terraform/internal/terminal"
)

func TestCloud_writeTFPolicyEvaluations(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	t.Cleanup(bCleanup)

	stream, done := terminal.StreamsForTesting(t)
	b.renderer = &jsonformat.Renderer{
		Streams:  stream,
		Colorize: mockColorize(),
	}

	rendered := []tfPolicyStageOutcomes{
		{
			eval: &tfe.TFPolicyEvaluation{
				StageType:   tfe.TFPolicyEvaluationStageTypePlan,
				Status:      tfe.TFPolicyEvaluationStatusFailed,
				ResultCount: &tfe.TFPolicyEvaluationResultCount{Passed: 1, MandatoryFailed: 1, AdvisoryFailed: 1, Unknown: 1, Errored: 1},
			},
			sets: []*tfe.TFPolicySetOutcome{
				{
					PolicySetName: "AWS policies",
					Outcomes: []*tfe.TFPolicySetPolicyOutcome{
						{PolicyName: "ec2_policy", FileName: "aws_compute.policy.hcl", Status: "passed", EnforcementLevel: tfe.TFPolicyEvaluationOutcomeEnforcementLevelMandatory},
						{PolicyName: "s3_policy", FileName: "aws_storage.policy.hcl", Status: "failed", EnforcementLevel: tfe.TFPolicyEvaluationOutcomeEnforcementLevelMandatory},
						{PolicyName: "tag_policy", FileName: "aws_tags.policy.hcl", Status: "failed", EnforcementLevel: tfe.TFPolicyEvaluationOutcomeEnforcementLevelAdvisory},
						{PolicyName: "vpc_policy", FileName: "aws_vpc.policy.hcl", Status: "unknown", EnforcementLevel: tfe.TFPolicyEvaluationOutcomeEnforcementLevelMandatory},
						{PolicyName: "ami_policy", FileName: "aws_ami.policy.hcl", Status: "errored", EnforcementLevel: tfe.TFPolicyEvaluationOutcomeEnforcementLevelMandatory},
					},
				},
				// Empty set is still shown.
				{PolicySetName: "Empty set"},
			},
		},
		{
			eval: &tfe.TFPolicyEvaluation{
				StageType:   tfe.TFPolicyEvaluationStageTypeInit,
				Status:      tfe.TFPolicyEvaluationStatusPassed,
				ResultCount: &tfe.TFPolicyEvaluationResultCount{Passed: 1},
			},
			sets: []*tfe.TFPolicySetOutcome{
				{
					PolicySetName: "Cloudflare policies",
					Outcomes: []*tfe.TFPolicySetPolicyOutcome{
						{PolicyName: "dns_policy", FileName: "cf_dns.policy.hcl", Status: "passed", EnforcementLevel: tfe.TFPolicyEvaluationOutcomeEnforcementLevelMandatory},
					},
				},
			},
		},
	}

	b.writeTFPolicyEvaluations(rendered)

	got := done(t).Stdout()

	wants := []string{
		"Terraform policy Evaluations",
		"Overall result : ", // overall
		"FAILED",
		"6 policies evaluated", // 5 + 1
		"Plan stage:",
		"5 total, 1 passed, 1 failed, 1 advisory, 1 unknown, 1 errored",
		"Policy set 1: ", "AWS policies",
		`Policy name: "ec2_policy"`, "in aws_compute.policy.hcl",
		"Result: ", "Passed",
		`Policy name: "s3_policy"`, "Failed",
		`Policy name: "tag_policy"`, "Advisory",
		`Policy name: "vpc_policy"`, "Unknown",
		`Policy name: "ami_policy"`, "Errored",
		"Empty set",       // empty set still shown
		"Pre-plan stage:", // Init mapped to Pre-plan
		"1 total, 1 passed, 0 failed, 0 unknown",
		"Cloudflare policies",
		`Policy name: "dns_policy"`, "in cf_dns.policy.hcl",
	}
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q\n---got---\n%s", want, got)
		}
	}
}

func TestCloud_writeTFPolicyEvaluations_allPassed(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	t.Cleanup(bCleanup)

	stream, done := terminal.StreamsForTesting(t)
	b.renderer = &jsonformat.Renderer{Streams: stream, Colorize: mockColorize()}

	rendered := []tfPolicyStageOutcomes{
		{
			eval: &tfe.TFPolicyEvaluation{
				StageType:   tfe.TFPolicyEvaluationStageTypePlan,
				Status:      tfe.TFPolicyEvaluationStatusPassed,
				ResultCount: &tfe.TFPolicyEvaluationResultCount{Passed: 1},
			},
			sets: []*tfe.TFPolicySetOutcome{
				{
					PolicySetName: "AWS policies",
					Outcomes: []*tfe.TFPolicySetPolicyOutcome{
						{PolicyName: "ec2_policy", FileName: "aws_compute.policy.hcl", Status: "passed"},
					},
				},
			},
		},
	}

	b.writeTFPolicyEvaluations(rendered)
	got := done(t).Stdout()

	for _, want := range []string{"Overall result : ", "PASSED", "1 policies evaluated", "Plan stage:"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q\n---got---\n%s", want, got)
		}
	}
	if strings.Contains(got, "FAILED") {
		t.Errorf("did not expect FAILED in all-passed output:\n%s", got)
	}
}

func TestTFPolicyStageCounts(t *testing.T) {
	cases := map[string]struct {
		sets []*tfe.TFPolicySetOutcome
		want string
	}{
		"pass only": {
			sets: []*tfe.TFPolicySetOutcome{{Outcomes: []*tfe.TFPolicySetPolicyOutcome{
				{Status: "passed"}, {Status: "passed"},
			}}},
			want: "2 total, 2 passed, 0 failed, 0 unknown",
		},
		"advisory and errored only shown when present": {
			sets: []*tfe.TFPolicySetOutcome{{Outcomes: []*tfe.TFPolicySetPolicyOutcome{
				{Status: "passed"},
				{Status: "failed", EnforcementLevel: tfe.TFPolicyEvaluationOutcomeEnforcementLevelMandatory},
				{Status: "failed", EnforcementLevel: tfe.TFPolicyEvaluationOutcomeEnforcementLevelAdvisory},
				{Status: "unknown"},
				{Status: "errored"},
			}}},
			want: "5 total, 1 passed, 1 failed, 1 advisory, 1 unknown, 1 errored",
		},
		"empty": {
			sets: nil,
			want: "0 total, 0 passed, 0 failed, 0 unknown",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := tfPolicyStageCounts(tc.sets); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTFPolicyOutcomeResult(t *testing.T) {
	cases := []struct {
		status      string
		enforcement tfe.TFPolicyEvaluationOutcomeEnforcementLevel
		want        string
	}{
		{"passed", tfe.TFPolicyEvaluationOutcomeEnforcementLevelMandatory, "[green][bold]Passed"},
		{"failed", tfe.TFPolicyEvaluationOutcomeEnforcementLevelMandatory, "[red][bold]Failed"},
		{"failed", tfe.TFPolicyEvaluationOutcomeEnforcementLevelAdvisory, "[blue][bold]Advisory"},
		{"unknown", tfe.TFPolicyEvaluationOutcomeEnforcementLevelMandatory, "[yellow][bold]Unknown"},
		{"errored", tfe.TFPolicyEvaluationOutcomeEnforcementLevelMandatory, "[red][bold]Errored"},
		{"weird", tfe.TFPolicyEvaluationOutcomeEnforcementLevelMandatory, "[bold]weird"},
	}
	for _, tc := range cases {
		o := &tfe.TFPolicySetPolicyOutcome{Status: tc.status, EnforcementLevel: tc.enforcement}
		if got := tfPolicyOutcomeResult(o); got != tc.want {
			t.Errorf("status=%q enforcement=%q: got %q, want %q", tc.status, tc.enforcement, got, tc.want)
		}
	}
}

func TestTFPolicyStageLabel(t *testing.T) {
	cases := map[tfe.TFPolicyEvaluationStageType]string{
		tfe.TFPolicyEvaluationStageTypeInit:       "Pre-plan",
		tfe.TFPolicyEvaluationStageTypePlan:       "Plan",
		tfe.TFPolicyEvaluationStageTypeApply:      "Apply",
		tfe.TFPolicyEvaluationStageType("custom"): "custom",
	}
	for stage, want := range cases {
		if got := tfPolicyStageLabel(stage); got != want {
			t.Errorf("stage %q: got %q, want %q", stage, got, want)
		}
	}
}

func TestTFPolicyEvaluationCount(t *testing.T) {
	if got := tfPolicyEvaluationCount(nil); got != 0 {
		t.Errorf("nil count: got %d, want 0", got)
	}
	rc := &tfe.TFPolicyEvaluationResultCount{Passed: 2, MandatoryFailed: 1, AdvisoryFailed: 1, Errored: 1, Unknown: 3}
	if got := tfPolicyEvaluationCount(rc); got != 8 {
		t.Errorf("got %d, want 8", got)
	}
}
