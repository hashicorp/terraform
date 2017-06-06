package local

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
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

func TestLocal_applyBackendFail(t *testing.T) {
	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply")
	defer modCleanup()

	b := TestLocal(t)
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory")
	}
	err = os.Chdir(filepath.Dir(b.StatePath))
	if err != nil {
		t.Fatalf("failed to set temporary working directory")
	}
	defer os.Chdir(wd)

	b.Backend = &backendWithFailingState{}
	b.CLI = new(cli.MockUi)
	p := TestLocalProvider(t, b, "test")

	p.ApplyReturn = &terraform.InstanceState{ID: "yes"}

	op := testOperationApply()
	op.Module = mod

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	<-run.Done()
	if run.Err == nil {
		t.Fatalf("apply succeeded; want error")
	}

	errStr := run.Err.Error()
	if !strings.Contains(errStr, "terraform state push errored.tfstate") {
		t.Fatalf("wrong error message:\n%s", errStr)
	}

	msgStr := b.CLI.(*cli.MockUi).ErrorWriter.String()
	if !strings.Contains(msgStr, "Failed to save state: fake failure") {
		t.Fatalf("missing original error message in output:\n%s", msgStr)
	}

	// The fallback behavior should've created a file errored.tfstate in the
	// current working directory.
	checkState(t, "errored.tfstate", `
test_instance.foo:
  ID = yes
	`)
}

type backendWithFailingState struct {
	Local
}

func (b *backendWithFailingState) State(name string) (state.State, error) {
	return &failingState{
		&state.LocalState{
			Path: "failing-state.tfstate",
		},
	}, nil
}

type failingState struct {
	*state.LocalState
}

func (s failingState) WriteState(state *terraform.State) error {
	return errors.New("fake failure")
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
