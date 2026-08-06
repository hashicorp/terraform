// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/cli"
	testing_command "github.com/hashicorp/terraform/internal/command/testing"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/zclconf/go-cty/cty"
)

// ConsoleCommand is tested primarily with tests in the "repl" package.
// It is not tested here because the Console uses a readline-like library
// that takes over stdin/stdout. It is difficult to test directly. The
// core logic is tested in "repl"
//
// This file still contains some tests using the stdin-based input.

func TestConsole_basic(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)

	p := testProvider()
	ui := testUiWrapped(t)
	view, _ := testView(t)
	c := &ConsoleCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	var output bytes.Buffer
	defer testStdinPipe(t, strings.NewReader("1+5\n"))()
	outCloser := testStdoutCapture(t, &output)

	args := []string{}
	code := c.Run(args)
	outCloser()
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	actual := output.String()
	if actual != "6\n" {
		t.Fatalf("bad: %q", actual)
	}
}

func TestConsole_tfvars(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply-vars"), td)
	t.Chdir(td)

	// Write a terraform.tvars
	varFilePath := filepath.Join(td, "terraform.tfvars")
	if err := ioutil.WriteFile(varFilePath, []byte(applyVarFile), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}

	p := testProvider()
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"value": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}
	ui := testUiWrapped(t)
	view, _ := testView(t)
	c := &ConsoleCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	var output bytes.Buffer
	defer testStdinPipe(t, strings.NewReader("var.foo\n"))()
	outCloser := testStdoutCapture(t, &output)

	args := []string{}
	code := c.Run(args)
	outCloser()
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	actual := output.String()
	if actual != "\"bar\"\n" {
		t.Fatalf("bad: %q", actual)
	}
}

func TestConsole_unsetRequiredVars(t *testing.T) {
	// This test is verifying that it's possible to run "terraform console"
	// without providing values for all required variables, without
	// "terraform console" producing an interactive prompt for those variables
	// or producing errors. Instead, it should allow evaluation in that
	// partial context but see the unset variables values as being unknown.
	//
	// This test fixture includes variable "foo" {}, which we are
	// intentionally not setting here.
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply-vars"), td)
	t.Chdir(td)

	p := testProvider()
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"value": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}
	ui := testUiWrapped(t)
	view, _ := testView(t)
	c := &ConsoleCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	var output bytes.Buffer
	defer testStdinPipe(t, strings.NewReader("var.foo\n"))()
	outCloser := testStdoutCapture(t, &output)

	args := []string{}
	code := c.Run(args)
	outCloser()

	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if got, want := output.String(), "(known after apply)\n"; got != want {
		t.Fatalf("unexpected output\n got: %q\nwant: %q", got, want)
	}
}

func TestConsole_variables(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("variables"), td)
	t.Chdir(td)

	p := testProvider()
	ui := testUiWrapped(t)
	view, _ := testView(t)
	c := &ConsoleCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	commands := map[string]string{
		"var.foo\n":          "\"bar\"\n",
		"var.snack\n":        "\"popcorn\"\n",
		"var.secret_snack\n": "(sensitive value)\n",
		"local.snack_bar\n":  "[\n  \"popcorn\",\n  (sensitive value),\n]\n",
	}

	args := []string{}

	for cmd, val := range commands {
		var output bytes.Buffer
		defer testStdinPipe(t, strings.NewReader(cmd))()
		outCloser := testStdoutCapture(t, &output)
		code := c.Run(args)
		outCloser()
		if code != 0 {
			t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
		}

		actual := output.String()
		if output.String() != val {
			t.Fatalf("bad: %q, expected %q", actual, val)
		}
	}
}

func TestConsole_modules(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("modules"), td)
	t.Chdir(td)

	p := applyFixtureProvider()
	ui := testUiWrapped(t)
	view, _ := testView(t)

	c := &ConsoleCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	commands := map[string]string{
		"module.child.myoutput\n":          "\"bar\"\n",
		"module.count_child[0].myoutput\n": "\"bar\"\n",
		"local.foo\n":                      "3\n",
	}

	args := []string{}

	for cmd, val := range commands {
		var output bytes.Buffer
		defer testStdinPipe(t, strings.NewReader(cmd))()
		outCloser := testStdoutCapture(t, &output)
		code := c.Run(args)
		outCloser()
		if code != 0 {
			t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
		}

		actual := output.String()
		if output.String() != val {
			t.Fatalf("bad: %q, expected %q", actual, val)
		}
	}
}

