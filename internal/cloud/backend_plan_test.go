// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/cloud/cloudplan"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/clistate"
	"github.com/hashicorp/terraform/internal/command/jsonformat"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/initwd"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/planfile"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/mitchellh/cli"
)

func testOperationPlan(t *testing.T, configDir string) (*backend.Operation, func(), func(*testing.T) *terminal.TestOutput) {
	t.Helper()

	return testOperationPlanWithTimeout(t, configDir, 0)
}

func testOperationPlanWithTimeout(t *testing.T, configDir string, timeout time.Duration) (*backend.Operation, func(), func(*testing.T) *terminal.TestOutput) {
	t.Helper()

	_, configLoader, configCleanup := initwd.MustLoadConfigForTests(t, configDir)

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	stateLockerView := views.NewStateLocker(arguments.ViewHuman, view)
	operationView := views.NewOperation(arguments.ViewHuman, false, view)

	// Many of our tests use an overridden "null" provider that's just in-memory
	// inside the test process, not a separate plugin on disk.
	depLocks := depsfile.NewLocks()
	depLocks.SetProviderOverridden(addrs.MustParseProviderSourceString("registry.terraform.io/hashicorp/null"))

	return &backend.Operation{
		ConfigDir:       configDir,
		ConfigLoader:    configLoader,
		PlanRefresh:     true,
		StateLocker:     clistate.NewLocker(timeout, stateLockerView),
		Type:            backend.OperationTypePlan,
		View:            operationView,
		DependencyLocks: depLocks,
	}, configCleanup, done
}

func TestCloud_planBasic(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	op, configCleanup, done := testOperationPlan(t, "./testdata/plan")
	defer configCleanup()
	defer done(t)

	op.Workspace = testBackendSingleWorkspaceName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result != backend.OperationSuccess {
		t.Fatalf("operation failed: %s", b.CLI.(*cli.MockUi).ErrorWriter.String())
	}
	if run.PlanEmpty {
		t.Fatal("expected a non-empty plan")
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "Running plan in Terraform Cloud") {
		t.Fatalf("expected TFC header in output: %s", output)
	}
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("expected plan summary in output: %s", output)
	}

	stateMgr, _ := b.StateMgr(testBackendSingleWorkspaceName)
	// An error suggests that the state was not unlocked after the operation finished
	if _, err := stateMgr.Lock(statemgr.NewLockInfo()); err != nil {
		t.Fatalf("unexpected error locking state after successful plan: %s", err.Error())
	}
}

func TestCloud_planJSONBasic(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	stream, close := terminal.StreamsForTesting(t)

	b.renderer = &jsonformat.Renderer{
		Streams:  stream,
		Colorize: mockColorize(),
	}

	op, configCleanup, done := testOperationPlan(t, "./testdata/plan-json-basic")
	defer configCleanup()
	defer done(t)

	op.Workspace = testBackendSingleWorkspaceName

	mockSROWorkspace(t, b, op.Workspace)

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result != backend.OperationSuccess {
		t.Fatalf("operation failed: %s", b.CLI.(*cli.MockUi).ErrorWriter.String())
	}
	if run.PlanEmpty {
		t.Fatal("expected a non-empty plan")
	}

	outp := close(t)
	gotOut := outp.Stdout()

	if !strings.Contains(gotOut, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("expected plan summary in output: %s", gotOut)
	}

	stateMgr, _ := b.StateMgr(testBackendSingleWorkspaceName)
	// An error suggests that the state was not unlocked after the operation finished
	if _, err := stateMgr.Lock(statemgr.NewLockInfo()); err != nil {
		t.Fatalf("unexpected error locking state after successful plan: %s", err.Error())
	}
}

func TestCloud_planCanceled(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	op, configCleanup, done := testOperationPlan(t, "./testdata/plan")
	defer configCleanup()
	defer done(t)

	op.Workspace = testBackendSingleWorkspaceName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	// Stop the run to simulate a Ctrl-C.
	run.Stop()

	<-run.Done()
	if run.Result == backend.OperationSuccess {
		t.Fatal("expected plan operation to fail")
	}

	stateMgr, _ := b.StateMgr(testBackendSingleWorkspaceName)
	// An error suggests that the state was not unlocked after the operation finished
	if _, err := stateMgr.Lock(statemgr.NewLockInfo()); err != nil {
		t.Fatalf("unexpected error locking state after cancelled plan: %s", err.Error())
	}
}

func TestCloud_planLongLine(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	op, configCleanup, done := testOperationPlan(t, "./testdata/plan-long-line")
	defer configCleanup()
	defer done(t)

	op.Workspace = testBackendSingleWorkspaceName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result != backend.OperationSuccess {
		t.Fatalf("operation failed: %s", b.CLI.(*cli.MockUi).ErrorWriter.String())
	}
	if run.PlanEmpty {
		t.Fatal("expected a non-empty plan")
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "Running plan in Terraform Cloud") {
		t.Fatalf("expected TFC header in output: %s", output)
	}
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("expected plan summary in output: %s", output)
	}
}

