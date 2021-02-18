package remote

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	tfe "github.com/hashicorp/go-tfe"
	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/command/arguments"
	"github.com/hashicorp/terraform/command/clistate"
	"github.com/hashicorp/terraform/command/views"
	"github.com/hashicorp/terraform/internal/initwd"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/plans/planfile"
	"github.com/hashicorp/terraform/states/statemgr"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
	tfversion "github.com/hashicorp/terraform/version"
	"github.com/mitchellh/cli"
)

func testOperationApply(t *testing.T, configDir string) (*backend.Operation, func()) {
	t.Helper()

	return testOperationApplyWithTimeout(t, configDir, 0)
}

func testOperationApplyWithTimeout(t *testing.T, configDir string, timeout time.Duration) (*backend.Operation, func()) {
	t.Helper()

	_, configLoader, configCleanup := initwd.MustLoadConfigForTests(t, configDir)

	streams, _ := terminal.StreamsForTesting(t)
	view := views.NewStateLocker(arguments.ViewHuman, views.NewView(streams))

	return &backend.Operation{
		ConfigDir:       configDir,
		ConfigLoader:    configLoader,
		Parallelism:     defaultParallelism,
		PlanRefresh:     true,
		ShowDiagnostics: testLogDiagnostics(t),
		StateLocker:     clistate.NewLocker(timeout, view),
		Type:            backend.OperationTypeApply,
	}, configCleanup
}

func testOperationApplyWithDiagnostics(t *testing.T, configDir string) (*backend.Operation, func(), func() tfdiags.Diagnostics) {
	t.Helper()

	op, cleanup := testOperationApply(t, configDir)

	record, playback := testRecordDiagnostics(t)
	op.ShowDiagnostics = record

	return op, cleanup, playback
}

func TestRemote_applyBasic(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	op, configCleanup := testOperationApply(t, "./testdata/apply")
	defer configCleanup()

	input := testInput(t, map[string]string{
		"approve": "yes",
	})

	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = backend.DefaultStateName

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

	if len(input.answers) > 0 {
		t.Fatalf("expected no unused answers, got: %v", input.answers)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "Running apply in the remote backend") {
		t.Fatalf("expected remote backend header in output: %s", output)
	}
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("expected plan summery in output: %s", output)
	}
	if !strings.Contains(output, "1 added, 0 changed, 0 destroyed") {
		t.Fatalf("expected apply summery in output: %s", output)
	}

	stateMgr, _ := b.StateMgr(backend.DefaultStateName)
	// An error suggests that the state was not unlocked after apply
	if _, err := stateMgr.Lock(statemgr.NewLockInfo()); err != nil {
		t.Fatalf("unexpected error locking state after apply: %s", err.Error())
	}
}

func TestRemote_applyCanceled(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	op, configCleanup := testOperationApply(t, "./testdata/apply")
	defer configCleanup()

	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	// Stop the run to simulate a Ctrl-C.
	run.Stop()

	<-run.Done()
	if run.Result == backend.OperationSuccess {
		t.Fatal("expected apply operation to fail")
	}

	stateMgr, _ := b.StateMgr(backend.DefaultStateName)
	if _, err := stateMgr.Lock(statemgr.NewLockInfo()); err != nil {
		t.Fatalf("unexpected error locking state after cancelling apply: %s", err.Error())
	}
}

func TestRemote_applyWithoutPermissions(t *testing.T) {
	b, bCleanup := testBackendNoDefault(t)
	defer bCleanup()

	// Create a named workspace without permissions.
	w, err := b.client.Workspaces.Create(
		context.Background(),
		b.organization,
		tfe.WorkspaceCreateOptions{
			Name: tfe.String(b.prefix + "prod"),
		},
	)
	if err != nil {
		t.Fatalf("error creating named workspace: %v", err)
	}
	w.Permissions.CanQueueApply = false

	op, configCleanup, playback := testOperationApplyWithDiagnostics(t, "./testdata/apply")
	defer configCleanup()

	op.UIOut = b.CLI
	op.Workspace = "prod"

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result == backend.OperationSuccess {
		t.Fatal("expected apply operation to fail")
	}

	errOutput := playback().Err().Error()
	if !strings.Contains(errOutput, "Insufficient rights to apply changes") {
		t.Fatalf("expected a permissions error, got: %v", errOutput)
	}
}