func TestConsole_modulesPlan(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply"), td)
	t.Chdir(td)

	p := applyFixtureProvider()
	ui := testUiWrapped(t)
	view, _ := testView(t)

	c := &ConsoleCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	commands := map[string]string{
		"test_instance.foo.ami\n": "\"bar\"\n",
	}

	// The -plan option means that we'll be evaluating expressions against
	// a plan constructed from this configuration, instead of against its
	// (non-existent) prior state.
	args := []string{"-plan"}

	for cmd, val := range commands {
		var output bytes.Buffer
		defer testStdinPipe(t, strings.NewReader(cmd))()
		outCloser := testStdoutCapture(t, &output)
		code := c.Run(args)
		outCloser()
		if code != 0 {
			t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
		}

		actual := output.String()
		if output.String() != val {
			t.Fatalf("bad: %q, expected %q", actual, val)
		}
	}
}

func TestConsole_scope(t *testing.T) {
	testCases := map[string]struct {
		args                 []string
		expectedConsoleEvals map[string]string
	}{
		"root-module": {
			expectedConsoleEvals: map[string]string{
				"var.root\n":   "\"variable data\"\n",
				"local.root\n": "\"variable data -> local\"\n",
				// resource data is unknown as the state of this resource doesn't exist
				"test_resource.root.value\n":                    "(known after apply)\n",
				"module.child_single.child_output\n":            "\"child module output\"\n",
				"module.child_foreach[\"key1\"].child_output\n": "\"child module output\"\n",
				"module.child_foreach[\"key2\"].child_output\n": "\"child module output\"\n",
				"module.child_count[0].child_output\n":          "\"child module output\"\n",
				"module.child_count[1].child_output\n":          "\"child module output\"\n",
			},
		},
		"root-module-planeval": {
			args: []string{"-plan"},
			expectedConsoleEvals: map[string]string{
				"var.root\n":                                    "\"variable data\"\n",
				"local.root\n":                                  "\"variable data -> local\"\n",
				"test_resource.root.value\n":                    "\"resource attr set to local -> variable data -> local\"\n",
				"module.child_single.child_output\n":            "\"child module output\"\n",
				"module.child_foreach[\"key1\"].child_output\n": "\"child module output\"\n",
				"module.child_foreach[\"key2\"].child_output\n": "\"child module output\"\n",
				"module.child_count[0].child_output\n":          "\"child module output\"\n",
				"module.child_count[1].child_output\n":          "\"child module output\"\n",
			},
		},
		"child-single-module": {
			args: []string{"-scope=module.child_single"},
			expectedConsoleEvals: map[string]string{
				"var.child_input\n": "\"variable data -> child\"\n",
				"local.child\n":     "\"variable data -> child -> local\"\n",
				// resource data is unknown as the state of this resource doesn't exist
				"test_resource.child.value\n":                    "(known after apply)\n",
				"module.grandchild[\"key\"].grandchild_output\n": "\"grandchild module output\"\n",
			},
		},
		"child-single-module-planeval": {
			args: []string{"-plan", "-scope=module.child_single"},
			expectedConsoleEvals: map[string]string{
				"var.child_input\n":                              "\"variable data -> child\"\n",
				"local.child\n":                                  "\"variable data -> child -> local\"\n",
				"test_resource.child.value\n":                    "\"resource attr set to local -> variable data -> child -> local\"\n",
				"module.grandchild[\"key\"].grandchild_output\n": "\"grandchild module output\"\n",
			},
		},
		"child-foreach-module": {
			args: []string{`-scope=module.child_foreach["key1"]`},
			expectedConsoleEvals: map[string]string{
				"var.child_input\n": "\"variable data -> child[key1]\"\n",
				"local.child\n":     "\"variable data -> child[key1] -> local\"\n",
				// resource data is unknown as the state of this resource doesn't exist
				"test_resource.child.value\n":                    "(known after apply)\n",
				"module.grandchild[\"key\"].grandchild_output\n": "\"grandchild module output\"\n",
			},
		},
		"child-foreach-module-planeval": {
			args: []string{"-plan", `-scope=module.child_foreach["key2"]`},
			expectedConsoleEvals: map[string]string{
				"var.child_input\n":                              "\"variable data -> child[key2]\"\n",
				"local.child\n":                                  "\"variable data -> child[key2] -> local\"\n",
				"test_resource.child.value\n":                    "\"resource attr set to local -> variable data -> child[key2] -> local\"\n",
				"module.grandchild[\"key\"].grandchild_output\n": "\"grandchild module output\"\n",
			},
		},
		"child-count-module": {
			args: []string{`-scope=module.child_count[0]`},
			expectedConsoleEvals: map[string]string{
				"var.child_input\n": "\"variable data -> child[0]\"\n",
				"local.child\n":     "\"variable data -> child[0] -> local\"\n",
				// resource data is unknown as the state of this resource doesn't exist
				"test_resource.child.value\n":                    "(known after apply)\n",
				"module.grandchild[\"key\"].grandchild_output\n": "\"grandchild module output\"\n",
			},
		},
		"child-count-module-planeval": {
			args: []string{"-plan", `-scope=module.child_count[1]`},
			expectedConsoleEvals: map[string]string{
				"var.child_input\n":                              "\"variable data -> child[1]\"\n",
				"local.child\n":                                  "\"variable data -> child[1] -> local\"\n",
				"test_resource.child.value\n":                    "\"resource attr set to local -> variable data -> child[1] -> local\"\n",
				"module.grandchild[\"key\"].grandchild_output\n": "\"grandchild module output\"\n",
			},
		},
		"grandchild-module": {
			args: []string{`-scope=module.child_single.module.grandchild["key"]`},
			expectedConsoleEvals: map[string]string{
				"var.grandchild_input\n": "\"variable data -> child -> grandchild[key]\"\n",
				"local.grandchild\n":     "\"variable data -> child -> grandchild[key] -> local\"\n",
				// resource data is unknown as the state of this resource doesn't exist
				"test_resource.grandchild.value\n": "(known after apply)\n",
			},
		},
		"grandchild-module-planeval": {
			args: []string{"-plan", `-scope=module.child_count[0].module.grandchild["key"]`},
			expectedConsoleEvals: map[string]string{
				"var.grandchild_input\n":           "\"variable data -> child[0] -> grandchild[key]\"\n",
				"local.grandchild\n":               "\"variable data -> child[0] -> grandchild[key] -> local\"\n",
				"test_resource.grandchild.value\n": "\"resource attr set to local -> variable data -> child[0] -> grandchild[key] -> local\"\n",
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			td := t.TempDir()
			testCopyDir(t, testFixturePath("console-nested-module-scopes"), td)
			t.Chdir(td)

			ui := cli.NewMockUi()
			view, _ := testView(t)
			p := testing_command.NewProvider(&testing_command.ResourceStore{}).Provider

			c := &ConsoleCommand{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(p),
					Ui:               ui,
					View:             view,
				},
			}

			tc.args = append(tc.args, []string{"-var", "root=variable data"}...)

			for cmd, val := range tc.expectedConsoleEvals {
				var output bytes.Buffer
				defer testStdinPipe(t, strings.NewReader(cmd))()
				outCloser := testStdoutCapture(t, &output)
				code := c.Run(tc.args)
				outCloser()
				if code != 0 {
					t.Errorf("bad exit code: %d\n\n%s", code, ui.ErrorWriter.String())
				}

				actual := output.String()
				if output.String() != val {
					t.Errorf("bad result for %q: %q, expected %q", cmd, actual, val)
				}
			}
		})
	}
}