func TestCloud_planJSONFull(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	stream, close := terminal.StreamsForTesting(t)

	b.renderer = &jsonformat.Renderer{
		Streams:  stream,
		Colorize: mockColorize(),
	}

	op, configCleanup, done := testOperationPlan(t, "./testdata/plan-json-full")
	defer configCleanup()
	defer done(t)

	op.Workspace = testBackendSingleWorkspaceName

	mockSROWorkspace(t, b, op.Workspace)

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result != backend.OperationSuccess {
		t.Fatalf("operation failed: %s", b.CLI.(*cli.MockUi).ErrorWriter.String())
	}
	if run.PlanEmpty {
		t.Fatal("expected a non-empty plan")
	}

	outp := close(t)
	gotOut := outp.Stdout()

	if !strings.Contains(gotOut, "tfcoremock_simple_resource.example: Refreshing state... [id=my-simple-resource]") {
		t.Fatalf("expected plan log: %s", gotOut)
	}

	if !strings.Contains(gotOut, "2 to add, 0 to change, 0 to destroy") {
		t.Fatalf("expected plan summary in output: %s", gotOut)
	}

	stateMgr, _ := b.StateMgr(testBackendSingleWorkspaceName)
	// An error suggests that the state was not unlocked after the operation finished
	if _, err := stateMgr.Lock(statemgr.NewLockInfo()); err != nil {
		t.Fatalf("unexpected error locking state after successful plan: %s", err.Error())
	}
}

func TestCloud_planWithoutPermissions(t *testing.T) {
	b, bCleanup := testBackendWithTags(t)
	defer bCleanup()

	// Create a named workspace without permissions.
	w, err := b.client.Workspaces.Create(
		context.Background(),
		b.organization,
		tfe.WorkspaceCreateOptions{
			Name: tfe.String("prod"),
		},
	)
	if err != nil {
		t.Fatalf("error creating named workspace: %v", err)
	}
	w.Permissions.CanQueueRun = false

	op, configCleanup, done := testOperationPlan(t, "./testdata/plan")
	defer configCleanup()

	op.Workspace = "prod"

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	output := done(t)
	if run.Result == backend.OperationSuccess {
		t.Fatal("expected plan operation to fail")
	}

	errOutput := output.Stderr()
	if !strings.Contains(errOutput, "Insufficient rights to generate a plan") {
		t.Fatalf("expected a permissions error, got: %v", errOutput)
	}
}

func TestCloud_planWithParallelism(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	op, configCleanup, done := testOperationPlan(t, "./testdata/plan")
	defer configCleanup()

	if b.ContextOpts == nil {
		b.ContextOpts = &terraform.ContextOpts{}
	}
	b.ContextOpts.Parallelism = 3
	op.Workspace = testBackendSingleWorkspaceName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	output := done(t)
	if run.Result == backend.OperationSuccess {
		t.Fatal("expected plan operation to fail")
	}

	errOutput := output.Stderr()
	if !strings.Contains(errOutput, "parallelism values are currently not supported") {
		t.Fatalf("expected a parallelism error, got: %v", errOutput)
	}
}

func TestCloud_planWithPlan(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	op, configCleanup, done := testOperationPlan(t, "./testdata/plan")
	defer configCleanup()

	op.PlanFile = planfile.NewWrappedLocal(&planfile.Reader{})
	op.Workspace = testBackendSingleWorkspaceName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	output := done(t)
	if run.Result == backend.OperationSuccess {
		t.Fatal("expected plan operation to fail")
	}
	if !run.PlanEmpty {
		t.Fatalf("expected plan to be empty")
	}

	errOutput := output.Stderr()
	if !strings.Contains(errOutput, "saved plan is currently not supported") {
		t.Fatalf("expected a saved plan error, got: %v", errOutput)
	}
}

func TestCloud_planWithPath(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	op, configCleanup, done := testOperationPlan(t, "./testdata/plan")
	defer configCleanup()
	defer done(t)

	tmpDir := t.TempDir()
	pfPath := tmpDir + "/plan.tfplan"
	op.PlanOutPath = pfPath
	op.Workspace = testBackendSingleWorkspaceName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result != backend.OperationSuccess {
		t.Fatalf("operation failed: %s", b.CLI.(*cli.MockUi).ErrorWriter.String())
	}
	if run.PlanEmpty {
		t.Fatal("expected a non-empty plan")
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "Running plan in Terraform Cloud") {
		t.Fatalf("expected TFC header in output: %s", output)
	}
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("expected plan summary in output: %s", output)
	}

	plan, err := cloudplan.LoadSavedPlanBookmark(pfPath)
	if err != nil {
		t.Fatalf("error loading cloud plan file: %v", err)
	}
	if !strings.Contains(plan.RunID, "run-") || plan.Hostname != "app.terraform.io" {
		t.Fatalf("unexpected contents in saved cloud plan: %v", plan)
	}
}

