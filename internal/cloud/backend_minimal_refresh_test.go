// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/cli"
	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/go-version"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/terminal"
)

// TestCloud_minimalRefresh verifies that the -minimal-refresh planning option
// is forwarded to HCP Terraform / TFE as the minimal-refresh run attribute,
// and that the attribute is omitted entirely (nil) for ordinary runs.
func TestCloud_minimalRefresh(t *testing.T) {
	cases := map[string]struct {
		apply          bool
		minimalRefresh bool
		destroy        bool
		target         bool
		replace        bool
	}{
		"plan":                                         {},
		"plan minimal refresh":                         {minimalRefresh: true},
		"plan minimal refresh destroy":                 {minimalRefresh: true, destroy: true},
		"plan minimal refresh target":                  {minimalRefresh: true, target: true},
		"plan minimal refresh replace":                 {minimalRefresh: true, replace: true},
		"apply":                                        {apply: true},
		"apply minimal refresh":                        {apply: true, minimalRefresh: true},
		"apply minimal refresh target":                 {apply: true, minimalRefresh: true, target: true},
		"apply minimal refresh replace":                {apply: true, minimalRefresh: true, replace: true},
		"apply minimal refresh and target and replace": {apply: true, minimalRefresh: true, target: true, replace: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			b, bCleanup := testBackendWithName(t)
			defer bCleanup()
			b.client.SetFakeRemoteAPIVersion(MinimalRefreshMinAPIVersion)

			var gotOpts *tfe.RunCreateOptions
			b.client.Runs.(*MockRuns).ModifyNewRun = func(client *MockClient, options tfe.RunCreateOptions, run *tfe.Run) {
				gotOpts = &options
			}

			var op *backendrun.Operation
			var configCleanup func()
			if tc.apply {
				o, cc, done := testOperationApply(t, "./testdata/apply")
				defer done(t)
				op, configCleanup = o, cc
			} else {
				o, cc, done := testOperationPlan(t, "./testdata/plan")
				defer done(t)
				op, configCleanup = o, cc
			}
			defer configCleanup()

			op.Workspace = testBackendSingleWorkspaceName
			op.PlanMinimalRefresh = tc.minimalRefresh
			if tc.destroy {
				op.PlanMode = plans.DestroyMode
			}
			if tc.target {
				addr, _ := addrs.ParseAbsResourceStr("null_resource.foo")
				op.Targets = []addrs.Targetable{addr}
			}
			if tc.replace {
				addr, _ := addrs.ParseAbsResourceInstanceStr("null_resource.foo")
				op.ForceReplace = []addrs.AbsResourceInstance{addr}
			}

			run, err := b.Operation(context.Background(), op)
			if err != nil {
				t.Fatalf("error starting operation: %v", err)
			}
			<-run.Done()

			errOutput := b.CLI.(*cli.MockUi).ErrorWriter.String()
			if run.Result != backendrun.OperationSuccess {
				t.Fatalf("operation failed: %s", errOutput)
			}
			if strings.Contains(errOutput, "not supported") {
				t.Fatalf("unexpected unsupported diagnostic: %s", errOutput)
			}

			if gotOpts == nil {
				t.Fatal("no run was created")
			}
			if tc.minimalRefresh {
				if gotOpts.MinimalRefresh == nil || !*gotOpts.MinimalRefresh {
					t.Errorf("expected MinimalRefresh to be true, got %v", gotOpts.MinimalRefresh)
				}
			} else if gotOpts.MinimalRefresh != nil {
				t.Errorf("expected MinimalRefresh to be nil, got %v", *gotOpts.MinimalRefresh)
			}

			// Minimal refresh must not be converted into -refresh=false or
			// -refresh-only.
			if gotOpts.Refresh == nil || !*gotOpts.Refresh {
				t.Errorf("expected Refresh to be true, got %v", gotOpts.Refresh)
			}
			if gotOpts.RefreshOnly != nil {
				t.Errorf("expected RefreshOnly to be nil, got %v", *gotOpts.RefreshOnly)
			}

			// A plan must remain speculative; an apply must be applicable.
			if got, want := gotOpts.ConfigurationVersion.Speculative, !tc.apply; got != want {
				t.Errorf("wrong Speculative: got %v, want %v", got, want)
			}
			if tc.apply && !run.PlanEmpty {
				runsAPI := b.client.Runs.(*MockRuns)
				for _, r := range runsAPI.Runs {
					if r.Apply == nil {
						t.Errorf("expected an applicable run")
					}
				}
			}

			if got, want := gotOpts.IsDestroy != nil && *gotOpts.IsDestroy, tc.destroy; got != want {
				t.Errorf("wrong IsDestroy: got %v, want %v", got, want)
			}
			var wantTargets, wantReplace []string
			if tc.target {
				wantTargets = []string{"null_resource.foo"}
			}
			if tc.replace {
				wantReplace = []string{"null_resource.foo"}
			}
			if diff := cmp.Diff(wantTargets, gotOpts.TargetAddrs); diff != "" {
				t.Errorf("wrong TargetAddrs\n%s", diff)
			}
			if diff := cmp.Diff(wantReplace, gotOpts.ReplaceAddrs); diff != "" {
				t.Errorf("wrong ReplaceAddrs\n%s", diff)
			}
		})
	}
}