func TestConsole_scope_errors(t *testing.T) {
	testCases := map[string]struct {
		args          []string
		expectedError string
	}{
		"invalid-traversal": {
			args:          []string{"-scope=foo."},
			expectedError: `Dot must be followed by attribute name.`,
		},
		"invalid-module-address-syntax": {
			args:          []string{"-scope=test_resource.root"},
			expectedError: `A module instance address must begin with "module."`,
		},
		"module-not-found": {
			args:          []string{"-scope=module.nonexistent"},
			expectedError: `The module address "module.nonexistent" does not have an evaluation scope.`,
		},
		"module-not-found-planeval": {
			args:          []string{"-plan", "-scope=module.nonexistent[0]"},
			expectedError: `The module address "module.nonexistent[0]" does not have an evaluation scope.`,
		},
		"module-instance-not-found": {
			args:          []string{"-scope=module.child_count[2]"},
			expectedError: `The module address "module.child_count[2]" does not have an evaluation scope.`,
		},
		"module-instance-not-found-planeval": {
			args:          []string{"-plan", `-scope=module.child_foreach["key3"]`},
			expectedError: "The module address \"module.child_foreach[\"key3\"]\" does not have an evaluation\nscope.",
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			td := t.TempDir()
			testCopyDir(t, testFixturePath("console-nested-module-scopes"), td)
			t.Chdir(td)

			ui := cli.NewMockUi()
			view, _ := testView(t)
			p := testing_command.NewProvider(&testing_command.ResourceStore{}).Provider

			c := &ConsoleCommand{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(p),
					View:             view,
					Ui:               ui,
				},
			}

			tc.args = append(tc.args, []string{"-var", "root=variable data"}...)

			defer testStdinPipe(t, strings.NewReader(""))()
			code := c.Run(tc.args)
			if code == 0 {
				t.Fatal("expected a non-zero exit code with error")
			}

			actual := ui.ErrorWriter.String()
			if !strings.Contains(actual, tc.expectedError) {
				t.Fatalf("expected error to include %q, but got:\n%s", tc.expectedError, actual)
			}
		})
	}
}