func TestCloud_planWithoutRefresh(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	op, configCleanup, done := testOperationPlan(t, "./testdata/plan")
	defer configCleanup()
	defer done(t)

	op.PlanRefresh = false
	op.Workspace = testBackendSingleWorkspaceName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result != backend.OperationSuccess {
		t.Fatalf("operation failed: %s", b.CLI.(*cli.MockUi).ErrorWriter.String())
	}
	if run.PlanEmpty {
		t.Fatal("expected a non-empty plan")
	}

	// We should find a run inside the mock client that has refresh set
	// to false.
	runsAPI := b.client.Runs.(*MockRuns)
	if got, want := len(runsAPI.Runs), 1; got != want {
		t.Fatalf("wrong number of runs in the mock client %d; want %d", got, want)
	}
	for _, run := range runsAPI.Runs {
		if diff := cmp.Diff(false, run.Refresh); diff != "" {
			t.Errorf("wrong Refresh setting in the created run\n%s", diff)
		}
	}
}

func TestCloud_planWithRefreshOnly(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	op, configCleanup, done := testOperationPlan(t, "./testdata/plan")
	defer configCleanup()
	defer done(t)

	op.PlanMode = plans.RefreshOnlyMode
	op.Workspace = testBackendSingleWorkspaceName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result != backend.OperationSuccess {
		t.Fatalf("operation failed: %s", b.CLI.(*cli.MockUi).ErrorWriter.String())
	}
	if run.PlanEmpty {
		t.Fatal("expected a non-empty plan")
	}

	// We should find a run inside the mock client that has refresh-only set
	// to true.
	runsAPI := b.client.Runs.(*MockRuns)
	if got, want := len(runsAPI.Runs), 1; got != want {
		t.Fatalf("wrong number of runs in the mock client %d; want %d", got, want)
	}
	for _, run := range runsAPI.Runs {
		if diff := cmp.Diff(true, run.RefreshOnly); diff != "" {
			t.Errorf("wrong RefreshOnly setting in the created run\n%s", diff)
		}
	}
}

func TestCloud_planWithTarget(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	// When the backend code creates a new run, we'll tweak it so that it
	// has a cost estimation object with the "skipped_due_to_targeting" status,
	// emulating how a real server is expected to behave in that case.
	b.client.Runs.(*MockRuns).ModifyNewRun = func(client *MockClient, options tfe.RunCreateOptions, run *tfe.Run) {
		const fakeID = "fake"
		// This is the cost estimate object embedded in the run itself which
		// the backend will use to learn the ID to request from the cost
		// estimates endpoint. It's pending to simulate what a freshly-created
		// run is likely to look like.
		run.CostEstimate = &tfe.CostEstimate{
			ID:     fakeID,
			Status: "pending",
		}
		// The backend will then use the main cost estimation API to retrieve
		// the same ID indicated in the object above, where we'll then return
		// the status "skipped_due_to_targeting" to trigger the special skip
		// message in the backend output.
		client.CostEstimates.Estimations[fakeID] = &tfe.CostEstimate{
			ID:     fakeID,
			Status: "skipped_due_to_targeting",
		}
	}

	op, configCleanup, done := testOperationPlan(t, "./testdata/plan")
	defer configCleanup()
	defer done(t)

	addr, _ := addrs.ParseAbsResourceStr("null_resource.foo")

	op.Targets = []addrs.Targetable{addr}
	op.Workspace = testBackendSingleWorkspaceName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result != backend.OperationSuccess {
		t.Fatal("expected plan operation to succeed")
	}
	if run.PlanEmpty {
		t.Fatalf("expected plan to be non-empty")
	}

	// testBackendDefault above attached a "mock UI" to our backend, so we
	// can retrieve its non-error output via the OutputWriter in-memory buffer.
	gotOutput := b.CLI.(*cli.MockUi).OutputWriter.String()
	if wantOutput := "Not available for this plan, because it was created with the -target option."; !strings.Contains(gotOutput, wantOutput) {
		t.Errorf("missing message about skipped cost estimation\ngot:\n%s\nwant substring: %s", gotOutput, wantOutput)
	}

	// We should find a run inside the mock client that has the same
	// target address we requested above.
	runsAPI := b.client.Runs.(*MockRuns)
	if got, want := len(runsAPI.Runs), 1; got != want {
		t.Fatalf("wrong number of runs in the mock client %d; want %d", got, want)
	}
	for _, run := range runsAPI.Runs {
		if diff := cmp.Diff([]string{"null_resource.foo"}, run.TargetAddrs); diff != "" {
			t.Errorf("wrong TargetAddrs in the created run\n%s", diff)
		}
	}
}