func TestRemote_applyWithVCS(t *testing.T) {
	b, bCleanup := testBackendNoDefault(t)
	defer bCleanup()

	// Create a named workspace with a VCS.
	_, err := b.client.Workspaces.Create(
		context.Background(),
		b.organization,
		tfe.WorkspaceCreateOptions{
			Name:    tfe.String(b.prefix + "prod"),
			VCSRepo: &tfe.VCSRepoOptions{},
		},
	)
	if err != nil {
		t.Fatalf("error creating named workspace: %v", err)
	}

	op, configCleanup, playback := testOperationApplyWithDiagnostics(t, "./testdata/apply")
	defer configCleanup()

	op.Workspace = "prod"

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result == backend.OperationSuccess {
		t.Fatal("expected apply operation to fail")
	}
	if !run.PlanEmpty {
		t.Fatalf("expected plan to be empty")
	}

	errOutput := playback().Err().Error()
	if !strings.Contains(errOutput, "not allowed for workspaces with a VCS") {
		t.Fatalf("expected a VCS error, got: %v", errOutput)
	}
}

func TestRemote_applyWithParallelism(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	op, configCleanup, playback := testOperationApplyWithDiagnostics(t, "./testdata/apply")
	defer configCleanup()

	op.Parallelism = 3
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result == backend.OperationSuccess {
		t.Fatal("expected apply operation to fail")
	}

	errOutput := playback().Err().Error()
	if !strings.Contains(errOutput, "parallelism values are currently not supported") {
		t.Fatalf("expected a parallelism error, got: %v", errOutput)
	}
}

func TestRemote_applyWithPlan(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	op, configCleanup, playback := testOperationApplyWithDiagnostics(t, "./testdata/apply")
	defer configCleanup()

	op.PlanFile = &planfile.Reader{}
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result == backend.OperationSuccess {
		t.Fatal("expected apply operation to fail")
	}
	if !run.PlanEmpty {
		t.Fatalf("expected plan to be empty")
	}

	errOutput := playback().Err().Error()
	if !strings.Contains(errOutput, "saved plan is currently not supported") {
		t.Fatalf("expected a saved plan error, got: %v", errOutput)
	}
}

func TestRemote_applyWithoutRefresh(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	op, configCleanup, playback := testOperationApplyWithDiagnostics(t, "./testdata/apply")
	defer configCleanup()

	op.PlanRefresh = false
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result == backend.OperationSuccess {
		t.Fatal("expected apply operation to fail")
	}

	errOutput := playback().Err().Error()
	if !strings.Contains(errOutput, "refresh is currently not supported") {
		t.Fatalf("expected a refresh error, got: %v", errOutput)
	}
}

func TestRemote_applyWithTarget(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	op, configCleanup := testOperationApply(t, "./testdata/apply")
	defer configCleanup()

	addr, _ := addrs.ParseAbsResourceStr("null_resource.foo")

	op.Targets = []addrs.Targetable{addr}
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result != backend.OperationSuccess {
		t.Fatal("expected apply operation to succeed")
	}
	if run.PlanEmpty {
		t.Fatalf("expected plan to be non-empty")
	}

	// We should find a run inside the mock client that has the same
	// target address we requested above.
	runsAPI := b.client.Runs.(*mockRuns)
	if got, want := len(runsAPI.runs), 1; got != want {
		t.Fatalf("wrong number of runs in the mock client %d; want %d", got, want)
	}
	for _, run := range runsAPI.runs {
		if diff := cmp.Diff([]string{"null_resource.foo"}, run.TargetAddrs); diff != "" {
			t.Errorf("wrong TargetAddrs in the created run\n%s", diff)
		}
	}
}

