package local

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/terraform"
)

func TestLocal_applyBasic(t *testing.T) {
	b := TestLocal(t)
	p := TestLocalProvider(t, b, "test")

	p.ApplyReturn = &terraform.InstanceState{ID: "yes"}

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply")
	defer modCleanup()

	op := testOperationApply()
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

	if !p.DiffCalled {
		t.Fatal("diff should be called")
	}

	if !p.ApplyCalled {
		t.Fatal("apply should be called")
	}

	checkState(t, b.StateOutPath, `
test_instance.foo:
  ID = yes
	`)
}

func TestLocal_applyEmptyDir(t *testing.T) {
	b := TestLocal(t)
	p := TestLocalProvider(t, b, "test")

	p.ApplyReturn = &terraform.InstanceState{ID: "yes"}

	op := testOperationApply()
	op.Module = nil

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	<-run.Done()
	if run.Err == nil {
		t.Fatal("should error")
	}

	if p.ApplyCalled {
		t.Fatal("apply should not be called")
	}

	if _, err := os.Stat(b.StateOutPath); err == nil {
		t.Fatal("should not exist")
	}
}

func TestLocal_applyEmptyDirDestroy(t *testing.T) {
	b := TestLocal(t)
	p := TestLocalProvider(t, b, "test")

	p.ApplyReturn = nil

	op := testOperationApply()
	op.Module = nil
	op.Destroy = true

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	<-run.Done()
	if run.Err != nil {
		t.Fatalf("err: %s", err)
	}

	if p.ApplyCalled {
		t.Fatal("apply should not be called")
	}

	checkState(t, b.StateOutPath, `<no state>`)
}

func TestLocal_applyError(t *testing.T) {
	b := TestLocal(t)
	p := TestLocalProvider(t, b, "test")

	var lock sync.Mutex
	errored := false
	p.ApplyFn = func(
		info *terraform.InstanceInfo,
		s *terraform.InstanceState,
		d *terraform.InstanceDiff) (*terraform.InstanceState, error) {
		lock.Lock()
		defer lock.Unlock()

		if !errored && info.Id == "test_instance.bar" {
			errored = true
			return nil, fmt.Errorf("error")
		}

		return &terraform.InstanceState{ID: "foo"}, nil
	}
	p.DiffFn = func(
		*terraform.InstanceInfo,
		*terraform.InstanceState,
		*terraform.ResourceConfig) (*terraform.InstanceDiff, error) {
		return &terraform.InstanceDiff{
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"ami": &terraform.ResourceAttrDiff{
					New: "bar",
				},
			},
		}, nil
	}

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply-error")
	defer modCleanup()

	op := testOperationApply()
	op.Module = mod

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	<-run.Done()
	if run.Err == nil {
		t.Fatal("should error")
	}

	checkState(t, b.StateOutPath, `
test_instance.foo:
  ID = foo
	`)
}

func testOperationApply() *backend.Operation {
	return &backend.Operation{
		Type: backend.OperationTypeApply,
	}
}

// testApplyState is just a common state that we use for testing refresh.
func testApplyState() *terraform.State {
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