func TestCloud_planWithReplace(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	op, configCleanup, done := testOperationPlan(t, "./testdata/plan")
	defer configCleanup()
	defer done(t)

	addr, _ := addrs.ParseAbsResourceInstanceStr("null_resource.foo")

	op.ForceReplace = []addrs.AbsResourceInstance{addr}
	op.Workspace = testBackendSingleWorkspaceName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result != backend.OperationSuccess {
		t.Fatal("expected plan operation to succeed")
	}
	if run.PlanEmpty {
		t.Fatalf("expected plan to be non-empty")
	}

	// We should find a run inside the mock client that has the same
	// refresh address we requested above.
	runsAPI := b.client.Runs.(*MockRuns)
	if got, want := len(runsAPI.Runs), 1; got != want {
		t.Fatalf("wrong number of runs in the mock client %d; want %d", got, want)
	}
	for _, run := range runsAPI.Runs {
		if diff := cmp.Diff([]string{"null_resource.foo"}, run.ReplaceAddrs); diff != "" {
			t.Errorf("wrong ReplaceAddrs in the created run\n%s", diff)
		}
	}
}

func TestCloud_planWithRequiredVariables(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	op, configCleanup, done := testOperationPlan(t, "./testdata/plan-variables")
	defer configCleanup()
	defer done(t)

	op.Variables = testVariables(terraform.ValueFromCLIArg, "foo") // "bar" variable defined in config is  missing
	op.Workspace = testBackendSingleWorkspaceName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	// The usual error of a required variable being missing is deferred and the operation
	// is successful.
	if run.Result != backend.OperationSuccess {
		t.Fatal("expected plan operation to succeed")
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "Running plan in Terraform Cloud") {
		t.Fatalf("unexpected TFC header in output: %s", output)
	}
}

func TestCloud_planNoConfig(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	op, configCleanup, done := testOperationPlan(t, "./testdata/empty")
	defer configCleanup()

	op.Workspace = testBackendSingleWorkspaceName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	output := done(t)
	if run.Result == backend.OperationSuccess {
		t.Fatal("expected plan operation to fail")
	}
	if !run.PlanEmpty {
		t.Fatalf("expected plan to be empty")
	}

	errOutput := output.Stderr()
	if !strings.Contains(errOutput, "configuration files found") {
		t.Fatalf("expected configuration files error, got: %v", errOutput)
	}
}

func TestCloud_planNoChanges(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	op, configCleanup, done := testOperationPlan(t, "./testdata/plan-no-changes")
	defer configCleanup()
	defer done(t)

	op.Workspace = testBackendSingleWorkspaceName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result != backend.OperationSuccess {
		t.Fatalf("operation failed: %s", b.CLI.(*cli.MockUi).ErrorWriter.String())
	}
	if !run.PlanEmpty {
		t.Fatalf("expected plan to be empty")
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "No changes. Infrastructure is up-to-date.") {
		t.Fatalf("expected no changes in plan summary: %s", output)
	}
	if !strings.Contains(output, "Sentinel Result: true") {
		t.Fatalf("expected policy check result in output: %s", output)
	}
}

func TestCloud_planForceLocal(t *testing.T) {
	// Set TF_FORCE_LOCAL_BACKEND so the cloud backend will use
	// the local backend with itself as embedded backend.
	if err := os.Setenv("TF_FORCE_LOCAL_BACKEND", "1"); err != nil {
		t.Fatalf("error setting environment variable TF_FORCE_LOCAL_BACKEND: %v", err)
	}
	defer os.Unsetenv("TF_FORCE_LOCAL_BACKEND")

	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	op, configCleanup, done := testOperationPlan(t, "./testdata/plan")
	defer configCleanup()
	defer done(t)

	op.Workspace = testBackendSingleWorkspaceName

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewOperation(arguments.ViewHuman, false, views.NewView(streams))
	op.View = view

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result != backend.OperationSuccess {
		t.Fatalf("operation failed: %s", b.CLI.(*cli.MockUi).ErrorWriter.String())
	}
	if run.PlanEmpty {
		t.Fatalf("expected a non-empty plan")
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if strings.Contains(output, "Running plan in Terraform Cloud") {
		t.Fatalf("unexpected TFC header in output: %s", output)
	}
	if output := done(t).Stdout(); !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("expected plan summary in output: %s", output)
	}
}

func TestCloud_planWithoutOperationsEntitlement(t *testing.T) {
	b, bCleanup := testBackendNoOperations(t)
	defer bCleanup()

	op, configCleanup, done := testOperationPlan(t, "./testdata/plan")
	defer configCleanup()
	defer done(t)

	op.Workspace = testBackendSingleWorkspaceName

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewOperation(arguments.ViewHuman, false, views.NewView(streams))
	op.View = view

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result != backend.OperationSuccess {
		t.Fatalf("operation failed: %s", b.CLI.(*cli.MockUi).ErrorWriter.String())
	}
	if run.PlanEmpty {
		t.Fatalf("expected a non-empty plan")
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if strings.Contains(output, "Running plan in Terraform Cloud") {
		t.Fatalf("unexpected TFC header in output: %s", output)
	}
	if output := done(t).Stdout(); !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("expected plan summary in output: %s", output)
	}
}