func TestRemote_applyWithTargetIncompatibleAPIVersion(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	op, configCleanup, playback := testOperationPlanWithDiagnostics(t, "./testdata/plan")
	defer configCleanup()

	// Set the tfe client's RemoteAPIVersion to an empty string, to mimic
	// API versions prior to 2.3.
	b.client.SetFakeRemoteAPIVersion("")

	addr, _ := addrs.ParseAbsResourceStr("null_resource.foo")

	op.Targets = []addrs.Targetable{addr}
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result == backend.OperationSuccess {
		t.Fatal("expected apply operation to fail")
	}
	if !run.PlanEmpty {
		t.Fatalf("expected plan to be empty")
	}

	errOutput := playback().Err().Error()
	if !strings.Contains(errOutput, "Resource targeting is not supported") {
		t.Fatalf("expected a targeting error, got: %v", errOutput)
	}
}

func TestRemote_applyWithVariables(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	op, configCleanup, playback := testOperationApplyWithDiagnostics(t, "./testdata/apply-variables")
	defer configCleanup()

	op.Variables = testVariables(terraform.ValueFromNamedFile, "foo", "bar")
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result == backend.OperationSuccess {
		t.Fatal("expected apply operation to fail")
	}

	errOutput := playback().Err().Error()
	if !strings.Contains(errOutput, "variables are currently not supported") {
		t.Fatalf("expected a variables error, got: %v", errOutput)
	}
}

func TestRemote_applyNoConfig(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	op, configCleanup, playback := testOperationApplyWithDiagnostics(t, "./testdata/empty")
	defer configCleanup()

	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result == backend.OperationSuccess {
		t.Fatal("expected apply operation to fail")
	}
	if !run.PlanEmpty {
		t.Fatalf("expected plan to be empty")
	}

	errOutput := playback().Err().Error()
	if !strings.Contains(errOutput, "configuration files found") {
		t.Fatalf("expected configuration files error, got: %v", errOutput)
	}

	stateMgr, _ := b.StateMgr(backend.DefaultStateName)
	// An error suggests that the state was not unlocked after apply
	if _, err := stateMgr.Lock(statemgr.NewLockInfo()); err != nil {
		t.Fatalf("unexpected error locking state after failed apply: %s", err.Error())
	}
}

func TestRemote_applyNoChanges(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	op, configCleanup := testOperationApply(t, "./testdata/apply-no-changes")
	defer configCleanup()

	op.Workspace = backend.DefaultStateName

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
		t.Fatalf("expected no changes in plan summery: %s", output)
	}
	if !strings.Contains(output, "Sentinel Result: true") {
		t.Fatalf("expected policy check result in output: %s", output)
	}
}

func TestRemote_applyNoApprove(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	op, configCleanup, playback := testOperationApplyWithDiagnostics(t, "./testdata/apply")
	defer configCleanup()

	input := testInput(t, map[string]string{
		"approve": "no",
	})

	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result == backend.OperationSuccess {
		t.Fatal("expected apply operation to fail")
	}
	if !run.PlanEmpty {
		t.Fatalf("expected plan to be empty")
	}

	if len(input.answers) > 0 {
		t.Fatalf("expected no unused answers, got: %v", input.answers)
	}

	errOutput := playback().Err().Error()
	if !strings.Contains(errOutput, "Apply discarded") {
		t.Fatalf("expected an apply discarded error, got: %v", errOutput)
	}
}

func TestRemote_applyAutoApprove(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	op, configCleanup := testOperationApply(t, "./testdata/apply")
	defer configCleanup()

	input := testInput(t, map[string]string{
		"approve": "no",
	})

	op.AutoApprove = true
	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = backend.DefaultStateName

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

	if len(input.answers) != 1 {
		t.Fatalf("expected an unused answer, got: %v", input.answers)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "Running apply in the remote backend") {
		t.Fatalf("expected remote backend header in output: %s", output)
	}
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("expected plan summery in output: %s", output)
	}
	if !strings.Contains(output, "1 added, 0 changed, 0 destroyed") {
		t.Fatalf("expected apply summery in output: %s", output)
	}
}

