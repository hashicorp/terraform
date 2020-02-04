package command

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchellh/cli"
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
	ui := new(cli.MockUi)
	c := &ConsoleCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
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
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	// Write a terraform.tvars
	varFilePath := filepath.Join(tmp, "terraform.tfvars")
	if err := ioutil.WriteFile(varFilePath, []byte(applyVarFile), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ConsoleCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	var output bytes.Buffer
	defer testStdinPipe(t, strings.NewReader("var.foo\n"))()
	outCloser := testStdoutCapture(t, &output)

	args := []string{
		testFixturePath("apply-vars"),
	}
	code := c.Run(args)
	outCloser()
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	actual := output.String()
	if actual != "bar\n" {
		t.Fatalf("bad: %q", actual)
	}
}

func TestConsole_unsetRequiredVars(t *testing.T) {
	// This test is verifying that it's possible to run "terraform console"
	// without providing values for all required variables, without
	// "terraform console" producing an interactive prompt for those variables
	// or producing errors. Instead, it should allow evaluation in that
	// partial context but see the unset variables values as being unknown.

	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ConsoleCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	var output bytes.Buffer
	defer testStdinPipe(t, strings.NewReader("var.foo\n"))()
	outCloser := testStdoutCapture(t, &output)

	args := []string{
		// This test fixture includes variable "foo" {}, which we are
		// intentionally not setting here.
		testFixturePath("apply-vars"),
	}
	code := c.Run(args)
	outCloser()

	// Because we're running "terraform console" in piped input mode, we're
	// expecting it to return a nonzero exit status here but the message
	// must be the one indicating that it did attempt to evaluate var.foo and
	// got an unknown value in return, rather than an error about var.foo
	// not being set or a failure to prompt for it.
	if code == 0 {
		t.Fatalf("unexpected success\n%s", ui.OutputWriter.String())
	}

	// The error message should be the one console produces when it encounters
	// an unknown value.
	got := ui.ErrorWriter.String()
	want := `Error: Result depends on values that cannot be determined`
	if !strings.Contains(got, want) {
		t.Fatalf("wrong output\ngot:\n%s\n\nwant string containing %q", got, want)
	}
}