func TestCloud_planWorkspaceWithoutOperations(t *testing.T) {
	b, bCleanup := testBackendWithTags(t)
	defer bCleanup()

	ctx := context.Background()

	// Create a named workspace that doesn't allow operations.
	_, err := b.client.Workspaces.Create(
		ctx,
		b.organization,
		tfe.WorkspaceCreateOptions{
			Name: tfe.String("no-operations"),
		},
	)
	if err != nil {
		t.Fatalf("error creating named workspace: %v", err)
	}

	op, configCleanup, done := testOperationPlan(t, "./testdata/plan")
	defer configCleanup()
	defer done(t)

	op.Workspace = "no-operations"

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewOperation(arguments.ViewHuman, false, views.NewView(streams))
	op.View = view

	run, err := b.Operation(ctx, op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result != backend.OperationSuccess {
		t.Fatalf("operation failed: %s", b.CLI.(*cli.MockUi).ErrorWriter.String())
	}
	if run.PlanEmpty {
		t.Fatalf("expected a non-empty plan")
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if strings.Contains(output, "Running plan in Terraform Cloud") {
		t.Fatalf("unexpected TFC header in output: %s", output)
	}
	if output := done(t).Stdout(); !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("expected plan summary in output: %s", output)
	}
}

func TestCloud_planLockTimeout(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	ctx := context.Background()

	// Retrieve the workspace used to run this operation in.
	w, err := b.client.Workspaces.Read(ctx, b.organization, b.WorkspaceMapping.Name)
	if err != nil {
		t.Fatalf("error retrieving workspace: %v", err)
	}

	// Create a new configuration version.
	c, err := b.client.ConfigurationVersions.Create(ctx, w.ID, tfe.ConfigurationVersionCreateOptions{})
	if err != nil {
		t.Fatalf("error creating configuration version: %v", err)
	}

	// Create a pending run to block this run.
	_, err = b.client.Runs.Create(ctx, tfe.RunCreateOptions{
		ConfigurationVersion: c,
		Workspace:            w,
	})
	if err != nil {
		t.Fatalf("error creating pending run: %v", err)
	}

	op, configCleanup, done := testOperationPlanWithTimeout(t, "./testdata/plan", 50)
	defer configCleanup()
	defer done(t)

	input := testInput(t, map[string]string{
		"cancel":  "yes",
		"approve": "yes",
	})

	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = testBackendSingleWorkspaceName

	_, err = b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, syscall.SIGINT)
	select {
	case <-sigint:
		// Stop redirecting SIGINT signals.
		signal.Stop(sigint)
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected lock timeout after 50 milliseconds, waited 200 milliseconds")
	}

	if len(input.answers) != 2 {
		t.Fatalf("expected unused answers, got: %v", input.answers)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "Running plan in Terraform Cloud") {
		t.Fatalf("expected TFC header in output: %s", output)
	}
	if !strings.Contains(output, "Lock timeout exceeded") {
		t.Fatalf("expected lock timout error in output: %s", output)
	}
	if strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("unexpected plan summary in output: %s", output)
	}
}

func TestCloud_planDestroy(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	op, configCleanup, done := testOperationPlan(t, "./testdata/plan")
	defer configCleanup()
	defer done(t)

	op.PlanMode = plans.DestroyMode
	op.Workspace = testBackendSingleWorkspaceName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result != backend.OperationSuccess {
		t.Fatalf("operation failed: %s", b.CLI.(*cli.MockUi).ErrorWriter.String())
	}
	if run.PlanEmpty {
		t.Fatalf("expected a non-empty plan")
	}
}

func TestCloud_planDestroyNoConfig(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	op, configCleanup, done := testOperationPlan(t, "./testdata/empty")
	defer configCleanup()
	defer done(t)

	op.PlanMode = plans.DestroyMode
	op.Workspace = testBackendSingleWorkspaceName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result != backend.OperationSuccess {
		t.Fatalf("operation failed: %s", b.CLI.(*cli.MockUi).ErrorWriter.String())
	}
	if run.PlanEmpty {
		t.Fatalf("expected a non-empty plan")
	}
}

func TestCloud_planWithWorkingDirectory(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	options := tfe.WorkspaceUpdateOptions{
		WorkingDirectory: tfe.String("terraform"),
	}

	// Configure the workspace to use a custom working directory.
	_, err := b.client.Workspaces.Update(context.Background(), b.organization, b.WorkspaceMapping.Name, options)
	if err != nil {
		t.Fatalf("error configuring working directory: %v", err)
	}

	op, configCleanup, done := testOperationPlan(t, "./testdata/plan-with-working-directory/terraform")
	defer configCleanup()
	defer done(t)

	op.Workspace = testBackendSingleWorkspaceName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result != backend.OperationSuccess {
		t.Fatalf("operation failed: %s", b.CLI.(*cli.MockUi).ErrorWriter.String())
	}
	if run.PlanEmpty {
		t.Fatalf("expected a non-empty plan")
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "The remote workspace is configured to work with configuration") {
		t.Fatalf("expected working directory warning: %s", output)
	}
	if !strings.Contains(output, "Running plan in Terraform Cloud") {
		t.Fatalf("expected TFC header in output: %s", output)
	}
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("expected plan summary in output: %s", output)
	}
}

