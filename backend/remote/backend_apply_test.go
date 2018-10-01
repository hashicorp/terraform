package remote

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

func testOperationApply() *backend.Operation {
	return &backend.Operation{
		Type: backend.OperationTypeApply,
	}
}

func TestRemote_applyBasic(t *testing.T) {
	b := testBackendDefault(t)

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply")
	defer modCleanup()

	input := testInput(t, map[string]string{
		"approve": "yes",
	})

	op := testOperationApply()
	op.Module = mod
	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err != nil {
		t.Fatalf("error running operation: %v", run.Err)
	}

	if len(input.answers) > 0 {
		t.Fatalf("expected no unused answers, got: %v", input.answers)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("missing plan summery in output: %s", output)
	}
	if !strings.Contains(output, "1 added, 0 changed, 0 destroyed") {
		t.Fatalf("missing apply summery in output: %s", output)
	}
}

func TestRemote_applyWithVCS(t *testing.T) {
	b := testBackendNoDefault(t)

	// Create the named workspace with a VCS.
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

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply")
	defer modCleanup()

	op := testOperationApply()
	op.Module = mod
	op.Workspace = "prod"

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}
	<-run.Done()

	if run.Err == nil {
		t.Fatalf("expected an apply error, got: %v", run.Err)
	}
	if !strings.Contains(run.Err.Error(), "not allowed for workspaces with a VCS") {
		t.Fatalf("expected a VCS error, got: %v", run.Err)
	}
}

func TestRemote_applyWithPlan(t *testing.T) {
	b := testBackendDefault(t)

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply")
	defer modCleanup()

	op := testOperationApply()
	op.Module = mod
	op.Plan = &terraform.Plan{}
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}
	<-run.Done()

	if run.Err == nil {
		t.Fatalf("expected an apply error, got: %v", run.Err)
	}
	if !strings.Contains(run.Err.Error(), "saved plan is currently not supported") {
		t.Fatalf("expected a saved plan error, got: %v", run.Err)
	}
}

func TestRemote_applyWithTarget(t *testing.T) {
	b := testBackendDefault(t)

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply")
	defer modCleanup()

	op := testOperationApply()
	op.Module = mod
	op.Targets = []string{"null_resource.foo"}
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}
	<-run.Done()

	if run.Err == nil {
		t.Fatalf("expected an apply error, got: %v", run.Err)
	}
	if !strings.Contains(run.Err.Error(), "targeting is currently not supported") {
		t.Fatalf("expected a targeting error, got: %v", run.Err)
	}
}

func TestRemote_applyNoConfig(t *testing.T) {
	b := testBackendDefault(t)

	op := testOperationApply()
	op.Module = nil
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}
	<-run.Done()

	if run.Err == nil {
		t.Fatalf("expected an apply error, got: %v", run.Err)
	}
	if !strings.Contains(run.Err.Error(), "configuration files found") {
		t.Fatalf("expected configuration files error, got: %v", run.Err)
	}
}

func TestRemote_applyNoChanges(t *testing.T) {
	b := testBackendDefault(t)

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply-no-changes")
	defer modCleanup()

	op := testOperationApply()
	op.Module = mod
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err != nil {
		t.Fatalf("error running operation: %v", run.Err)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "No changes. Infrastructure is up-to-date.") {
		t.Fatalf("expected no changes in plan summery: %s", output)
	}
}

func TestRemote_applyNoApprove(t *testing.T) {
	b := testBackendDefault(t)

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply")
	defer modCleanup()

	input := testInput(t, map[string]string{
		"approve": "no",
	})

	op := testOperationApply()
	op.Module = mod
	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err == nil {
		t.Fatalf("expected an apply error, got: %v", run.Err)
	}
	if !strings.Contains(run.Err.Error(), "Apply discarded") {
		t.Fatalf("expected an apply discarded error, got: %v", run.Err)
	}
	if len(input.answers) > 0 {
		t.Fatalf("expected no unused answers, got: %v", input.answers)
	}
}

func TestRemote_applyAutoApprove(t *testing.T) {
	b := testBackendDefault(t)

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply")
	defer modCleanup()

	input := testInput(t, map[string]string{
		"approve": "no",
	})

	op := testOperationApply()
	op.AutoApprove = true
	op.Module = mod
	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err != nil {
		t.Fatalf("error running operation: %v", run.Err)
	}

	if len(input.answers) != 1 {
		t.Fatalf("expected an unused answer, got: %v", input.answers)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("missing plan summery in output: %s", output)
	}
	if !strings.Contains(output, "1 added, 0 changed, 0 destroyed") {
		t.Fatalf("missing apply summery in output: %s", output)
	}
}

func TestRemote_applyLockTimeout(t *testing.T) {
	b := testBackendDefault(t)
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

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply")
	defer modCleanup()

	input := testInput(t, map[string]string{
		"cancel":  "yes",
		"approve": "yes",
	})

	op := testOperationApply()
	op.StateLockTimeout = 5 * time.Second
	op.Module = mod
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
	case <-time.After(10 * time.Second):
		t.Fatalf("expected lock timeout after 5 seconds, waited 10 seconds")
	}

	if len(input.answers) != 2 {
		t.Fatalf("expected unused answers, got: %v", input.answers)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "Lock timeout exceeded") {
		t.Fatalf("missing lock timout error in output: %s", output)
	}
	if strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("unexpected plan summery in output: %s", output)
	}
	if strings.Contains(output, "1 added, 0 changed, 0 destroyed") {
		t.Fatalf("unexpected apply summery in output: %s", output)
	}
}