// TestCloud_minimalRefreshAPIVersion verifies that -minimal-refresh is rejected
// before any run is created when the server's API version is too old, and is
// forwarded when the server is at or above the minimum.
func TestCloud_minimalRefreshAPIVersion(t *testing.T) {
	minVersion := version.Must(version.NewVersion(MinimalRefreshMinAPIVersion))
	segs := minVersion.Segments()
	below := fmt.Sprintf("%d.%d", segs[0], segs[1]-1)
	if segs[1] == 0 {
		below = fmt.Sprintf("%d.99", segs[0]-1)
	}
	above := fmt.Sprintf("%d.%d", segs[0], segs[1]+1)

	cases := map[string]struct {
		apiVersion string
		apply      bool
		wantErr    bool
	}{
		"plan below minimum":  {apiVersion: below, wantErr: true},
		"apply below minimum": {apiVersion: below, apply: true, wantErr: true},
		"plan unparseable":    {apiVersion: "", wantErr: true},
		"plan at minimum":     {apiVersion: MinimalRefreshMinAPIVersion},
		"apply at minimum":    {apiVersion: MinimalRefreshMinAPIVersion, apply: true},
		"plan above minimum":  {apiVersion: above},
		"apply above minimum": {apiVersion: above, apply: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			b, bCleanup := testBackendWithName(t)
			defer bCleanup()
			b.client.SetFakeRemoteAPIVersion(tc.apiVersion)

			var op *backendrun.Operation
			var configCleanup func()
			var done func(*testing.T) *terminal.TestOutput
			if tc.apply {
				op, configCleanup, done = testOperationApply(t, "./testdata/apply")
			} else {
				op, configCleanup, done = testOperationPlan(t, "./testdata/plan")
			}
			defer configCleanup()

			op.Workspace = testBackendSingleWorkspaceName
			op.PlanMinimalRefresh = true

			run, err := b.Operation(context.Background(), op)
			if err != nil {
				t.Fatalf("error starting operation: %v", err)
			}
			<-run.Done()
			errOutput := strings.Join(strings.Fields(done(t).Stderr()), " ")
			runsAPI := b.client.Runs.(*MockRuns)

			if tc.wantErr {
				if run.Result == backendrun.OperationSuccess {
					t.Fatal("expected operation to fail")
				}
				want := fmt.Sprintf("The host %s does not support the -minimal-refresh option", b.Hostname)
				if !strings.Contains(errOutput, want) {
					t.Errorf("missing %q in error output:\n%s", want, errOutput)
				}
				if len(runsAPI.Runs) != 0 {
					t.Errorf("expected no runs to be created, got %d", len(runsAPI.Runs))
				}
				return
			}

			if run.Result != backendrun.OperationSuccess {
				t.Fatalf("operation failed: %s", errOutput)
			}
			if got := len(runsAPI.Runs); got != 1 {
				t.Fatalf("expected 1 run, got %d", got)
			}
			for _, r := range runsAPI.Runs {
				if !r.MinimalRefresh {
					t.Error("expected MinimalRefresh to be forwarded")
				}
			}
		})
	}
}