func TestCloud_planWithWorkingDirectoryFromCurrentPath(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	options := tfe.WorkspaceUpdateOptions{
		WorkingDirectory: tfe.String("terraform"),
	}

	// Configure the workspace to use a custom working directory.
	_, err := b.client.Workspaces.Update(context.Background(), b.organization, b.WorkspaceMapping.Name, options)
	if err != nil {
		t.Fatalf("error configuring working directory: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("error getting current working directory: %v", err)
	}

	// We need to change into the configuration directory to make sure
	// the logic to upload the correct slug is working as expected.
	if err := os.Chdir("./testdata/plan-with-working-directory/terraform"); err != nil {
		t.Fatalf("error changing directory: %v", err)
	}
	defer os.Chdir(wd) // Make sure we change back again when were done.

	// For this test we need to give our current directory instead of the
	// full path to the configuration as we already changed directories.
	op, configCleanup, done := testOperationPlan(t, ".")
	defer configCleanup()
	defer done(t)

	op.Workspace = testBackendSingleWorkspaceName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result != backend.OperationSuccess {
		t.Fatalf("operation failed: %s", b.CLI.(*cli.MockUi).ErrorWriter.String())
	}
	if run.PlanEmpty {
		t.Fatalf("expected a non-empty plan")
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "Running plan in Terraform Cloud") {
		t.Fatalf("expected TFC header in output: %s", output)
	}
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("expected plan summary in output: %s", output)
	}
}

func TestCloud_planCostEstimation(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	op, configCleanup, done := testOperationPlan(t, "./testdata/plan-cost-estimation")
	defer configCleanup()
	defer done(t)

	op.Workspace = testBackendSingleWorkspaceName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result != backend.OperationSuccess {
		t.Fatalf("operation failed: %s", b.CLI.(*cli.MockUi).ErrorWriter.String())
	}
	if run.PlanEmpty {
		t.Fatalf("expected a non-empty plan")
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "Running plan in Terraform Cloud") {
		t.Fatalf("expected TFC header in output: %s", output)
	}
	if !strings.Contains(output, "Resources: 1 of 1 estimated") {
		t.Fatalf("expected cost estimate result in output: %s", output)
	}
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("expected plan summary in output: %s", output)
	}
}

func TestCloud_planPolicyPass(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	op, configCleanup, done := testOperationPlan(t, "./testdata/plan-policy-passed")
	defer configCleanup()
	defer done(t)

	op.Workspace = testBackendSingleWorkspaceName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result != backend.OperationSuccess {
		t.Fatalf("operation failed: %s", b.CLI.(*cli.MockUi).ErrorWriter.String())
	}
	if run.PlanEmpty {
		t.Fatalf("expected a non-empty plan")
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "Running plan in Terraform Cloud") {
		t.Fatalf("expected TFC header in output: %s", output)
	}
	if !strings.Contains(output, "Sentinel Result: true") {
		t.Fatalf("expected policy check result in output: %s", output)
	}
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("expected plan summary in output: %s", output)
	}
}

func TestCloud_planPolicyHardFail(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	op, configCleanup, done := testOperationPlan(t, "./testdata/plan-policy-hard-failed")
	defer configCleanup()

	op.Workspace = testBackendSingleWorkspaceName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	viewOutput := done(t)
	if run.Result == backend.OperationSuccess {
		t.Fatal("expected plan operation to fail")
	}
	if !run.PlanEmpty {
		t.Fatalf("expected plan to be empty")
	}

	errOutput := viewOutput.Stderr()
	if !strings.Contains(errOutput, "hard failed") {
		t.Fatalf("expected a policy check error, got: %v", errOutput)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "Running plan in Terraform Cloud") {
		t.Fatalf("expected TFC header in output: %s", output)
	}
	if !strings.Contains(output, "Sentinel Result: false") {
		t.Fatalf("expected policy check result in output: %s", output)
	}
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("expected plan summary in output: %s", output)
	}
}

func TestCloud_planPolicySoftFail(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	op, configCleanup, done := testOperationPlan(t, "./testdata/plan-policy-soft-failed")
	defer configCleanup()

	op.Workspace = testBackendSingleWorkspaceName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	viewOutput := done(t)
	if run.Result == backend.OperationSuccess {
		t.Fatal("expected plan operation to fail")
	}
	if !run.PlanEmpty {
		t.Fatalf("expected plan to be empty")
	}

	errOutput := viewOutput.Stderr()
	if !strings.Contains(errOutput, "soft failed") {
		t.Fatalf("expected a policy check error, got: %v", errOutput)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "Running plan in Terraform Cloud") {
		t.Fatalf("expected TFC header in output: %s", output)
	}
	if !strings.Contains(output, "Sentinel Result: false") {
		t.Fatalf("expected policy check result in output: %s", output)
	}
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("expected plan summary in output: %s", output)
	}
}

