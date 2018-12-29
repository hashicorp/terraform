package local

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/configs/configload"
	"github.com/hashicorp/terraform/terraform"
	"github.com/zclconf/go-cty/cty"
)

func TestLocal_refresh(t *testing.T) {
	b, cleanup := TestLocal(t)
	defer cleanup()

	p := TestLocalProvider(t, b, "test", refreshFixtureSchema())
	terraform.TestStateFile(t, b.StatePath, testRefreshState())

	p.RefreshFn = nil
	p.RefreshReturn = &terraform.InstanceState{ID: "yes"}

	op, configCleanup := testOperationRefresh(t, "./test-fixtures/refresh")
	defer configCleanup()

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
  provider = provider.test
	`)
}

func TestLocal_refreshNoConfig(t *testing.T) {
	b, cleanup := TestLocal(t)
	defer cleanup()
	p := TestLocalProvider(t, b, "test", &terraform.ProviderSchema{})
	terraform.TestStateFile(t, b.StatePath, testRefreshState())

	p.RefreshFn = nil
	p.RefreshReturn = &terraform.InstanceState{ID: "yes"}

	op, configCleanup := testOperationRefresh(t, "./test-fixtures/empty")
	defer configCleanup()

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
  provider = provider.test
	`)
}

// GH-12174
func TestLocal_refreshNilModuleWithInput(t *testing.T) {
	b, cleanup := TestLocal(t)
	defer cleanup()
	p := TestLocalProvider(t, b, "test", &terraform.ProviderSchema{})
	terraform.TestStateFile(t, b.StatePath, testRefreshState())

	p.RefreshFn = nil
	p.RefreshReturn = &terraform.InstanceState{ID: "yes"}

	b.OpInput = true

	op, configCleanup := testOperationRefresh(t, "./test-fixtures/empty")
	defer configCleanup()

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
  provider = provider.test
	`)
}

func TestLocal_refreshInput(t *testing.T) {
	b, cleanup := TestLocal(t)
	defer cleanup()
	p := TestLocalProvider(t, b, "test", nil)
	terraform.TestStateFile(t, b.StatePath, testRefreshState())

	p.GetSchemaReturn = &terraform.ProviderSchema{
		Provider: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"value": {Type: cty.String, Optional: true},
			},
		},
		ResourceTypes: map[string]*configschema.Block{
			"test_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
	}
	p.ConfigureFn = func(c *terraform.ResourceConfig) error {
		if v, ok := c.Get("value"); !ok || v != "bar" {
			return fmt.Errorf("no value set")
		}

		return nil
	}

	p.RefreshFn = nil
	p.RefreshReturn = &terraform.InstanceState{ID: "yes"}

	// Enable input asking since it is normally disabled by default
	b.OpInput = true
	b.ContextOpts.UIInput = &terraform.MockUIInput{InputReturnString: "bar"}

	op, configCleanup := testOperationRefresh(t, "./test-fixtures/refresh-var-unset")
	defer configCleanup()
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
  provider = provider.test
	`)
}

func TestLocal_refreshValidate(t *testing.T) {
	b, cleanup := TestLocal(t)
	defer cleanup()
	p := TestLocalProvider(t, b, "test", refreshFixtureSchema())
	terraform.TestStateFile(t, b.StatePath, testRefreshState())

	p.RefreshFn = nil
	p.RefreshReturn = &terraform.InstanceState{ID: "yes"}

	// Enable validation
	b.OpValidation = true

	op, configCleanup := testOperationRefresh(t, "./test-fixtures/refresh")
	defer configCleanup()

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
  provider = provider.test
	`)
}

func testOperationRefresh(t *testing.T, configDir string) (*backend.Operation, func()) {
	t.Helper()

	_, configLoader, configCleanup := configload.MustLoadConfigForTests(t, configDir)

	return &backend.Operation{
		Type:         backend.OperationTypeRefresh,
		ConfigDir:    configDir,
		ConfigLoader: configLoader,
	}, configCleanup
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

// refreshFixtureSchema returns a schema suitable for processing the
// configuration in test-fixtures/refresh . This schema should be
// assigned to a mock provider named "test".
func refreshFixtureSchema() *terraform.ProviderSchema {
	return &terraform.ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_instance": {
				Attributes: map[string]*configschema.Attribute{
					"ami": {Type: cty.String, Optional: true},
				},
			},
		},
	}
}
