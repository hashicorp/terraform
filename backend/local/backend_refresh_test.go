package local

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/terraform"
)

func TestLocal_refresh(t *testing.T) {
	b := TestLocal(t)
	p := TestLocalProvider(t, b, "test")
	terraform.TestStateFile(t, b.StatePath, testRefreshState())

	p.RefreshFn = nil
	p.RefreshReturn = &terraform.InstanceState{ID: "yes"}

	mod, modCleanup := module.TestTree(t, "./test-fixtures/refresh")
	defer modCleanup()

	op := testOperationRefresh()
	op.Module = mod

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	<-run.Done()

	if !p.RefreshCalled {
		t.Fatal("refresh should be called")
	}

	checkState(t, b.StateOutPath, `
test_instance.foo:
  ID = yes
	`)
}

func TestLocal_refreshNilModule(t *testing.T) {
	b := TestLocal(t)
	p := TestLocalProvider(t, b, "test")
	terraform.TestStateFile(t, b.StatePath, testRefreshState())

	p.RefreshFn = nil
	p.RefreshReturn = &terraform.InstanceState{ID: "yes"}

	op := testOperationRefresh()
	op.Module = nil

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	<-run.Done()

	if !p.RefreshCalled {
		t.Fatal("refresh should be called")
	}

	checkState(t, b.StateOutPath, `
test_instance.foo:
  ID = yes
	`)
}

// GH-12174
func TestLocal_refreshNilModuleWithInput(t *testing.T) {
	b := TestLocal(t)
	p := TestLocalProvider(t, b, "test")
	terraform.TestStateFile(t, b.StatePath, testRefreshState())

	p.RefreshFn = nil
	p.RefreshReturn = &terraform.InstanceState{ID: "yes"}

	b.OpInput = true

	op := testOperationRefresh()
	op.Module = nil

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	<-run.Done()

	if !p.RefreshCalled {
		t.Fatal("refresh should be called")
	}

	checkState(t, b.StateOutPath, `
test_instance.foo:
  ID = yes
	`)
}

func TestLocal_refreshInput(t *testing.T) {
	b := TestLocal(t)
	p := TestLocalProvider(t, b, "test")
	terraform.TestStateFile(t, b.StatePath, testRefreshState())

	p.ConfigureFn = func(c *terraform.ResourceConfig) error {
		if v, ok := c.Get("value"); !ok || v != "bar" {
			return fmt.Errorf("no value set")
		}

		return nil
	}

	p.RefreshFn = nil
	p.RefreshReturn = &terraform.InstanceState{ID: "yes"}

	mod, modCleanup := module.TestTree(t, "./test-fixtures/refresh-var-unset")
	defer modCleanup()

	// Enable input asking since it is normally disabled by default
	b.OpInput = true
	b.ContextOpts.UIInput = &terraform.MockUIInput{InputReturnString: "bar"}

	op := testOperationRefresh()
	op.Module = mod
	op.UIIn = b.ContextOpts.UIInput

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	<-run.Done()

	if !p.RefreshCalled {
		t.Fatal("refresh should be called")
	}

	checkState(t, b.StateOutPath, `
test_instance.foo:
  ID = yes
	`)
}

func TestLocal_refreshValidate(t *testing.T) {
	b := TestLocal(t)
	p := TestLocalProvider(t, b, "test")
	terraform.TestStateFile(t, b.StatePath, testRefreshState())

	p.RefreshFn = nil
	p.RefreshReturn = &terraform.InstanceState{ID: "yes"}

	mod, modCleanup := module.TestTree(t, "./test-fixtures/refresh")
	defer modCleanup()

	// Enable validation
	b.OpValidation = true

	op := testOperationRefresh()
	op.Module = mod

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	<-run.Done()

	if !p.ValidateCalled {
		t.Fatal("validate should be called")
	}

	checkState(t, b.StateOutPath, `
test_instance.foo:
  ID = yes
	`)
}

func testOperationRefresh() *backend.Operation {
	return &backend.Operation{
		Type: backend.OperationTypeRefresh,
	}
}

// testRefreshState is just a common state that we use for testing refresh.
func testRefreshState() *terraform.State {
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
				Outputs: map[string]*terraform.OutputState{},
			},
		},
	}
}
