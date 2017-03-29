package local

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/terraform"
)

func TestLocal_planBasic(t *testing.T) {
	b := TestLocal(t)
	p := TestLocalProvider(t, b, "test")

	mod, modCleanup := module.TestTree(t, "./test-fixtures/plan")
	defer modCleanup()

	op := testOperationPlan()
	op.Module = mod
	op.PlanRefresh = true

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	<-run.Done()
	if run.Err != nil {
		t.Fatalf("err: %s", err)
	}

	if !p.DiffCalled {
		t.Fatal("diff should be called")
	}
}

func TestLocal_planNoConfig(t *testing.T) {
	b := TestLocal(t)
	TestLocalProvider(t, b, "test")

	op := testOperationPlan()
	op.Module = nil
	op.PlanRefresh = true

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	<-run.Done()

	err = run.Err
	if err == nil {
		t.Fatal("should error")
	}
	if !strings.Contains(err.Error(), "configuration") {
		t.Fatalf("bad: %s", err)
	}
}

func TestLocal_planRefreshFalse(t *testing.T) {
	b := TestLocal(t)
	p := TestLocalProvider(t, b, "test")
	terraform.TestStateFile(t, b.StatePath, testPlanState())

	mod, modCleanup := module.TestTree(t, "./test-fixtures/plan")
	defer modCleanup()

	op := testOperationPlan()
	op.Module = mod

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	<-run.Done()
	if run.Err != nil {
		t.Fatalf("err: %s", err)
	}

	if p.RefreshCalled {
		t.Fatal("refresh should not be called")
	}

	if !run.PlanEmpty {
		t.Fatal("plan should be empty")
	}
}

func TestLocal_planDestroy(t *testing.T) {
	b := TestLocal(t)
	p := TestLocalProvider(t, b, "test")
	terraform.TestStateFile(t, b.StatePath, testPlanState())

	mod, modCleanup := module.TestTree(t, "./test-fixtures/plan")
	defer modCleanup()

	outDir := testTempDir(t)
	defer os.RemoveAll(outDir)
	planPath := filepath.Join(outDir, "plan.tfplan")

	op := testOperationPlan()
	op.Destroy = true
	op.PlanRefresh = true
	op.Module = mod
	op.PlanOutPath = planPath

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	<-run.Done()
	if run.Err != nil {
		t.Fatalf("err: %s", err)
	}

	if !p.RefreshCalled {
		t.Fatal("refresh should be called")
	}

	if run.PlanEmpty {
		t.Fatal("plan should not be empty")
	}

	plan := testReadPlan(t, planPath)
	for _, m := range plan.Diff.Modules {
		for _, r := range m.Resources {
			if !r.Destroy {
				t.Fatalf("bad: %#v", r)
			}
		}
	}
}

func TestLocal_planDestroyNoConfig(t *testing.T) {
	b := TestLocal(t)
	p := TestLocalProvider(t, b, "test")
	terraform.TestStateFile(t, b.StatePath, testPlanState())

	outDir := testTempDir(t)
	defer os.RemoveAll(outDir)
	planPath := filepath.Join(outDir, "plan.tfplan")

	op := testOperationPlan()
	op.Destroy = true
	op.PlanRefresh = true
	op.Module = nil
	op.PlanOutPath = planPath

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	<-run.Done()
	if run.Err != nil {
		t.Fatalf("err: %s", err)
	}

	if !p.RefreshCalled {
		t.Fatal("refresh should be called")
	}

	if run.PlanEmpty {
		t.Fatal("plan should not be empty")
	}

	plan := testReadPlan(t, planPath)
	for _, m := range plan.Diff.Modules {
		for _, r := range m.Resources {
			if !r.Destroy {
				t.Fatalf("bad: %#v", r)
			}
		}
	}
}

func TestLocal_planOutPathNoChange(t *testing.T) {
	b := TestLocal(t)
	TestLocalProvider(t, b, "test")
	terraform.TestStateFile(t, b.StatePath, testPlanState())

	mod, modCleanup := module.TestTree(t, "./test-fixtures/plan")
	defer modCleanup()

	outDir := testTempDir(t)
	defer os.RemoveAll(outDir)
	planPath := filepath.Join(outDir, "plan.tfplan")

	op := testOperationPlan()
	op.Module = mod
	op.PlanOutPath = planPath

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	<-run.Done()
	if run.Err != nil {
		t.Fatalf("err: %s", err)
	}

	plan := testReadPlan(t, planPath)
	if !plan.Diff.Empty() {
		t.Fatalf("expected empty plan to be written")
	}
}

func testOperationPlan() *backend.Operation {
	return &backend.Operation{
		Type: backend.OperationTypePlan,
	}
}

// testPlanState is just a common state that we use for testing refresh.
func testPlanState() *terraform.State {
	return &terraform.State{
		Version: 2,
		Modules: []*terraform.ModuleState{
			&terraform.ModuleState{
				Path: []string{"root"},
				Resources: map[string]*terraform.ResourceState{
					"test_instance.foo": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "bar",
						},
					},
				},
			},
		},
	}
}

func testReadPlan(t *testing.T, path string) *terraform.Plan {
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer f.Close()

	p, err := terraform.ReadPlan(f)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return p
}