func TestCloud_planWithRemoteError(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	op, configCleanup, done := testOperationPlan(t, "./testdata/plan-with-error")
	defer configCleanup()
	defer done(t)

	op.Workspace = testBackendSingleWorkspaceName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result == backend.OperationSuccess {
		t.Fatal("expected plan operation to fail")
	}
	if run.Result.ExitStatus() != 1 {
		t.Fatalf("expected exit code 1, got %d", run.Result.ExitStatus())
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "Running plan in Terraform Cloud") {
		t.Fatalf("expected TFC header in output: %s", output)
	}
	if !strings.Contains(output, "null_resource.foo: 1 error") {
		t.Fatalf("expected plan error in output: %s", output)
	}
}

func TestCloud_planJSONWithRemoteError(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	stream, close := terminal.StreamsForTesting(t)

	// Initialize the plan renderer
	b.renderer = &jsonformat.Renderer{
		Streams:  stream,
		Colorize: mockColorize(),
	}

	op, configCleanup, done := testOperationPlan(t, "./testdata/plan-json-error")
	defer configCleanup()
	defer done(t)

	op.Workspace = testBackendSingleWorkspaceName

	mockSROWorkspace(t, b, op.Workspace)

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result == backend.OperationSuccess {
		t.Fatal("expected plan operation to fail")
	}
	if run.Result.ExitStatus() != 1 {
		t.Fatalf("expected exit code 1, got %d", run.Result.ExitStatus())
	}

	outp := close(t)
	gotOut := outp.Stdout()

	if !strings.Contains(gotOut, "Unsupported block type") {
		t.Fatalf("unexpected plan error in output: %s", gotOut)
	}
}

func TestCloud_planOtherError(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	op, configCleanup, done := testOperationPlan(t, "./testdata/plan")
	defer configCleanup()
	defer done(t)

	op.Workspace = "network-error" // custom error response in backend_mock.go

	_, err := b.Operation(context.Background(), op)
	if err == nil {
		t.Errorf("expected error, got success")
	}

	if !strings.Contains(err.Error(),
		"Terraform Cloud returned an unexpected error:\n\nI'm a little teacup") {
		t.Fatalf("expected error message, got: %s", err.Error())
	}
}

func TestCloud_planImportConfigGeneration(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	stream, close := terminal.StreamsForTesting(t)

	b.renderer = &jsonformat.Renderer{
		Streams:  stream,
		Colorize: mockColorize(),
	}

	op, configCleanup, done := testOperationPlan(t, "./testdata/plan-import-config-gen")
	defer configCleanup()
	defer done(t)

	genPath := filepath.Join(op.ConfigDir, "generated.tf")
	op.GenerateConfigOut = genPath
	defer os.Remove(genPath)

	op.Workspace = testBackendSingleWorkspaceName

	mockSROWorkspace(t, b, op.Workspace)

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result != backend.OperationSuccess {
		t.Fatalf("operation failed: %s", b.CLI.(*cli.MockUi).ErrorWriter.String())
	}
	if run.PlanEmpty {
		t.Fatal("expected a non-empty plan")
	}
	outp := close(t)
	gotOut := outp.Stdout()

	if !strings.Contains(gotOut, "1 to import, 0 to add, 0 to change, 0 to destroy") {
		t.Fatalf("expected plan summary in output: %s", gotOut)
	}

	stateMgr, _ := b.StateMgr(testBackendSingleWorkspaceName)
	// An error suggests that the state was not unlocked after the operation finished
	if _, err := stateMgr.Lock(statemgr.NewLockInfo()); err != nil {
		t.Fatalf("unexpected error locking state after successful plan: %s", err.Error())
	}

	testFileEquals(t, genPath, filepath.Join(op.ConfigDir, "generated.tf.expected"))
}

func TestCloud_planImportGenerateInvalidConfig(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	stream, close := terminal.StreamsForTesting(t)

	b.renderer = &jsonformat.Renderer{
		Streams:  stream,
		Colorize: mockColorize(),
	}

	op, configCleanup, done := testOperationPlan(t, "./testdata/plan-import-config-gen-validation-error")
	defer configCleanup()
	defer done(t)

	genPath := filepath.Join(op.ConfigDir, "generated.tf")
	op.GenerateConfigOut = genPath
	defer os.Remove(genPath)

	op.Workspace = testBackendSingleWorkspaceName

	mockSROWorkspace(t, b, op.Workspace)

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result != backend.OperationFailure {
		t.Fatalf("expected operation to fail")
	}
	if run.Result.ExitStatus() != 1 {
		t.Fatalf("expected exit code 1, got %d", run.Result.ExitStatus())
	}

	outp := close(t)
	gotOut := outp.Stdout()

	if !strings.Contains(gotOut, "Conflicting configuration arguments") {
		t.Fatalf("Expected error in output: %s", gotOut)
	}

	testFileEquals(t, genPath, filepath.Join(op.ConfigDir, "generated.tf.expected"))
}

