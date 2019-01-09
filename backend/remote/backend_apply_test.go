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
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/internal/initwd"
	"github.com/hashicorp/terraform/plans/planfile"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

func testOperationApply(t *testing.T, configDir string) (*backend.Operation, func()) {
	t.Helper()

	_, configLoader, configCleanup := initwd.MustLoadConfigForTests(t, configDir)

	return &backend.Operation{
		ConfigDir:    configDir,
		ConfigLoader: configLoader,
		Parallelism:  defaultParallelism,
		PlanRefresh:  true,
		Type:         backend.OperationTypeApply,
	}, configCleanup
}

func TestRemote_applyBasic(t *testing.T) {
	b := testBackendDefault(t)

	op, configCleanup := testOperationApply(t, "./test-fixtures/apply")
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
}

func TestRemote_applyWithoutPermissions(t *testing.T) {
	b := testBackendNoDefault(t)

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
	w.Permissions.CanUpdate = false

	op, configCleanup := testOperationApply(t, "./test-fixtures/apply")
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

	errOutput := b.CLI.(*cli.MockUi).ErrorWriter.String()
	if !strings.Contains(errOutput, "Insufficient rights to apply changes") {
		t.Fatalf("expected a permissions error, got: %v", errOutput)
	}
}

func TestRemote_applyWithVCS(t *testing.T) {
	b := testBackendNoDefault(t)

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

	op, configCleanup := testOperationApply(t, "./test-fixtures/apply")
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

	errOutput := b.CLI.(*cli.MockUi).ErrorWriter.String()
	if !strings.Contains(errOutput, "not allowed for workspaces with a VCS") {
		t.Fatalf("expected a VCS error, got: %v", errOutput)
	}
}

func TestRemote_applyWithParallelism(t *testing.T) {
	b := testBackendDefault(t)

	op, configCleanup := testOperationApply(t, "./test-fixtures/apply")
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

	errOutput := b.CLI.(*cli.MockUi).ErrorWriter.String()
	if !strings.Contains(errOutput, "parallelism values are currently not supported") {
		t.Fatalf("expected a parallelism error, got: %v", errOutput)
	}
}

func TestRemote_applyWithPlan(t *testing.T) {
	b := testBackendDefault(t)

	op, configCleanup := testOperationApply(t, "./test-fixtures/apply")
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

	errOutput := b.CLI.(*cli.MockUi).ErrorWriter.String()
	if !strings.Contains(errOutput, "saved plan is currently not supported") {
		t.Fatalf("expected a saved plan error, got: %v", errOutput)
	}
}

func TestRemote_applyWithoutRefresh(t *testing.T) {
	b := testBackendDefault(t)

	op, configCleanup := testOperationApply(t, "./test-fixtures/apply")
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

	errOutput := b.CLI.(*cli.MockUi).ErrorWriter.String()
	if !strings.Contains(errOutput, "refresh is currently not supported") {
		t.Fatalf("expected a refresh error, got: %v", errOutput)
	}
}

func TestRemote_applyWithTarget(t *testing.T) {
	b := testBackendDefault(t)

	op, configCleanup := testOperationApply(t, "./test-fixtures/apply")
	defer configCleanup()

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

	errOutput := b.CLI.(*cli.MockUi).ErrorWriter.String()
	if !strings.Contains(errOutput, "targeting is currently not supported") {
		t.Fatalf("expected a targeting error, got: %v", errOutput)
	}
}

func TestRemote_applyWithVariables(t *testing.T) {
	b := testBackendDefault(t)

	op, configCleanup := testOperationApply(t, "./test-fixtures/apply-variables")
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

	errOutput := b.CLI.(*cli.MockUi).ErrorWriter.String()
	if !strings.Contains(errOutput, "variables are currently not supported") {
		t.Fatalf("expected a variables error, got: %v", errOutput)
	}
}

