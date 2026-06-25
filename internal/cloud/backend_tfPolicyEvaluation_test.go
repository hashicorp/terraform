// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
	"context"
	"strings"
	"testing"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/mitchellh/colorstring"

	"github.com/hashicorp/terraform/internal/command/jsonformat"
	"github.com/hashicorp/terraform/internal/terminal"
)

// fakeTFPolicyOutcomes is a paginated fake for the tfe.TFPolicyEvaluationOutcomes
// interface. It serves one configured page per call and records the page numbers
// requested so tests can assert pagination is followed.
type fakeTFPolicyOutcomes struct {
	pages [][]*tfe.TFPolicySetOutcome
	calls []int
}

func (f *fakeTFPolicyOutcomes) List(_ context.Context, _ string, opts *tfe.TFPolicyEvaluationListOptions) (*tfe.TFPolicyEvaluationOutcomeList, error) {
	page := 1
	if opts != nil && opts.PageNumber > 0 {
		page = opts.PageNumber
	}
	f.calls = append(f.calls, page)

	total := len(f.pages)
	next := 0
	if page < total {
		next = page + 1
	}
	return &tfe.TFPolicyEvaluationOutcomeList{
		Pagination: &tfe.Pagination{CurrentPage: page, NextPage: next, TotalPages: total},
		Items:      f.pages[page-1],
	}, nil
}

func TestCloud_listTFPolicyOutcomes_pagination(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	t.Cleanup(bCleanup)

	fake := &fakeTFPolicyOutcomes{
		pages: [][]*tfe.TFPolicySetOutcome{
			{{PolicySetName: "set-a"}, {PolicySetName: "set-b"}},
			{{PolicySetName: "set-c"}},
		},
	}
	b.client.TFPolicyEvaluationOutcomes = fake

	sets, err := b.listTFPolicyOutcomes(context.Background(), "tfpe-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"set-a", "set-b", "set-c"}
	if len(sets) != len(want) {
		t.Fatalf("want %d outcomes across pages, got %d", len(want), len(sets))
	}
	for i, s := range sets {
		if s.PolicySetName != want[i] {
			t.Errorf("set %d: want %q, got %q", i, want[i], s.PolicySetName)
		}
	}
	// Both pages must be requested, in order.
	if len(fake.calls) != 2 || fake.calls[0] != 1 || fake.calls[1] != 2 {
		t.Errorf("want page requests [1 2], got %v", fake.calls)
	}
}

func TestCloud_writeTFPolicyEvaluations(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	t.Cleanup(bCleanup)

	stream, done := terminal.StreamsForTesting(t)
	// Colors are disabled so the test asserts the exact text structure without
	// ANSI escape noise; color rendering is the colorstring library's concern.
	b.renderer = &jsonformat.Renderer{
		Streams:  stream,
		Colorize: &colorstring.Colorize{Disable: true},
	}

	rendered := []tfPolicyStageOutcomes{
		{
			eval: &tfe.TFPolicyEvaluation{
				StageType: tfe.TFPolicyEvaluationStageTypePlan,
				Status:    tfe.TFPolicyEvaluationStatusFailed,
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
				StageType: tfe.TFPolicyEvaluationStageTypeInit,
				Status:    tfe.TFPolicyEvaluationStatusPassed,
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

	want := `
------------------------------------------------------------------------

Terraform policy Evaluations

→→ Overall result : FAILED
This means that one or more Terraform policies failed.

6 policies evaluated

Plan stage: FAILED
5 total, 1 passed, 1 failed, 1 advisory, 1 unknown, 1 errored
  → Policy set 1: AWS policies
    → Policy name: "ec2_policy" in aws_compute.policy.hcl
      → Result: Passed
    → Policy name: "s3_policy" in aws_storage.policy.hcl
      → Result: Failed
    → Policy name: "tag_policy" in aws_tags.policy.hcl
      → Result: Advisory
    → Policy name: "vpc_policy" in aws_vpc.policy.hcl
      → Result: Unknown
    → Policy name: "ami_policy" in aws_ami.policy.hcl
      → Result: Errored
  → Policy set 2: Empty set
Pre-plan stage: PASSED
1 total, 1 passed, 0 failed, 0 unknown
  → Policy set 1: Cloudflare policies
    → Policy name: "dns_policy" in cf_dns.policy.hcl
      → Result: Passed
`
	if got != want {
		t.Errorf("unexpected output\n--- got ---\n%s\n--- want ---\n%s", got, want)
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
