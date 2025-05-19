// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/internal/plans"
)

// A brief discourse on the theory of testing for this feature. Doing
// `terraform show cloudplan.tfplan` relies on the correctness of the following
// behaviors:
//
// 1. HCP Terraform API returns redacted or unredacted plan JSON on request, if permission
// requirements are met and the run is in a condition where that JSON exists.
// 2. Cloud.ShowPlanForRun() makes correct API calls, calculates metadata
// properly given a tfe.Run, and returns either a cloudplan.RemotePlanJSON or an err.
// 3. The Show command instantiates Cloud backend when given a cloud planfile,
// calls .ShowPlanForRun() on it, and passes result to Display() impls.
// 4. Display() impls yield the correct output when given a cloud plan json biscuit.
//
// 1 is axiomatic and outside our domain. 3 is regrettably totally untestable
// unless we refactor the Meta command to enable stubbing out a backend factory
// or something, which seems inadvisable at this juncture. 4 is exercised over
// in internal/command/views/show_test.go. And thus, this file only cares about
// item 2.

// 404 on run: special error message
func TestCloud_showMissingRun(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()
	mockSROWorkspace(t, b, testBackendSingleWorkspaceName)

	absentRunID := "run-WwwwXxxxYyyyZzzz"
	_, err := b.ShowPlanForRun(context.Background(), absentRunID, "app.terraform.io", true)
	if !strings.Contains(err.Error(), "terraform login") {
		t.Fatalf("expected error message to suggest checking your login status, instead got: %s", err)
	}
}

// If redacted json is available but unredacted is not
func TestCloud_showMissingUnredactedJson(t *testing.T) {
	b, mc, bCleanup := testBackendAndMocksWithName(t)
	defer bCleanup()
	mockSROWorkspace(t, b, testBackendSingleWorkspaceName)

	ctx := context.Background()

	runID, err := testCloudRunForShow(mc, "./testdata/plan-json-basic-no-unredacted", tfe.RunPlannedAndSaved, tfe.PlanFinished)
	if err != nil {
		t.Fatalf("failed to init test data: %s", err)
	}
	// Showing the human-formatted plan should still work as expected!
	redacted, err := b.ShowPlanForRun(ctx, runID, "app.terraform.io", true)
	if err != nil {
		t.Fatalf("failed to show plan for human, even though redacted json should be present: %s", err)
	}
	if !strings.Contains(string(redacted.JSONBytes), `"plan_format_version":`) {
		t.Fatalf("show for human doesn't include expected redacted json content")
	}
	// Should be marked as containing changes and non-errored
	canNotApply := false
	errored := false
	for _, opt := range redacted.Qualities {
		if opt == plans.NoChanges {
			canNotApply = true
		}
		if opt == plans.Errored {
			errored = true
		}
	}
	if canNotApply || errored {
		t.Fatalf("expected neither errored nor can't-apply in opts, instead got: %#v", redacted.Qualities)
	}

	// But show -json should result in a special error.
	_, err = b.ShowPlanForRun(ctx, runID, "app.terraform.io", false)
	if err == nil {
		t.Fatalf("unexpected success: reading unredacted json without admin permissions should have errored")
	}
	if !strings.Contains(err.Error(), "admin") {
		t.Fatalf("expected error message to suggest your permissions are wrong, instead got: %s", err)
	}
}

// If both kinds of json are available, both kinds of show should work
func TestCloud_showIncludesUnredactedJson(t *testing.T) {
	b, mc, bCleanup := testBackendAndMocksWithName(t)
	defer bCleanup()
	mockSROWorkspace(t, b, testBackendSingleWorkspaceName)

	ctx := context.Background()

	runID, err := testCloudRunForShow(mc, "./testdata/plan-json-basic", tfe.RunPlannedAndSaved, tfe.PlanFinished)
	if err != nil {
		t.Fatalf("failed to init test data: %s", err)
	}
	// Showing the human-formatted plan should work as expected:
	redacted, err := b.ShowPlanForRun(ctx, runID, "app.terraform.io", true)
	if err != nil {
		t.Fatalf("failed to show plan for human, even though redacted json should be present: %s", err)
	}
	if !strings.Contains(string(redacted.JSONBytes), `"plan_format_version":`) {
		t.Fatalf("show for human doesn't include expected redacted json content")
	}
	// Showing the external json plan format should work as expected:
	unredacted, err := b.ShowPlanForRun(ctx, runID, "app.terraform.io", false)
	if err != nil {
		t.Fatalf("failed to show plan for robot, even though unredacted json should be present: %s", err)
	}
	if !strings.Contains(string(unredacted.JSONBytes), `"format_version":`) {
		t.Fatalf("show for robot doesn't include expected unredacted json content")
	}
}