func TestRemote_applyDestroy(t *testing.T) {
	b := testBackendDefault(t)

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply-destroy")
	defer modCleanup()

	input := testInput(t, map[string]string{
		"approve": "yes",
	})

	op := testOperationApply()
	op.Destroy = true
	op.Module = mod
	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err != nil {
		t.Fatalf("error running operation: %v", run.Err)
	}

	if len(input.answers) > 0 {
		t.Fatalf("expected no unused answers, got: %v", input.answers)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "0 to add, 0 to change, 1 to destroy") {
		t.Fatalf("missing plan summery in output: %s", output)
	}
	if !strings.Contains(output, "0 added, 0 changed, 1 destroyed") {
		t.Fatalf("missing apply summery in output: %s", output)
	}
}

func TestRemote_applyDestroyNoConfig(t *testing.T) {
	b := testBackendDefault(t)

	input := testInput(t, map[string]string{
		"approve": "yes",
	})

	op := testOperationApply()
	op.Destroy = true
	op.Module = nil
	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err != nil {
		t.Fatalf("unexpected apply error: %v", run.Err)
	}

	if len(input.answers) > 0 {
		t.Fatalf("expected no unused answers, got: %v", input.answers)
	}
}

func TestRemote_applyPolicyPass(t *testing.T) {
	b := testBackendDefault(t)

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply-policy-passed")
	defer modCleanup()

	input := testInput(t, map[string]string{
		"approve": "yes",
	})

	op := testOperationApply()
	op.Module = mod
	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err != nil {
		t.Fatalf("error running operation: %v", run.Err)
	}

	if len(input.answers) > 0 {
		t.Fatalf("expected no unused answers, got: %v", input.answers)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("missing plan summery in output: %s", output)
	}
	if !strings.Contains(output, "Sentinel Result: true") {
		t.Fatalf("missing Sentinel result in output: %s", output)
	}
	if !strings.Contains(output, "1 added, 0 changed, 0 destroyed") {
		t.Fatalf("missing apply summery in output: %s", output)
	}
}

func TestRemote_applyPolicyHardFail(t *testing.T) {
	b := testBackendDefault(t)

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply-policy-hard-failed")
	defer modCleanup()

	input := testInput(t, map[string]string{
		"approve": "yes",
	})

	op := testOperationApply()
	op.Module = mod
	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err == nil {
		t.Fatalf("expected an apply error, got: %v", run.Err)
	}
	if !strings.Contains(run.Err.Error(), "hard failed") {
		t.Fatalf("expected a policy check error, got: %v", run.Err)
	}
	if len(input.answers) != 1 {
		t.Fatalf("expected an unused answers, got: %v", input.answers)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("missing plan summery in output: %s", output)
	}
	if !strings.Contains(output, "Sentinel Result: false") {
		t.Fatalf("missing Sentinel result in output: %s", output)
	}
	if strings.Contains(output, "1 added, 0 changed, 0 destroyed") {
		t.Fatalf("unexpected apply summery in output: %s", output)
	}
}

func TestRemote_applyPolicySoftFail(t *testing.T) {
	b := testBackendDefault(t)

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply-policy-soft-failed")
	defer modCleanup()

	input := testInput(t, map[string]string{
		"override": "override",
		"approve":  "yes",
	})

	op := testOperationApply()
	op.Module = mod
	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err != nil {
		t.Fatalf("error running operation: %v", run.Err)
	}

	if len(input.answers) > 0 {
		t.Fatalf("expected no unused answers, got: %v", input.answers)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("missing plan summery in output: %s", output)
	}
	if !strings.Contains(output, "Sentinel Result: false") {
		t.Fatalf("missing Sentinel result in output: %s", output)
	}
	if !strings.Contains(output, "1 added, 0 changed, 0 destroyed") {
		t.Fatalf("missing apply summery in output: %s", output)
	}
}

func TestRemote_applyPolicySoftFailAutoApprove(t *testing.T) {
	b := testBackendDefault(t)

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply-policy-soft-failed")
	defer modCleanup()

	input := testInput(t, map[string]string{
		"override": "override",
	})

	op := testOperationApply()
	op.AutoApprove = true
	op.Module = mod
	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err != nil {
		t.Fatalf("error running operation: %v", run.Err)
	}

	if len(input.answers) > 0 {
		t.Fatalf("expected no unused answers, got: %v", input.answers)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("missing plan summery in output: %s", output)
	}
	if !strings.Contains(output, "Sentinel Result: false") {
		t.Fatalf("missing Sentinel result in output: %s", output)
	}
	if !strings.Contains(output, "1 added, 0 changed, 0 destroyed") {
		t.Fatalf("missing apply summery in output: %s", output)
	}
}
