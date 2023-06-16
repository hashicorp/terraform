// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"os"
	"strings"
	"testing"

	"github.com/mitchellh/cli"
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
	defer testChdir(t, td)()

	// first run init with mock provider sources to install the module
	initUi := new(cli.MockUi)
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"foo": {"1.0.0"},
		"bar": {"2.0.0"},
		"baz": {"1.2.2"},
	})
	defer close()
	m := Meta{
		testingOverrides: metaOverridesForProvider(testProvider()),
		Ui:               initUi,
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
		"provider[registry.terraform.io/hashicorp/foo]",
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
