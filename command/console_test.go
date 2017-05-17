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
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
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
			ContextOpts: testCtxConfig(p),
			Ui:          ui,
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
