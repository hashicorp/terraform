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
	tempDir, err := fmtFixtureWriteDir()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.RemoveAll(tempDir)

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
	tempDir, err := fmtFixtureWriteDir()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.RemoveAll(tempDir)

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
	tempDir, err := fmtFixtureWriteDir()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.RemoveAll(tempDir)

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
	tempDir, err := fmtFixtureWriteDir()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.RemoveAll(tempDir)

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

func TestFmtPopulate(t *testing.T) {
	cases := []struct {
		Name     string
		Input    string
		Expected string
	}{
		{
			Name: "basic single-line double slash with extra detail",
			Input: `
// Foo.
//
// Bar.
variable "foo" {
  default = "bar"
}
			`,
			Expected: `
// Foo.
//
// Bar.
variable "foo" {
  description = "Foo."

  default = "bar"
}
			`,
		},
		{
			Name: "double slash, multi-line first block",
			Input: `
// Foo. Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod
// tempor incididunt ut labore et dolore magna aliqua.
//
// Bar.
variable "foo" {
  default = "bar"
}
			`,
			Expected: `
// Foo. Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod
// tempor incididunt ut labore et dolore magna aliqua.
//
// Bar.
variable "foo" {
  description = "Foo. Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua."

  default = "bar"
}
			`,
		},
		{
			Name: "slashes, fancy bordered comment",
			Input: `
///////////////////////////////////////////////////////////////////////////////
// Foo. Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod
// tempor incididunt ut labore et dolore magna aliqua.
//
// Bar.
///////////////////////////////////////////////////////////////////////////////
variable "foo" {
  default = "bar"
}
			`,
			Expected: `
///////////////////////////////////////////////////////////////////////////////
// Foo. Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod
// tempor incididunt ut labore et dolore magna aliqua.
//
// Bar.
///////////////////////////////////////////////////////////////////////////////
variable "foo" {
  description = "Foo. Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua."

  default = "bar"
}
			`,
		},
		{
			Name: "basic single-line hash with extra detail",
			Input: `
# Foo.
#
# Bar.
variable "foo" {
  default = "bar"
}
			`,
			Expected: `
# Foo.
#
# Bar.
variable "foo" {
  description = "Foo."

  default = "bar"
}
			`,
		},
		{
			Name: "hash, multi-line first block",
			Input: `
# Foo. Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod
# tempor incididunt ut labore et dolore magna aliqua.
#
# Bar.
variable "foo" {
  default = "bar"
}
			`,
			Expected: `
# Foo. Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod
# tempor incididunt ut labore et dolore magna aliqua.
#
# Bar.
variable "foo" {
  description = "Foo. Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua."

  default = "bar"
}
			`,
		},
		{
			Name: "slashes, fancy bordered comment",
			Input: `
###############################################################################
# Foo. Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod
# tempor incididunt ut labore et dolore magna aliqua.
#
# Bar.
###############################################################################
variable "foo" {
  default = "bar"
}
			`,
			Expected: `
###############################################################################
# Foo. Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod
# tempor incididunt ut labore et dolore magna aliqua.
#
# Bar.
###############################################################################
variable "foo" {
  description = "Foo. Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua."

  default = "bar"
}
			`,
		},
		{
			Name: "basic single-line slash-star with extra detail",
			Input: `
/*
 * Foo.
 *
 * Bar.
 */
variable "foo" {
  default = "bar"
}
			`,
			Expected: `
/*
 * Foo.
 *
 * Bar.
 */
variable "foo" {
  description = "Foo."

  default = "bar"
}
			`,
		},
		{
			Name: "slash-star, multi-line",
			Input: `
/*
 * Foo. Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod
 * tempor incididunt ut labore et dolore magna aliqua.
 *
 * Bar.
 */
variable "foo" {
  default = "bar"
}
			`,
			Expected: `
/*
 * Foo. Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod
 * tempor incididunt ut labore et dolore magna aliqua.
 *
 * Bar.
 */
variable "foo" {
  description = "Foo. Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua."

  default = "bar"
}
			`,
		},
		{
			Name: "slash-star, fancy bordered comment",
			Input: `
/*****************************************************************************
 * Foo. Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod
 * tempor incididunt ut labore et dolore magna aliqua.
 *
 * Bar.
 *****************************************************************************/
variable "foo" {
  default = "bar"
}
			`,
			Expected: `
/*****************************************************************************
 * Foo. Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod
 * tempor incididunt ut labore et dolore magna aliqua.
 *
 * Bar.
 *****************************************************************************/
variable "foo" {
  description = "Foo. Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua."

  default = "bar"
}
			`,
		},
		{
			Name: "slash-star, with no prefix (legit slash-star use)",
			Input: `
/*
Foo. Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod
tempor incididunt ut labore et dolore magna aliqua.

Bar.
 */
variable "foo" {
  default = "bar"
}
			`,
			Expected: `
/*
Foo. Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod
tempor incididunt ut labore et dolore magna aliqua.

Bar.
 */
variable "foo" {
  description = "Foo. Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua."

  default = "bar"
}
			`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			input := new(bytes.Buffer)
			input.WriteString(strings.TrimSpace(tc.Input))

			ui := new(cli.MockUi)
			c := &FmtCommand{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(testProvider()),
					Ui:               ui,
				},
				input: input,
			}

			args := []string{"-populate=true", "-"}
			if code := c.Run(args); code != 0 {
				t.Fatalf("wrong exit code. errors: \n%s", ui.ErrorWriter.String())
			}

			expected := []byte(strings.TrimSpace(tc.Expected) + "\n")
			actual := ui.OutputWriter.Bytes()

			if !bytes.Equal(expected, actual) {
				t.Fatalf("expected:\n%sgot:\n%s", expected, actual)
			}
		})
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

func fmtFixtureWriteDir() (string, error) {
	dir, err := ioutil.TempDir("", "tf")
	if err != nil {
		return "", err
	}

	err = ioutil.WriteFile(filepath.Join(dir, fmtFixture.filename), fmtFixture.input, 0644)
	if err != nil {
		os.RemoveAll(dir)
		return "", err
	}

	return dir, nil
}
