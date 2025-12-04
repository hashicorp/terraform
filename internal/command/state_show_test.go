// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hashicorp/cli"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/zclconf/go-cty/cty"
)

func TestStateShow(t *testing.T) {
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar","foo":"value","bar":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})
	statePath := testStateFile(t, state)

	p := testProvider()
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id":  {Type: cty.String, Optional: true, Computed: true},
						"foo": {Type: cty.String, Optional: true},
						"bar": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}

	streams, done := terminal.StreamsForTesting(t)
	c := &StateShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Streams:          streams,
		},
	}

	args := []string{
		"-state", statePath,
		"test_instance.foo",
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	// Test that outputs were displayed
	expected := strings.TrimSpace(testStateShowOutput) + "\n"
	actual := output.Stdout()
	if actual != expected {
		t.Fatalf("Expected:\n%q\n\nTo equal:\n%q", actual, expected)
	}
}

func TestStateShow_errorMarshallingState(t *testing.T) {
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo_invalid",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				// The error is caused by the state containing attributes that don't
				// match the schema for the resource.
				AttrsJSON: []byte(`{"non_existent_attr":"I'm gonna cause an error!"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})
	statePath := testStateFile(t, state)

	p := testProvider()
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id":  {Type: cty.String, Optional: true, Computed: true},
						"foo": {Type: cty.String, Optional: true},
						"bar": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}

	streams, done := terminal.StreamsForTesting(t)
	c := &StateShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Streams:          streams,
		},
	}

	args := []string{
		"-state", statePath,
		"test_instance.foo_invalid",
	}
	code := c.Run(args)
	output := done(t)
	if code != 1 {
		t.Fatalf("unexpected code: %d\n\n%s", code, output.Stdout())
	}

	// Test that error outputs were displayed
	expected := "unsupported attribute \"non_existent_attr\""
	actual := output.Stderr()
	if !strings.Contains(actual, expected) {
		t.Fatalf("Expected stderr output to include:\n%q\n\n Instead got:\n%q", expected, actual)
	}
}

func TestStateShow_multi(t *testing.T) {
	submod, _ := addrs.ParseModuleInstanceStr("module.sub")
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar","foo":"value","bar":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(submod),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"foo","foo":"value","bar":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   submod.Module(),
			},
		)
	})
	statePath := testStateFile(t, state)

	p := testProvider()
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id":  {Type: cty.String, Optional: true, Computed: true},
						"foo": {Type: cty.String, Optional: true},
						"bar": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}

	streams, done := terminal.StreamsForTesting(t)
	c := &StateShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Streams:          streams,
		},
	}

	args := []string{
		"-state", statePath,
		"test_instance.foo",
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	// Test that outputs were displayed
	expected := strings.TrimSpace(testStateShowOutput) + "\n"
	actual := output.Stdout()
	if actual != expected {
		t.Fatalf("Expected:\n%q\n\nTo equal:\n%q", actual, expected)
	}
}

func TestStateShow_noState(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)

	p := testProvider()
	streams, done := terminal.StreamsForTesting(t)
	c := &StateShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Streams:          streams,
		},
	}

	args := []string{
		"test_instance.foo",
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: %d", code)
	}
	output := done(t)
	if !strings.Contains(output.Stderr(), "No state file was found!") {
		t.Fatalf("expected a no state file error, got: %s", output.Stderr())
	}
}

func TestStateShow_emptyState(t *testing.T) {
	state := states.NewState()
	statePath := testStateFile(t, state)

	p := testProvider()
	streams, done := terminal.StreamsForTesting(t)
	c := &StateShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Streams:          streams,
		},
	}

	args := []string{
		"-state", statePath,
		"test_instance.foo",
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: %d", code)
	}
	output := done(t)
	if !strings.Contains(output.Stderr(), "No instance found for the given address!") {
		t.Fatalf("expected a no instance found error, got: %s", output.Stderr())
	}
}

func TestStateShow_configured_provider(t *testing.T) {
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar","foo":"value","bar":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test-beta"),
				Module:   addrs.RootModule,
			},
		)
	})
	statePath := testStateFile(t, state)

	p := testProvider()
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id":  {Type: cty.String, Optional: true, Computed: true},
						"foo": {Type: cty.String, Optional: true},
						"bar": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}

	streams, done := terminal.StreamsForTesting(t)
	c := &StateShowCommand{
		Meta: Meta{
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					addrs.NewDefaultProvider("test-beta"): providers.FactoryFixed(p),
				},
			},
			Streams: streams,
		},
	}

	args := []string{
		"-state", statePath,
		"test_instance.foo",
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	// Test that outputs were displayed
	expected := strings.TrimSpace(testStateShowOutput) + "\n"
	actual := output.Stdout()
	if actual != expected {
		t.Fatalf("Expected:\n%q\n\nTo equal:\n%q", actual, expected)
	}
}

// Tests using `terraform state show` subcommand in combination with pluggable state storage
//
// Note: Whereas other tests in this file use the local backend and require a state file in the test fixures,
// with pluggable state storage we can define the state via the mocked provider.
func TestStateShow_stateStore(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("state-store-unchanged"), td)
	t.Chdir(td)

	// Get bytes describing a state containing a resource
	state := states.NewState()
	rootModule := state.RootModule()
	rootModule.SetResourceInstanceCurrent(
		addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_instance",
			Name: "foo",
		}.Instance(addrs.NoKey),
		&states.ResourceInstanceObjectSrc{
			Status: states.ObjectReady,
			AttrsJSON: []byte(`{
				"input": "foobar"
			}`),
		},
		addrs.AbsProviderConfig{
			Provider: addrs.NewDefaultProvider("test"),
			Module:   addrs.RootModule,
		},
	)
	var stateBuf bytes.Buffer
	if err := statefile.Write(statefile.New(state, "", 1), &stateBuf); err != nil {
		t.Fatalf("error during test setup: %s", err)
	}

	// Create a mock that contains a persisted "default" state that uses the bytes from above.
	mockProvider := mockPluggableStateStorageProvider()
	mockProvider.MockStates = map[string]interface{}{
		"default": stateBuf.Bytes(),
	}
	mockProviderAddress := addrs.NewDefaultProvider("test")

	ui := cli.NewMockUi()
	streams, done := terminal.StreamsForTesting(t)
	c := &StateShowCommand{
		Meta: Meta{
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			Ui:      ui,
			Streams: streams,
		},
	}

	// `terraform show` command specifying a given resource addr
	expectedResourceAddr := "test_instance.foo"
	args := []string{expectedResourceAddr}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	// Test that outputs were displayed
	expected := "# test_instance.foo:\nresource \"test_instance\" \"foo\" {\n    input = \"foobar\"\n}\n"
	actual := output.Stdout()
	if actual != expected {
		t.Fatalf("Expected:\n%q\n\nTo equal: %q", actual, expected)
	}
}

const testStateShowOutput = `
# test_instance.foo:
resource "test_instance" "foo" {
    bar = "value"
    foo = "value"
    id  = "bar"
}
`