func TestRemote_applyApprovedExternally(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	op, configCleanup := testOperationApply(t, "./testdata/apply")
	defer configCleanup()

	input := testInput(t, map[string]string{
		"approve": "wait-for-external-update",
	})

	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = backend.DefaultStateName

	ctx := context.Background()

	run, err := b.Operation(ctx, op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	// Wait 50 milliseconds to make sure the run started.
	time.Sleep(50 * time.Millisecond)

	wl, err := b.client.Workspaces.List(
		ctx,
		b.organization,
		tfe.WorkspaceListOptions{
			ListOptions: tfe.ListOptions{PageNumber: 2, PageSize: 10},
		},
	)
	if err != nil {
		t.Fatalf("unexpected error listing workspaces: %v", err)
	}
	if len(wl.Items) != 1 {
		t.Fatalf("expected 1 workspace, got %d workspaces", len(wl.Items))
	}

	rl, err := b.client.Runs.List(ctx, wl.Items[0].ID, tfe.RunListOptions{})
	if err != nil {
		t.Fatalf("unexpected error listing runs: %v", err)
	}
	if len(rl.Items) != 1 {
		t.Fatalf("expected 1 run, got %d runs", len(rl.Items))
	}

	err = b.client.Runs.Apply(context.Background(), rl.Items[0].ID, tfe.RunApplyOptions{})
	if err != nil {
		t.Fatalf("unexpected error approving run: %v", err)
	}

	<-run.Done()
	if run.Result != backend.OperationSuccess {
		t.Fatalf("operation failed: %s", b.CLI.(*cli.MockUi).ErrorWriter.String())
	}
	if run.PlanEmpty {
		t.Fatalf("expected a non-empty plan")
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "Running apply in the remote backend") {
		t.Fatalf("expected remote backend header in output: %s", output)
	}
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("expected plan summery in output: %s", output)
	}
	if !strings.Contains(output, "approved using the UI or API") {
		t.Fatalf("expected external approval in output: %s", output)
	}
	if !strings.Contains(output, "1 added, 0 changed, 0 destroyed") {
		t.Fatalf("expected apply summery in output: %s", output)
	}
}

func TestRemote_applyDiscardedExternally(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	op, configCleanup := testOperationApply(t, "./testdata/apply")
	defer configCleanup()

	input := testInput(t, map[string]string{
		"approve": "wait-for-external-update",
	})

	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = backend.DefaultStateName

	ctx := context.Background()

	run, err := b.Operation(ctx, op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	// Wait 50 milliseconds to make sure the run started.
	time.Sleep(50 * time.Millisecond)

	wl, err := b.client.Workspaces.List(
		ctx,
		b.organization,
		tfe.WorkspaceListOptions{
			ListOptions: tfe.ListOptions{PageNumber: 2, PageSize: 10},
		},
	)
	if err != nil {
		t.Fatalf("unexpected error listing workspaces: %v", err)
	}
	if len(wl.Items) != 1 {
		t.Fatalf("expected 1 workspace, got %d workspaces", len(wl.Items))
	}

	rl, err := b.client.Runs.List(ctx, wl.Items[0].ID, tfe.RunListOptions{})
	if err != nil {
		t.Fatalf("unexpected error listing runs: %v", err)
	}
	if len(rl.Items) != 1 {
		t.Fatalf("expected 1 run, got %d runs", len(rl.Items))
	}

	err = b.client.Runs.Discard(context.Background(), rl.Items[0].ID, tfe.RunDiscardOptions{})
	if err != nil {
		t.Fatalf("unexpected error discarding run: %v", err)
	}

	<-run.Done()
	if run.Result == backend.OperationSuccess {
		t.Fatal("expected apply operation to fail")
	}
	if !run.PlanEmpty {
		t.Fatalf("expected plan to be empty")
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "Running apply in the remote backend") {
		t.Fatalf("expected remote backend header in output: %s", output)
	}
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("expected plan summery in output: %s", output)
	}
	if !strings.Contains(output, "discarded using the UI or API") {
		t.Fatalf("expected external discard output: %s", output)
	}
	if strings.Contains(output, "1 added, 0 changed, 0 destroyed") {
		t.Fatalf("unexpected apply summery in output: %s", output)
	}
}