func TestCloud_planInvalidGenConfigOutPath(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	op, configCleanup, done := testOperationPlan(t, "./testdata/plan-import-config-gen-exists")
	defer configCleanup()

	genPath := filepath.Join(op.ConfigDir, "generated.tf")
	op.GenerateConfigOut = genPath

	op.Workspace = testBackendSingleWorkspaceName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	output := done(t)
	if run.Result == backend.OperationSuccess {
		t.Fatal("expected plan operation to fail")
	}

	errOutput := output.Stderr()
	if !strings.Contains(errOutput, "generated file already exists") {
		t.Fatalf("expected configuration files error, got: %v", errOutput)
	}
}

func TestCloud_planShouldRenderSRO(t *testing.T) {
	t.Run("when instance is TFC", func(t *testing.T) {
		handlers := map[string]func(http.ResponseWriter, *http.Request){
			"/api/v2/ping": func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("TFP-API-Version", "2.5")
				w.Header().Set("TFP-AppName", "Terraform Cloud")
			},
		}
		b, bCleanup := testBackendWithHandlers(t, handlers)
		t.Cleanup(bCleanup)
		b.renderer = &jsonformat.Renderer{}

		t.Run("and SRO is enabled", func(t *testing.T) {
			r := &tfe.Run{
				Workspace: &tfe.Workspace{
					StructuredRunOutputEnabled: true,
				},
			}
			assertSRORendered(t, b, r, true)
		})

		t.Run("and SRO is not enabled", func(t *testing.T) {
			r := &tfe.Run{
				Workspace: &tfe.Workspace{
					StructuredRunOutputEnabled: false,
				},
			}
			assertSRORendered(t, b, r, false)
		})

	})

	t.Run("when instance is TFE and version supports CLI SRO", func(t *testing.T) {
		handlers := map[string]func(http.ResponseWriter, *http.Request){
			"/api/v2/ping": func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("TFP-API-Version", "2.5")
				w.Header().Set("TFP-AppName", "Terraform Enterprise")
				w.Header().Set("X-TFE-Version", "v202303-1")
			},
		}
		b, bCleanup := testBackendWithHandlers(t, handlers)
		t.Cleanup(bCleanup)
		b.renderer = &jsonformat.Renderer{}

		t.Run("and SRO is enabled", func(t *testing.T) {
			r := &tfe.Run{
				Workspace: &tfe.Workspace{
					StructuredRunOutputEnabled: true,
				},
			}
			assertSRORendered(t, b, r, true)
		})

		t.Run("and SRO is not enabled", func(t *testing.T) {
			r := &tfe.Run{
				Workspace: &tfe.Workspace{
					StructuredRunOutputEnabled: false,
				},
			}
			assertSRORendered(t, b, r, false)
		})
	})

	t.Run("when instance is a known unsupported TFE release", func(t *testing.T) {
		handlers := map[string]func(http.ResponseWriter, *http.Request){
			"/api/v2/ping": func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("TFP-API-Version", "2.5")
				w.Header().Set("TFP-AppName", "Terraform Enterprise")
				w.Header().Set("X-TFE-Version", "v202208-1")
			},
		}
		b, bCleanup := testBackendWithHandlers(t, handlers)
		t.Cleanup(bCleanup)
		b.renderer = &jsonformat.Renderer{}

		r := &tfe.Run{
			Workspace: &tfe.Workspace{
				StructuredRunOutputEnabled: true,
			},
		}
		assertSRORendered(t, b, r, false)
	})

	t.Run("when instance is an unknown TFE release", func(t *testing.T) {
		handlers := map[string]func(http.ResponseWriter, *http.Request){
			"/api/v2/ping": func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("TFP-API-Version", "2.5")
			},
		}
		b, bCleanup := testBackendWithHandlers(t, handlers)
		t.Cleanup(bCleanup)
		b.renderer = &jsonformat.Renderer{}

		r := &tfe.Run{
			Workspace: &tfe.Workspace{
				StructuredRunOutputEnabled: true,
			},
		}
		assertSRORendered(t, b, r, false)
	})

}

func assertSRORendered(t *testing.T, b *Cloud, r *tfe.Run, shouldRender bool) {
	got, err := b.shouldRenderStructuredRunOutput(r)
	if err != nil {
		t.Fatalf("expected no error: %v", err)
	}
	if shouldRender != got {
		t.Fatalf("expected SRO to be rendered: %t, got %t", shouldRender, got)
	}
}

func testFileEquals(t *testing.T, got, want string) {
	t.Helper()

	actual, err := os.ReadFile(got)
	if err != nil {
		t.Fatalf("error reading %s", got)
	}

	expected, err := os.ReadFile(want)
	if err != nil {
		t.Fatalf("error reading %s", want)
	}

	if diff := cmp.Diff(string(actual), string(expected)); len(diff) > 0 {
		t.Fatalf("got:\n%s\nwant:\n%s\ndiff:\n%s", actual, expected, diff)
	}
}
