// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hashicorp/cli"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/terminal"
)

func TestStatePull(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("state-pull-backend"), td)
	t.Chdir(td)

	p := testProvider()
	ui := cli.NewMockUi()
	c := &StatePullCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	expectedResource := `
    {
      "mode": "managed",
      "type": "null_resource",
      "name": "a",
      "provider": "provider[\"registry.terraform.io/-/null\"]",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "id": "8521602373864259745",
            "triggers": null
          },
          "sensitive_attributes": [],
          "identity_schema_version": 0
        }
      ]
    }
`
	actual := ui.OutputWriter.String()
	if !strings.Contains(actual, expectedResource) {
		t.Fatalf("expected state to contain: %s\n\nstate:%s", expectedResource, actual)
	}
}

// Tests using `terraform state pull` subcommand in combination with pluggable state storage
//
// Note: Whereas other tests in this file use the local backend and require a state file in the test fixures,
// with pluggable state storage we can define the state via the mocked provider.
func TestStatePull_stateStore(t *testing.T) {
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
	stateBytes := stateBuf.Bytes()

	// Create a mock that contains a persisted "default" state that uses the bytes from above.
	mockProvider := mockPluggableStateStorageProvider()
	mockProvider.MockStates = map[string]any{
		"default": stateBytes,
	}
	mockProviderAddress := addrs.NewDefaultProvider("test")
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.0.0"},
	})
	defer close()

	ui := cli.NewMockUi()
	streams, _ := terminal.StreamsForTesting(t)
	c := &StatePullCommand{
		Meta: Meta{
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			ProviderSource: providerSource,
			Ui:             ui,
			Streams:        streams,
		},
	}

	// `terraform show` command specifying a given resource addr
	expectedResourceAddr := "test_instance.foo"
	args := []string{expectedResourceAddr}
	code := c.Run(args)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Test that the state in the output matches the original state
	expectedResource := `
    {
      "mode": "managed",
      "type": "test_instance",
      "name": "foo",
      "provider": "provider[\"registry.terraform.io/hashicorp/test\"]",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "input": "foobar"
          },
          "sensitive_attributes": [],
          "identity_schema_version": 0
        }
      ]
    }
`
	actual := ui.OutputWriter.String()
	if !strings.Contains(actual, expectedResource) {
		t.Fatalf("expected state to contain: %s\n\nstate:%s", expectedResource, actual)
	}
}

func TestStatePull_noState(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)

	p := testProvider()
	ui := cli.NewMockUi()
	c := &StatePullCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	actual := ui.OutputWriter.String()
	if actual != "" {
		t.Fatalf("bad: %s", actual)
	}
}

func TestStatePull_checkRequiredVersion(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("command-check-required-version"), td)
	t.Chdir(td)

	p := testProvider()
	ui := cli.NewMockUi()
	c := &StatePullCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 1 {
		t.Fatalf("got exit status %d; want 1\nstderr:\n%s\n\nstdout:\n%s", code, ui.ErrorWriter.String(), ui.OutputWriter.String())
	}

	// Required version diags are correct
	errStr := ui.ErrorWriter.String()
	if !strings.Contains(errStr, `required_version = "~> 0.9.0"`) {
		t.Fatalf("output should point to unmet version constraint, but is:\n\n%s", errStr)
	}
	if strings.Contains(errStr, `required_version = ">= 0.13.0"`) {
		t.Fatalf("output should not point to met version constraint, but is:\n\n%s", errStr)
	}
}
