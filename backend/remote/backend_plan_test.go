package remote

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

func testOperationPlan() *backend.Operation {
	return &backend.Operation{
		Type: backend.OperationTypePlan,
	}
}

func TestRemote_planBasic(t *testing.T) {
	b := testBackendDefault(t)

	mod, modCleanup := module.TestTree(t, "./test-fixtures/plan")
	defer modCleanup()

	op := testOperationPlan()
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
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("missing plan summery in output: %s", output)
	}
}

func TestRemote_planWithPlan(t *testing.T) {
	b := testBackendDefault(t)

	mod, modCleanup := module.TestTree(t, "./test-fixtures/plan")
	defer modCleanup()

	op := testOperationPlan()
	op.Module = mod
	op.Plan = &terraform.Plan{}
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}
	<-run.Done()

	if run.Err == nil {
		t.Fatalf("expected a plan error, got: %v", run.Err)
	}
	if !strings.Contains(run.Err.Error(), "saved plan is currently not supported") {
		t.Fatalf("expected a saved plan error, got: %v", run.Err)
	}
}

func TestRemote_planWithPath(t *testing.T) {
	b := testBackendDefault(t)

	mod, modCleanup := module.TestTree(t, "./test-fixtures/plan")
	defer modCleanup()

	op := testOperationPlan()
	op.Module = mod
	op.PlanOutPath = "./test-fixtures/plan"
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}
	<-run.Done()

	if run.Err == nil {
		t.Fatalf("expected a plan error, got: %v", run.Err)
	}
	if !strings.Contains(run.Err.Error(), "generated plan is currently not supported") {
		t.Fatalf("expected a generated plan error, got: %v", run.Err)
	}
}

func TestRemote_planWithTarget(t *testing.T) {
	b := testBackendDefault(t)

	mod, modCleanup := module.TestTree(t, "./test-fixtures/plan")
	defer modCleanup()

	op := testOperationPlan()
	op.Module = mod
	op.Targets = []string{"null_resource.foo"}
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}
	<-run.Done()

	if run.Err == nil {
		t.Fatalf("expected a plan error, got: %v", run.Err)
	}
	if !strings.Contains(run.Err.Error(), "targeting is currently not supported") {
		t.Fatalf("expected a targeting error, got: %v", run.Err)
	}
}

func TestRemote_planNoConfig(t *testing.T) {
	b := testBackendDefault(t)

	op := testOperationPlan()
	op.Module = nil
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}
	<-run.Done()

	if run.Err == nil {
		t.Fatalf("expected a plan error, got: %v", run.Err)
	}
	if !strings.Contains(run.Err.Error(), "configuration files found") {
		t.Fatalf("expected configuration files error, got: %v", run.Err)
	}
}

func TestRemote_planDestroy(t *testing.T) {
	b := testBackendDefault(t)

	mod, modCleanup := module.TestTree(t, "./test-fixtures/plan")
	defer modCleanup()

	op := testOperationPlan()
	op.Destroy = true
	op.Module = mod
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err != nil {
		t.Fatalf("unexpected plan error: %v", run.Err)
	}
}

func TestRemote_planDestroyNoConfig(t *testing.T) {
	b := testBackendDefault(t)

	op := testOperationPlan()
	op.Destroy = true
	op.Module = nil
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err != nil {
		t.Fatalf("unexpected plan error: %v", run.Err)
	}
}
