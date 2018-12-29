package command

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchellh/cli"
)

func TestFmt_errorReporting(t *testing.T) {
	tempDir := fmtFixtureWriteDir(t)

	ui := new(cli.MockUi)
	c := &FmtCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	dummy_file := filepath.Join(tempDir, "doesnotexist")
	args := []string{dummy_file}
	if code := c.Run(args); code != 2 {
		t.Fatalf("wrong exit code. errors: \n%s", ui.ErrorWriter.String())
	}

	expected := fmt.Sprintf("Error running fmt: stat %s: no such file or directory", dummy_file)
	if actual := ui.ErrorWriter.String(); !strings.Contains(actual, expected) {
		t.Fatalf("expected:\n%s\n\nto include: %q", actual, expected)
	}
}

func TestFmt_tooManyArgs(t *testing.T) {
	ui := new(cli.MockUi)
	c := &FmtCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{
		"one",
		"two",
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("wrong exit code. errors: \n%s", ui.ErrorWriter.String())
	}

	expected := "The fmt command expects at most one argument."
	if actual := ui.ErrorWriter.String(); !strings.Contains(actual, expected) {
		t.Fatalf("expected:\n%s\n\nto include: %q", actual, expected)
	}
}

func TestFmt_workingDirectory(t *testing.T) {
	tempDir := fmtFixtureWriteDir(t)

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.Chdir(cwd)

	ui := new(cli.MockUi)
	c := &FmtCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("wrong exit code. errors: \n%s", ui.ErrorWriter.String())
	}

	expected := fmt.Sprintf("%s\n", fmtFixture.filename)
	if actual := ui.OutputWriter.String(); actual != expected {
		t.Fatalf("got: %q\nexpected: %q", actual, expected)
	}
}

func TestFmt_directoryArg(t *testing.T) {
	tempDir := fmtFixtureWriteDir(t)

	ui := new(cli.MockUi)
	c := &FmtCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{tempDir}
	if code := c.Run(args); code != 0 {
		t.Fatalf("wrong exit code. errors: \n%s", ui.ErrorWriter.String())
	}

	expected := fmt.Sprintf("%s\n", filepath.Join(tempDir, fmtFixture.filename))
	if actual := ui.OutputWriter.String(); actual != expected {
		t.Fatalf("got: %q\nexpected: %q", actual, expected)
	}
}

func TestFmt_stdinArg(t *testing.T) {
	input := new(bytes.Buffer)
	input.Write(fmtFixture.input)

	ui := new(cli.MockUi)
	c := &FmtCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
		input: input,
	}

	args := []string{"-"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("wrong exit code. errors: \n%s", ui.ErrorWriter.String())
	}

	expected := fmtFixture.golden
	if actual := ui.OutputWriter.Bytes(); !bytes.Equal(actual, expected) {
		t.Fatalf("got: %q\nexpected: %q", actual, expected)
	}
}

func TestFmt_nonDefaultOptions(t *testing.T) {
	tempDir := fmtFixtureWriteDir(t)

	ui := new(cli.MockUi)
	c := &FmtCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{
		"-list=false",
		"-write=false",
		"-diff",
		tempDir,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("wrong exit code. errors: \n%s", ui.ErrorWriter.String())
	}

	expected := fmt.Sprintf("-%s+%s", fmtFixture.input, fmtFixture.golden)
	if actual := ui.OutputWriter.String(); !strings.Contains(actual, expected) {
		t.Fatalf("expected:\n%s\n\nto include: %q", actual, expected)
	}
}

func TestFmt_check(t *testing.T) {
	tempDir := fmtFixtureWriteDir(t)

	ui := new(cli.MockUi)
	c := &FmtCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{
		"-check",
		tempDir,
	}
	if code := c.Run(args); code != 3 {
		t.Fatalf("wrong exit code. expected 3")
	}

	if actual := ui.OutputWriter.String(); !strings.Contains(actual, tempDir) {
		t.Fatalf("expected:\n%s\n\nto include: %q", actual, tempDir)
	}
}

func TestFmt_checkStdin(t *testing.T) {
	input := new(bytes.Buffer)
	input.Write(fmtFixture.input)

	ui := new(cli.MockUi)
	c := &FmtCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
		input: input,
	}

	args := []string{
		"-check",
		"-",
	}
	if code := c.Run(args); code != 3 {
		t.Fatalf("wrong exit code. expected 3, got %d", code)
	}

	if ui.OutputWriter != nil {
		t.Fatalf("expected no output, got: %q", ui.OutputWriter.String())
	}
}

var fmtFixture = struct {
	filename      string
	input, golden []byte
}{
	"main.tf",
	[]byte(`  foo  =  "bar"
`),
	[]byte(`foo = "bar"
`),
}

func fmtFixtureWriteDir(t *testing.T) string {
	dir := testTempDir(t)

	err := ioutil.WriteFile(filepath.Join(dir, fmtFixture.filename), fmtFixture.input, 0644)
	if err != nil {
		t.Fatal(err)
	}

	return dir
}