func TestRemote_applyWithAutoApply(t *testing.T) {
	b, bCleanup := testBackendNoDefault(t)
	defer bCleanup()

	// Create a named workspace that auto applies.
	_, err := b.client.Workspaces.Create(
		context.Background(),
		b.organization,
		tfe.WorkspaceCreateOptions{
			AutoApply: tfe.Bool(true),
			Name:      tfe.String(b.prefix + "prod"),
		},
	)
	if err != nil {
		t.Fatalf("error creating named workspace: %v", err)
	}

	op, configCleanup := testOperationApply(t, "./testdata/apply")
	defer configCleanup()

	input := testInput(t, map[string]string{
		"approve": "yes",
	})

	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = "prod"

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

	if len(input.answers) != 1 {
		t.Fatalf("expected an unused answer, got: %v", input.answers)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "Running apply in the remote backend") {
		t.Fatalf("expected remote backend header in output: %s", output)
	}
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("expected plan summery in output: %s", output)
	}
	if !strings.Contains(output, "1 added, 0 changed, 0 destroyed") {
		t.Fatalf("expected apply summery in output: %s", output)
	}
}

func TestRemote_applyForceLocal(t *testing.T) {
	// Set TF_FORCE_LOCAL_BACKEND so the remote backend will use
	// the local backend with itself as embedded backend.
	if err := os.Setenv("TF_FORCE_LOCAL_BACKEND", "1"); err != nil {
		t.Fatalf("error setting environment variable TF_FORCE_LOCAL_BACKEND: %v", err)
	}
	defer os.Unsetenv("TF_FORCE_LOCAL_BACKEND")

	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	op, configCleanup := testOperationApply(t, "./testdata/apply")
	defer configCleanup()

	input := testInput(t, map[string]string{
		"approve": "yes",
	})

	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = backend.DefaultStateName

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

	if len(input.answers) > 0 {
		t.Fatalf("expected no unused answers, got: %v", input.answers)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if strings.Contains(output, "Running apply in the remote backend") {
		t.Fatalf("unexpected remote backend header in output: %s", output)
	}
	if output := done(t).Stdout(); !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("expected plan summary in output: %s", output)
	}
	if !run.State.HasResources() {
		t.Fatalf("expected resources in state")
	}
}

func TestRemote_applyWorkspaceWithoutOperations(t *testing.T) {
	b, bCleanup := testBackendNoDefault(t)
	defer bCleanup()

	ctx := context.Background()

	// Create a named workspace that doesn't allow operations.
	_, err := b.client.Workspaces.Create(
		ctx,
		b.organization,
		tfe.WorkspaceCreateOptions{
			Name: tfe.String(b.prefix + "no-operations"),
		},
	)
	if err != nil {
		t.Fatalf("error creating named workspace: %v", err)
	}

	op, configCleanup := testOperationApply(t, "./testdata/apply")
	defer configCleanup()

	input := testInput(t, map[string]string{
		"approve": "yes",
	})

	op.UIIn = input
	op.UIOut = b.CLI
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

	if len(input.answers) > 0 {
		t.Fatalf("expected no unused answers, got: %v", input.answers)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if strings.Contains(output, "Running apply in the remote backend") {
		t.Fatalf("unexpected remote backend header in output: %s", output)
	}
	if output := done(t).Stdout(); !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("expected plan summary in output: %s", output)
	}
	if !run.State.HasResources() {
		t.Fatalf("expected resources in state")
	}
}