func TestRemote_applyNoConfig(t *testing.T) {
	b := testBackendDefault(t)

	op, configCleanup := testOperationApply(t, "./test-fixtures/empty")
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

	errOutput := b.CLI.(*cli.MockUi).ErrorWriter.String()
	if !strings.Contains(errOutput, "configuration files found") {
		t.Fatalf("expected configuration files error, got: %v", errOutput)
	}
}

func TestRemote_applyNoChanges(t *testing.T) {
	b := testBackendDefault(t)

	op, configCleanup := testOperationApply(t, "./test-fixtures/apply-no-changes")
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
	b := testBackendDefault(t)

	op, configCleanup := testOperationApply(t, "./test-fixtures/apply")
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

	errOutput := b.CLI.(*cli.MockUi).ErrorWriter.String()
	if !strings.Contains(errOutput, "Apply discarded") {
		t.Fatalf("expected an apply discarded error, got: %v", errOutput)
	}
}

func TestRemote_applyAutoApprove(t *testing.T) {
	b := testBackendDefault(t)

	op, configCleanup := testOperationApply(t, "./test-fixtures/apply")
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

func TestRemote_applyWithAutoApply(t *testing.T) {
	b := testBackendNoDefault(t)

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

	op, configCleanup := testOperationApply(t, "./test-fixtures/apply")
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

	b := testBackendDefault(t)

	op, configCleanup := testOperationApply(t, "./test-fixtures/apply")
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
	if strings.Contains(output, "Running apply in the remote backend") {
		t.Fatalf("unexpected remote backend header in output: %s", output)
	}
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("expected plan summery in output: %s", output)
	}
	if !strings.Contains(output, "1 added, 0 changed, 0 destroyed") {
		t.Fatalf("expected apply summery in output: %s", output)
	}
}

func TestRemote_applyWorkspaceWithoutOperations(t *testing.T) {
	b := testBackendNoDefault(t)
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

	op, configCleanup := testOperationApply(t, "./test-fixtures/apply")
	defer configCleanup()

	input := testInput(t, map[string]string{
		"approve": "yes",
	})

	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = "no-operations"

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
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("expected plan summery in output: %s", output)
	}
	if !strings.Contains(output, "1 added, 0 changed, 0 destroyed") {
		t.Fatalf("expected apply summery in output: %s", output)
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

	op, configCleanup := testOperationApply(t, "./test-fixtures/apply")
	defer configCleanup()

	input := testInput(t, map[string]string{
		"cancel":  "yes",
		"approve": "yes",
	})

	op.StateLockTimeout = 5 * time.Second
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
	b := testBackendDefault(t)

	op, configCleanup := testOperationApply(t, "./test-fixtures/apply-destroy")
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
	b := testBackendDefault(t)

	input := testInput(t, map[string]string{
		"approve": "yes",
	})

	op, configCleanup := testOperationApply(t, "./test-fixtures/empty")
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
	b := testBackendDefault(t)

	op, configCleanup := testOperationApply(t, "./test-fixtures/apply-policy-passed")
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
	b := testBackendDefault(t)

	op, configCleanup := testOperationApply(t, "./test-fixtures/apply-policy-hard-failed")
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

	errOutput := b.CLI.(*cli.MockUi).ErrorWriter.String()
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
	b := testBackendDefault(t)

	op, configCleanup := testOperationApply(t, "./test-fixtures/apply-policy-soft-failed")
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
	b := testBackendDefault(t)

	op, configCleanup := testOperationApply(t, "./test-fixtures/apply-policy-soft-failed")
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

	errOutput := b.CLI.(*cli.MockUi).ErrorWriter.String()
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
	b := testBackendDefault(t)

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

	op, configCleanup := testOperationApply(t, "./test-fixtures/apply-policy-soft-failed")
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
	b := testBackendDefault(t)

	op, configCleanup := testOperationApply(t, "./test-fixtures/apply-with-error")
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
