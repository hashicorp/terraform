// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/cli"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
)

func TestProviders(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := os.Chdir(testFixturePath("providers/basic")); err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.Chdir(cwd)

	ui := new(cli.MockUi)
	c := &ProvidersCommand{
		Meta: Meta{
			Ui: ui,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	wantOutput := []string{
		"provider[registry.terraform.io/hashicorp/foo]",
		"provider[registry.terraform.io/hashicorp/bar]",
		"provider[registry.terraform.io/hashicorp/baz]",
	}

	output := ui.OutputWriter.String()
	for _, want := range wantOutput {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %s:\n%s", want, output)
		}
	}
}

func TestProviders_noConfigs(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := os.Chdir(testFixturePath("")); err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.Chdir(cwd)

	ui := new(cli.MockUi)
	c := &ProvidersCommand{
		Meta: Meta{
			Ui: ui,
		},
	}

	args := []string{}
	if code := c.Run(args); code == 0 {
		t.Fatal("expected command to return non-zero exit code" +
			" when no configs are available")
	}

	output := ui.ErrorWriter.String()
	expectedErrMsg := "No configuration files"
	if !strings.Contains(output, expectedErrMsg) {
		t.Errorf("Expected error message: %s\nGiven output: %s", expectedErrMsg, output)
	}
}

func TestProviders_modules(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("providers/modules"), td)
	t.Chdir(td)

	// first run init with mock provider sources to install the module
	initUi := new(cli.MockUi)
	view, _ := testView(t)
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"foo": {"1.0.0"},
		"bar": {"2.0.0"},
		"baz": {"1.2.2"},
	})
	defer close()
	m := Meta{
		testingOverrides: metaOverridesForProvider(testProvider()),
		Ui:               initUi,
		View:             view,
		ProviderSource:   providerSource,
	}
	ic := &InitCommand{
		Meta: m,
	}
	if code := ic.Run([]string{}); code != 0 {
		t.Fatalf("init failed\n%s", initUi.ErrorWriter)
	}

	// Providers command
	ui := new(cli.MockUi)
	c := &ProvidersCommand{
		Meta: Meta{
			Ui: ui,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	wantOutput := []string{
		"provider[registry.terraform.io/hashicorp/foo] 1.0.0", // from required_providers
		"provider[registry.terraform.io/hashicorp/bar] 2.0.0", // from provider config
		"── module.kiddo",                               // tree node for child module
		"provider[registry.terraform.io/hashicorp/baz]", // implied by a resource in the child module
	}

	output := ui.OutputWriter.String()
	for _, want := range wantOutput {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %s:\n%s", want, output)
		}
	}
}

func TestProviders_state(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := os.Chdir(testFixturePath("providers/state")); err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.Chdir(cwd)

	ui := new(cli.MockUi)
	c := &ProvidersCommand{
		Meta: Meta{
			Ui: ui,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	wantOutput := []string{
		"provider[registry.terraform.io/hashicorp/foo] 1.0.0", // from required_providers
		"provider[registry.terraform.io/hashicorp/bar] 2.0.0", // from a provider config block
		"Providers required by state",                         // header for state providers
		"provider[registry.terraform.io/hashicorp/baz]",       // from a resouce in state (only)
	}

	output := ui.OutputWriter.String()
	for _, want := range wantOutput {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %s:\n%s", want, output)
		}
	}
}

func TestProviders_tests(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := os.Chdir(testFixturePath("providers/tests")); err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.Chdir(cwd)

	ui := new(cli.MockUi)
	c := &ProvidersCommand{
		Meta: Meta{
			Ui: ui,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	wantOutput := []string{
		"test.main",
		"provider[registry.terraform.io/hashicorp/bar]",
	}

	output := ui.OutputWriter.String()
	for _, want := range wantOutput {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %s:\n%s", want, output)
		}
	}
}

func TestProviders_state_withStateStore(t *testing.T) {
	// State with a 'baz' provider not in the config
	originalState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "baz_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("baz"),
				Module:   addrs.RootModule,
			},
		)
	})

	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("state-store-unchanged"), td)
	t.Chdir(td)

	// Get bytes describing the state
	var stateBuf bytes.Buffer
	if err := statefile.Write(statefile.New(originalState, "", 1), &stateBuf); err != nil {
		t.Fatalf("error during test setup: %s", err)
	}

	// Create a mock that contains a persisted "default" state that uses the bytes from above.
	mockProvider := mockPluggableStateStorageProvider()
	mockProvider.MockStates = map[string]interface{}{
		"default": stateBuf.Bytes(),
	}
	mockProviderAddress := addrs.NewDefaultProvider("test")

	ui := new(cli.MockUi)
	c := &ProvidersCommand{
		Meta: Meta{
			Ui:                        ui,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	wantOutput := []string{
		"Providers required by configuration:",
		"└── provider[registry.terraform.io/hashicorp/test] 1.2.3",
		"Providers required by state:",
		"provider[registry.terraform.io/hashicorp/baz]",
	}

	output := ui.OutputWriter.String()
	for _, want := range wantOutput {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %s:\n%s", want, output)
		}
	}
}