func TestRemote_applyLockTimeout(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	ctx := context.Background()

	// Retrieve the workspace used to run this operation in.
	w, err := b.client.Workspaces.Read(ctx, b.organization, b.workspace)
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

	op, configCleanup := testOperationApplyWithTimeout(t, "./testdata/apply", 50*time.Millisecond)
	defer configCleanup()

	input := testInput(t, map[string]string{
		"cancel":  "yes",
		"approve": "yes",
	})

	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = backend.DefaultStateName

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
	if !strings.Contains(output, "Running apply in the remote backend") {
		t.Fatalf("expected remote backend header in output: %s", output)
	}
	if !strings.Contains(output, "Lock timeout exceeded") {
		t.Fatalf("expected lock timout error in output: %s", output)
	}
	if strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("unexpected plan summery in output: %s", output)
	}
	if strings.Contains(output, "1 added, 0 changed, 0 destroyed") {
		t.Fatalf("unexpected apply summery in output: %s", output)
	}
}

func TestRemote_applyDestroy(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	op, configCleanup := testOperationApply(t, "./testdata/apply-destroy")
	defer configCleanup()

	input := testInput(t, map[string]string{
		"approve": "yes",
	})

	op.Destroy = true
	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = backend.DefaultStateName

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

	if len(input.answers) > 0 {
		t.Fatalf("expected no unused answers, got: %v", input.answers)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "Running apply in the remote backend") {
		t.Fatalf("expected remote backend header in output: %s", output)
	}
	if !strings.Contains(output, "0 to add, 0 to change, 1 to destroy") {
		t.Fatalf("expected plan summery in output: %s", output)
	}
	if !strings.Contains(output, "0 added, 0 changed, 1 destroyed") {
		t.Fatalf("expected apply summery in output: %s", output)
	}
}

func TestRemote_applyDestroyNoConfig(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	input := testInput(t, map[string]string{
		"approve": "yes",
	})

	op, configCleanup := testOperationApply(t, "./testdata/empty")
	defer configCleanup()

	op.Destroy = true
	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = backend.DefaultStateName

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

	if len(input.answers) > 0 {
		t.Fatalf("expected no unused answers, got: %v", input.answers)
	}
}

func TestRemote_applyPolicyPass(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	op, configCleanup := testOperationApply(t, "./testdata/apply-policy-passed")
	defer configCleanup()

	input := testInput(t, map[string]string{
		"approve": "yes",
	})

	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = backend.DefaultStateName

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

	if len(input.answers) > 0 {
		t.Fatalf("expected no unused answers, got: %v", input.answers)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "Running apply in the remote backend") {
		t.Fatalf("expected remote backend header in output: %s", output)
	}
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("expected plan summery in output: %s", output)
	}
	if !strings.Contains(output, "Sentinel Result: true") {
		t.Fatalf("expected policy check result in output: %s", output)
	}
	if !strings.Contains(output, "1 added, 0 changed, 0 destroyed") {
		t.Fatalf("expected apply summery in output: %s", output)
	}
}

func TestRemote_applyPolicyHardFail(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	op, configCleanup, playback := testOperationApplyWithDiagnostics(t, "./testdata/apply-policy-hard-failed")
	defer configCleanup()

	input := testInput(t, map[string]string{
		"approve": "yes",
	})

	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result == backend.OperationSuccess {
		t.Fatal("expected apply operation to fail")
	}
	if !run.PlanEmpty {
		t.Fatalf("expected plan to be empty")
	}

	if len(input.answers) != 1 {
		t.Fatalf("expected an unused answers, got: %v", input.answers)
	}

	errOutput := playback().Err().Error()
	if !strings.Contains(errOutput, "hard failed") {
		t.Fatalf("expected a policy check error, got: %v", errOutput)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "Running apply in the remote backend") {
		t.Fatalf("expected remote backend header in output: %s", output)
	}
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("expected plan summery in output: %s", output)
	}
	if !strings.Contains(output, "Sentinel Result: false") {
		t.Fatalf("expected policy check result in output: %s", output)
	}
	if strings.Contains(output, "1 added, 0 changed, 0 destroyed") {
		t.Fatalf("unexpected apply summery in output: %s", output)
	}
}

