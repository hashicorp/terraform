package command

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/mitchellh/cli"
	"github.com/zclconf/go-cty/cty"
)

// ConsoleCommand is tested primarily with tests in the "repl" package.
// It is not tested here because the Console uses a readline-like library
// that takes over stdin/stdout. It is difficult to test directly. The
// core logic is tested in "repl"
//
// This file still contains some tests using the stdin-based input.

func TestConsole_basic(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	p := testProvider()
	ui := cli.NewMockUi()
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
	td := tempDir(t)
	testCopyDir(t, testFixturePath("apply-vars"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Write a terraform.tvars
	varFilePath := filepath.Join(td, "terraform.tfvars")
	if err := ioutil.WriteFile(varFilePath, []byte(applyVarFile), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}

	p := testProvider()
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"value": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}
	ui := cli.NewMockUi()
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
	td := tempDir(t)
	testCopyDir(t, testFixturePath("apply-vars"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	p := testProvider()
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"value": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}
	ui := cli.NewMockUi()
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
	td := tempDir(t)
	testCopyDir(t, testFixturePath("variables"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	p := testProvider()
	ui := cli.NewMockUi()
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
		"var.secret_snack\n": "(sensitive)\n",
		"local.snack_bar\n":  "[\n  \"popcorn\",\n  (sensitive),\n]\n",
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
	td := tempDir(t)
	testCopyDir(t, testFixturePath("modules"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	p := applyFixtureProvider()
	ui := cli.NewMockUi()
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