func TestCloud_showNoChanges(t *testing.T) {
	b, mc, bCleanup := testBackendAndMocksWithName(t)
	defer bCleanup()
	mockSROWorkspace(t, b, testBackendSingleWorkspaceName)

	ctx := context.Background()

	runID, err := testCloudRunForShow(mc, "./testdata/plan-json-no-changes", tfe.RunPlannedAndSaved, tfe.PlanFinished)
	if err != nil {
		t.Fatalf("failed to init test data: %s", err)
	}
	// Showing the human-formatted plan should work as expected:
	redacted, err := b.ShowPlanForRun(ctx, runID, "app.terraform.io", true)
	if err != nil {
		t.Fatalf("failed to show plan for human, even though redacted json should be present: %s", err)
	}
	// Should be marked as no changes
	canNotApply := false
	for _, opt := range redacted.Qualities {
		if opt == plans.NoChanges {
			canNotApply = true
		}
	}
	if !canNotApply {
		t.Fatalf("expected opts to include CanNotApply, instead got: %#v", redacted.Qualities)
	}
}

func TestCloud_showFooterNotConfirmable(t *testing.T) {
	b, mc, bCleanup := testBackendAndMocksWithName(t)
	defer bCleanup()
	mockSROWorkspace(t, b, testBackendSingleWorkspaceName)

	ctx := context.Background()

	runID, err := testCloudRunForShow(mc, "./testdata/plan-json-full", tfe.RunDiscarded, tfe.PlanFinished)
	if err != nil {
		t.Fatalf("failed to init test data: %s", err)
	}

	// A little more custom run tweaking:
	mc.Runs.Runs[runID].Actions.IsConfirmable = false

	// Showing the human-formatted plan should work as expected:
	redacted, err := b.ShowPlanForRun(ctx, runID, "app.terraform.io", true)
	if err != nil {
		t.Fatalf("failed to show plan for human, even though redacted json should be present: %s", err)
	}

	// Footer should mention that you can't apply it:
	if !strings.Contains(redacted.RunFooter, "not confirmable") {
		t.Fatalf("footer should call out that run isn't confirmable, instead got: %s", redacted.RunFooter)
	}
}

func testCloudRunForShow(mc *MockClient, configDir string, runStatus tfe.RunStatus, planStatus tfe.PlanStatus) (string, error) {
	ctx := context.Background()

	// get workspace ID
	wsID := mc.Workspaces.workspaceNames[testBackendSingleWorkspaceName].ID
	// create and upload config version
	cvOpts := tfe.ConfigurationVersionCreateOptions{
		AutoQueueRuns: tfe.Bool(false),
		Speculative:   tfe.Bool(false),
	}
	cv, err := mc.ConfigurationVersions.Create(ctx, wsID, cvOpts)
	if err != nil {
		return "", err
	}
	absDir, err := filepath.Abs(configDir)
	if err != nil {
		return "", err
	}
	err = mc.ConfigurationVersions.Upload(ctx, cv.UploadURL, absDir)
	if err != nil {
		return "", err
	}
	// create run
	rOpts := tfe.RunCreateOptions{
		PlanOnly:             tfe.Bool(false),
		IsDestroy:            tfe.Bool(false),
		RefreshOnly:          tfe.Bool(false),
		ConfigurationVersion: cv,
		Workspace:            &tfe.Workspace{ID: wsID},
	}
	r, err := mc.Runs.Create(ctx, rOpts)
	if err != nil {
		return "", err
	}
	// mess with statuses (this is what requires full access to mock client)
	mc.Runs.Runs[r.ID].Status = runStatus
	mc.Plans.plans[r.Plan.ID].Status = planStatus

	// return the ID
	return r.ID, nil
}