func TestRemote_applyPolicySoftFail(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	op, configCleanup := testOperationApply(t, "./testdata/apply-policy-soft-failed")
	defer configCleanup()

	input := testInput(t, map[string]string{
		"override": "override",
		"approve":  "yes",
	})

	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = backend.DefaultStateName

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

	if len(input.answers) > 0 {
		t.Fatalf("expected no unused answers, got: %v", input.answers)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "Running apply in the remote backend") {
		t.Fatalf("expected remote backend header in output: %s", output)
	}
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("expected plan summery in output: %s", output)
	}
	if !strings.Contains(output, "Sentinel Result: false") {
		t.Fatalf("expected policy check result in output: %s", output)
	}
	if !strings.Contains(output, "1 added, 0 changed, 0 destroyed") {
		t.Fatalf("expected apply summery in output: %s", output)
	}
}

func TestRemote_applyPolicySoftFailAutoApprove(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	op, configCleanup, playback := testOperationApplyWithDiagnostics(t, "./testdata/apply-policy-soft-failed")
	defer configCleanup()

	input := testInput(t, map[string]string{
		"override": "override",
	})

	op.AutoApprove = true
	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result == backend.OperationSuccess {
		t.Fatal("expected apply operation to fail")
	}
	if !run.PlanEmpty {
		t.Fatalf("expected plan to be empty")
	}

	if len(input.answers) != 1 {
		t.Fatalf("expected an unused answers, got: %v", input.answers)
	}

	errOutput := playback().Err().Error()
	if !strings.Contains(errOutput, "soft failed") {
		t.Fatalf("expected a policy check error, got: %v", errOutput)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "Running apply in the remote backend") {
		t.Fatalf("expected remote backend header in output: %s", output)
	}
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("expected plan summery in output: %s", output)
	}
	if !strings.Contains(output, "Sentinel Result: false") {
		t.Fatalf("expected policy check result in output: %s", output)
	}
	if strings.Contains(output, "1 added, 0 changed, 0 destroyed") {
		t.Fatalf("unexpected apply summery in output: %s", output)
	}
}

func TestRemote_applyPolicySoftFailAutoApply(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	// Create a named workspace that auto applies.
	_, err := b.client.Workspaces.Create(
		context.Background(),
		b.organization,
		tfe.WorkspaceCreateOptions{
			AutoApply: tfe.Bool(true),
			Name:      tfe.String(b.prefix + "prod"),
		},
	)
	if err != nil {
		t.Fatalf("error creating named workspace: %v", err)
	}

	op, configCleanup := testOperationApply(t, "./testdata/apply-policy-soft-failed")
	defer configCleanup()

	input := testInput(t, map[string]string{
		"override": "override",
		"approve":  "yes",
	})

	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = "prod"

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

	if len(input.answers) != 1 {
		t.Fatalf("expected an unused answer, got: %v", input.answers)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "Running apply in the remote backend") {
		t.Fatalf("expected remote backend header in output: %s", output)
	}
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("expected plan summery in output: %s", output)
	}
	if !strings.Contains(output, "Sentinel Result: false") {
		t.Fatalf("expected policy check result in output: %s", output)
	}
	if !strings.Contains(output, "1 added, 0 changed, 0 destroyed") {
		t.Fatalf("expected apply summery in output: %s", output)
	}
}

func TestRemote_applyWithRemoteError(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	op, configCleanup := testOperationApply(t, "./testdata/apply-with-error")
	defer configCleanup()

	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result == backend.OperationSuccess {
		t.Fatal("expected apply operation to fail")
	}
	if run.Result.ExitStatus() != 1 {
		t.Fatalf("expected exit code 1, got %d", run.Result.ExitStatus())
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "null_resource.foo: 1 error") {
		t.Fatalf("expected apply error in output: %s", output)
	}
}

