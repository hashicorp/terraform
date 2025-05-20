// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package local

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/clistate"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/initwd"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/terraform"

	"github.com/zclconf/go-cty/cty"
)

func TestLocal_refresh(t *testing.T) {
	b := TestLocal(t)

	p := TestLocalProvider(t, b, "test", refreshFixtureSchema())
	testStateFile(t, b.StatePath, testRefreshState())

	p.ReadResourceFn = nil
	p.ReadResourceResponse = &providers.ReadResourceResponse{NewState: cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("yes"),
	})}

	op, configCleanup, done := testOperationRefresh(t, "./testdata/refresh")
	defer configCleanup()
	defer done(t)

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	<-run.Done()

	if !p.ReadResourceCalled {
		t.Fatal("ReadResource should be called")
	}

	checkState(t, b.StateOutPath, `
test_instance.foo:
  ID = yes
  provider = provider["registry.terraform.io/hashicorp/test"]
	`)

	// the backend should be unlocked after a run
	assertBackendStateUnlocked(t, b)
}

func TestLocal_refreshInput(t *testing.T) {
	b := TestLocal(t)

	schema := providers.ProviderSchema{
		Provider: providers.Schema{
			Body: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"value": {Type: cty.String, Optional: true},
				},
			},
		},
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id":  {Type: cty.String, Computed: true},
						"foo": {Type: cty.String, Optional: true},
						"ami": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}

	p := TestLocalProvider(t, b, "test", schema)
	testStateFile(t, b.StatePath, testRefreshState())

	p.ReadResourceFn = nil
	p.ReadResourceResponse = &providers.ReadResourceResponse{NewState: cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("yes"),
	})}
	p.ConfigureProviderFn = func(req providers.ConfigureProviderRequest) (resp providers.ConfigureProviderResponse) {
		val := req.Config.GetAttr("value")
		if val.IsNull() || val.AsString() != "bar" {
			resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("incorrect value %#v", val))
		}

		return
	}

	// Enable input asking since it is normally disabled by default
	b.OpInput = true
	b.ContextOpts.UIInput = &terraform.MockUIInput{InputReturnString: "bar"}

	op, configCleanup, done := testOperationRefresh(t, "./testdata/refresh-var-unset")
	defer configCleanup()
	defer done(t)
	op.UIIn = b.ContextOpts.UIInput

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	<-run.Done()

	if !p.ReadResourceCalled {
		t.Fatal("ReadResource should be called")
	}

	checkState(t, b.StateOutPath, `
test_instance.foo:
  ID = yes
  provider = provider["registry.terraform.io/hashicorp/test"]
	`)
}

func TestLocal_refreshValidate(t *testing.T) {
	b := TestLocal(t)
	p := TestLocalProvider(t, b, "test", refreshFixtureSchema())
	testStateFile(t, b.StatePath, testRefreshState())
	p.ReadResourceFn = nil
	p.ReadResourceResponse = &providers.ReadResourceResponse{NewState: cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("yes"),
	})}

	// Enable validation
	b.OpValidation = true

	op, configCleanup, done := testOperationRefresh(t, "./testdata/refresh")
	defer configCleanup()
	defer done(t)

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	<-run.Done()

	checkState(t, b.StateOutPath, `
test_instance.foo:
  ID = yes
  provider = provider["registry.terraform.io/hashicorp/test"]
	`)
}

func TestLocal_refreshValidateProviderConfigured(t *testing.T) {
	b := TestLocal(t)

	schema := providers.ProviderSchema{
		Provider: providers.Schema{
			Body: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"value": {Type: cty.String, Optional: true},
				},
			},
		},
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id":  {Type: cty.String, Computed: true},
						"ami": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}

	p := TestLocalProvider(t, b, "test", schema)
	testStateFile(t, b.StatePath, testRefreshState())
	p.ReadResourceFn = nil
	p.ReadResourceResponse = &providers.ReadResourceResponse{NewState: cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("yes"),
	})}

	// Enable validation
	b.OpValidation = true

	op, configCleanup, done := testOperationRefresh(t, "./testdata/refresh-provider-config")
	defer configCleanup()
	defer done(t)

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	<-run.Done()

	if !p.ValidateProviderConfigCalled {
		t.Fatal("Validate provider config should be called")
	}

	checkState(t, b.StateOutPath, `
test_instance.foo:
  ID = yes
  provider = provider["registry.terraform.io/hashicorp/test"]
	`)
}

// This test validates the state lacking behavior when the inner call to
// Context() fails
func TestLocal_refresh_context_error(t *testing.T) {
	b := TestLocal(t)
	testStateFile(t, b.StatePath, testRefreshState())
	op, configCleanup, done := testOperationRefresh(t, "./testdata/apply")
	defer configCleanup()
	defer done(t)

	// we coerce a failure in Context() by omitting the provider schema

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	<-run.Done()
	if run.Result == backendrun.OperationSuccess {
		t.Fatal("operation succeeded; want failure")
	}
	assertBackendStateUnlocked(t, b)
}

func TestLocal_refreshEmptyState(t *testing.T) {
	b := TestLocal(t)

	p := TestLocalProvider(t, b, "test", refreshFixtureSchema())
	testStateFile(t, b.StatePath, states.NewState())

	p.ReadResourceFn = nil
	p.ReadResourceResponse = &providers.ReadResourceResponse{NewState: cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("yes"),
	})}

	op, configCleanup, done := testOperationRefresh(t, "./testdata/refresh")
	defer configCleanup()

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	<-run.Done()

	output := done(t)

	if stderr := output.Stderr(); stderr != "" {
		t.Fatalf("expected only warning diags, got errors: %s", stderr)
	}
	if got, want := output.Stdout(), "Warning: Empty or non-existent state"; !strings.Contains(got, want) {
		t.Errorf("wrong diags\n got: %s\nwant: %s", got, want)
	}

	// the backend should be unlocked after a run
	assertBackendStateUnlocked(t, b)
}

func testOperationRefresh(t *testing.T, configDir string) (*backendrun.Operation, func(), func(*testing.T) *terminal.TestOutput) {
	t.Helper()

	_, configLoader, configCleanup := initwd.MustLoadConfigForTests(t, configDir, "tests")

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewOperation(arguments.ViewHuman, false, views.NewView(streams))

	// Many of our tests use an overridden "test" provider that's just in-memory
	// inside the test process, not a separate plugin on disk.
	depLocks := depsfile.NewLocks()
	depLocks.SetProviderOverridden(addrs.MustParseProviderSourceString("registry.terraform.io/hashicorp/test"))

	return &backendrun.Operation{
		Type:            backendrun.OperationTypeRefresh,
		ConfigDir:       configDir,
		ConfigLoader:    configLoader,
		StateLocker:     clistate.NewNoopLocker(),
		View:            view,
		DependencyLocks: depLocks,
	}, configCleanup, done
}

// testRefreshState is just a common state that we use for testing refresh.
func testRefreshState() *states.State {
	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_instance.foo").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"bar"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)
	return state
}

// refreshFixtureSchema returns a schema suitable for processing the
// configuration in testdata/refresh . This schema should be
// assigned to a mock provider named "test".
func refreshFixtureSchema() providers.ProviderSchema {
	return providers.ProviderSchema{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"ami": {Type: cty.String, Optional: true},
						"id":  {Type: cty.String, Computed: true},
					},
				},
			},
		},
	}
}