func TestRemote_applyVersionCheck(t *testing.T) {
	testCases := map[string]struct {
		localVersion  string
		remoteVersion string
		forceLocal    bool
		hasOperations bool
		wantErr       string
	}{
		"versions can be different for remote apply": {
			localVersion:  "0.14.0",
			remoteVersion: "0.13.5",
			hasOperations: true,
		},
		"versions can be different for local apply": {
			localVersion:  "0.14.0",
			remoteVersion: "0.13.5",
			hasOperations: false,
		},
		"force local with remote operations and different versions is acceptable": {
			localVersion:  "0.14.0",
			remoteVersion: "0.14.0-acme-provider-bundle",
			forceLocal:    true,
			hasOperations: true,
		},
		"no error if versions are identical": {
			localVersion:  "0.14.0",
			remoteVersion: "0.14.0",
			forceLocal:    true,
			hasOperations: true,
		},
		"no error if force local but workspace has remote operations disabled": {
			localVersion:  "0.14.0",
			remoteVersion: "0.13.5",
			forceLocal:    true,
			hasOperations: false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			b, bCleanup := testBackendDefault(t)
			defer bCleanup()

			// SETUP: Save original local version state and restore afterwards
			p := tfversion.Prerelease
			v := tfversion.Version
			s := tfversion.SemVer
			defer func() {
				tfversion.Prerelease = p
				tfversion.Version = v
				tfversion.SemVer = s
			}()

			// SETUP: Set local version for the test case
			tfversion.Prerelease = ""
			tfversion.Version = tc.localVersion
			tfversion.SemVer = version.Must(version.NewSemver(tc.localVersion))

			// SETUP: Set force local for the test case
			b.forceLocal = tc.forceLocal

			ctx := context.Background()

			// SETUP: set the operations and Terraform Version fields on the
			// remote workspace
			_, err := b.client.Workspaces.Update(
				ctx,
				b.organization,
				b.workspace,
				tfe.WorkspaceUpdateOptions{
					Operations:       tfe.Bool(tc.hasOperations),
					TerraformVersion: tfe.String(tc.remoteVersion),
				},
			)
			if err != nil {
				t.Fatalf("error creating named workspace: %v", err)
			}

			// RUN: prepare the apply operation and run it
			op, configCleanup, playback := testOperationApplyWithDiagnostics(t, "./testdata/apply")
			defer configCleanup()

			streams, done := terminal.StreamsForTesting(t)
			view := views.NewOperation(arguments.ViewHuman, false, views.NewView(streams))
			op.View = view
			defer done(t)

			input := testInput(t, map[string]string{
				"approve": "yes",
			})

			op.UIIn = input
			op.UIOut = b.CLI
			op.Workspace = backend.DefaultStateName

			run, err := b.Operation(ctx, op)
			if err != nil {
				t.Fatalf("error starting operation: %v", err)
			}

			// RUN: wait for completion
			<-run.Done()

			if tc.wantErr != "" {
				// ASSERT: if the test case wants an error, check for failure
				// and the error message
				if run.Result != backend.OperationFailure {
					t.Fatalf("expected run to fail, but result was %#v", run.Result)
				}
				errOutput := playback().Err().Error()
				if !strings.Contains(errOutput, tc.wantErr) {
					t.Fatalf("missing error %q\noutput: %s", tc.wantErr, errOutput)
				}
			} else {
				// ASSERT: otherwise, check for success and appropriate output
				// based on whether the run should be local or remote
				if run.Result != backend.OperationSuccess {
					t.Fatalf("operation failed: %s", b.CLI.(*cli.MockUi).ErrorWriter.String())
				}
				output := b.CLI.(*cli.MockUi).OutputWriter.String()
				hasRemote := strings.Contains(output, "Running apply in the remote backend")
				hasSummary := strings.Contains(output, "1 added, 0 changed, 0 destroyed")
				hasResources := run.State.HasResources()
				if !tc.forceLocal && tc.hasOperations {
					if !hasRemote {
						t.Errorf("missing remote backend header in output: %s", output)
					}
					if !hasSummary {
						t.Errorf("expected apply summary in output: %s", output)
					}
				} else {
					if hasRemote {
						t.Errorf("unexpected remote backend header in output: %s", output)
					}
					if !hasResources {
						t.Errorf("expected resources in state")
					}
				}
			}
		})
	}
}
