// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/cli"
)

func TestFmt_MockDataFiles(t *testing.T) {
	const inSuffix = "_in.tfmock.hcl"
	const outSuffix = "_out.tfmock.hcl"
	const gotSuffix = "_got.tfmock.hcl"
	entries, err := ioutil.ReadDir("testdata/tfmock-fmt")
	if err != nil {
		t.Fatal(err)
	}

	tmpDir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	for _, info := range entries {
		if info.IsDir() {
			continue
		}
		filename := info.Name()
		if !strings.HasSuffix(filename, inSuffix) {
			continue
		}
		testName := filename[:len(filename)-len(inSuffix)]
		t.Run(testName, func(t *testing.T) {
			inFile := filepath.Join("testdata", "tfmock-fmt", testName+inSuffix)
			wantFile := filepath.Join("testdata", "tfmock-fmt", testName+outSuffix)
			gotFile := filepath.Join(tmpDir, testName+gotSuffix)
			input, err := ioutil.ReadFile(inFile)
			if err != nil {
				t.Fatal(err)
			}
			want, err := ioutil.ReadFile(wantFile)
			if err != nil {
				t.Fatal(err)
			}
			err = ioutil.WriteFile(gotFile, input, 0700)
			if err != nil {
				t.Fatal(err)
			}

			ui := cli.NewMockUi()
			c := &FmtCommand{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(testProvider()),
					Ui:               ui,
				},
			}
			args := []string{gotFile}
			if code := c.Run(args); code != 0 {
				t.Fatalf("fmt command was unsuccessful:\n%s", ui.ErrorWriter.String())
			}

			got, err := ioutil.ReadFile(gotFile)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(string(want), string(got)); diff != "" {
				t.Errorf("wrong result\n%s", diff)
			}
		})
	}
}

func TestFmt_TestFiles(t *testing.T) {
	const inSuffix = "_in.tftest.hcl"
	const outSuffix = "_out.tftest.hcl"
	const gotSuffix = "_got.tftest.hcl"
	entries, err := ioutil.ReadDir("testdata/tftest-fmt")
	if err != nil {
		t.Fatal(err)
	}

	tmpDir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	for _, info := range entries {
		if info.IsDir() {
			continue
		}
		filename := info.Name()
		if !strings.HasSuffix(filename, inSuffix) {
			continue
		}
		testName := filename[:len(filename)-len(inSuffix)]
		t.Run(testName, func(t *testing.T) {
			inFile := filepath.Join("testdata", "tftest-fmt", testName+inSuffix)
			wantFile := filepath.Join("testdata", "tftest-fmt", testName+outSuffix)
			gotFile := filepath.Join(tmpDir, testName+gotSuffix)
			input, err := ioutil.ReadFile(inFile)
			if err != nil {
				t.Fatal(err)
			}
			want, err := ioutil.ReadFile(wantFile)
			if err != nil {
				t.Fatal(err)
			}
			err = ioutil.WriteFile(gotFile, input, 0700)
			if err != nil {
				t.Fatal(err)
			}

			ui := cli.NewMockUi()
			c := &FmtCommand{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(testProvider()),
					Ui:               ui,
				},
			}
			args := []string{gotFile}
			if code := c.Run(args); code != 0 {
				t.Fatalf("fmt command was unsuccessful:\n%s", ui.ErrorWriter.String())
			}

			got, err := ioutil.ReadFile(gotFile)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(string(want), string(got)); diff != "" {
				t.Errorf("wrong result\n%s", diff)
			}
		})
	}
}

func TestFmt(t *testing.T) {
	const inSuffix = "_in.tf"
	const outSuffix = "_out.tf"
	const gotSuffix = "_got.tf"
	entries, err := ioutil.ReadDir("testdata/fmt")
	if err != nil {
		t.Fatal(err)
	}

	tmpDir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	for _, info := range entries {
		if info.IsDir() {
			continue
		}
		filename := info.Name()
		if !strings.HasSuffix(filename, inSuffix) {
			continue
		}
		testName := filename[:len(filename)-len(inSuffix)]
		t.Run(testName, func(t *testing.T) {
			inFile := filepath.Join("testdata", "fmt", testName+inSuffix)
			wantFile := filepath.Join("testdata", "fmt", testName+outSuffix)
			gotFile := filepath.Join(tmpDir, testName+gotSuffix)
			input, err := ioutil.ReadFile(inFile)
			if err != nil {
				t.Fatal(err)
			}
			want, err := ioutil.ReadFile(wantFile)
			if err != nil {
				t.Fatal(err)
			}
			err = ioutil.WriteFile(gotFile, input, 0700)
			if err != nil {
				t.Fatal(err)
			}

			ui := cli.NewMockUi()
			c := &FmtCommand{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(testProvider()),
					Ui:               ui,
				},
			}
			args := []string{gotFile}
			if code := c.Run(args); code != 0 {
				t.Fatalf("fmt command was unsuccessful:\n%s", ui.ErrorWriter.String())
			}

			got, err := ioutil.ReadFile(gotFile)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(string(want), string(got)); diff != "" {
				t.Errorf("wrong result\n%s", diff)
			}
		})
	}
}

func TestFmt_nonexist(t *testing.T) {
	tempDir := fmtFixtureWriteDir(t)

	ui := new(cli.MockUi)
	c := &FmtCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	missingDir := filepath.Join(tempDir, "doesnotexist")
	args := []string{missingDir}
	if code := c.Run(args); code != 2 {
		t.Fatalf("wrong exit code. errors: \n%s", ui.ErrorWriter.String())
	}

	expected := "No file or directory at"
	if actual := ui.ErrorWriter.String(); !strings.Contains(actual, expected) {
		t.Fatalf("expected:\n%s\n\nto include: %q", actual, expected)
	}
}

func TestFmt_syntaxError(t *testing.T) {
	tempDir := testTempDir(t)

	invalidSrc := `
a = 1 +
`

	err := ioutil.WriteFile(filepath.Join(tempDir, "invalid.tf"), []byte(invalidSrc), 0644)
	if err != nil {
		t.Fatal(err)
	}

	ui := new(cli.MockUi)
	c := &FmtCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{tempDir}
	if code := c.Run(args); code != 2 {
		t.Fatalf("wrong exit code. errors: \n%s", ui.ErrorWriter.String())
	}

	expected := "Invalid expression"
	if actual := ui.ErrorWriter.String(); !strings.Contains(actual, expected) {
		t.Fatalf("expected:\n%s\n\nto include: %q", actual, expected)
	}
}

func TestFmt_snippetInError(t *testing.T) {
	tempDir := testTempDir(t)

	backendSrc := `terraform {backend "s3" {}}`

	err := ioutil.WriteFile(filepath.Join(tempDir, "backend.tf"), []byte(backendSrc), 0644)
	if err != nil {
		t.Fatal(err)
	}

	ui := new(cli.MockUi)
	c := &FmtCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{tempDir}
	if code := c.Run(args); code != 2 {
		t.Fatalf("wrong exit code. errors: \n%s", ui.ErrorWriter.String())
	}

	substrings := []string{
		"Argument definition required",
		"line 1, in terraform",
		`1: terraform {backend "s3" {}}`,
	}
	for _, substring := range substrings {
		if actual := ui.ErrorWriter.String(); !strings.Contains(actual, substring) {
			t.Errorf("expected:\n%s\n\nto include: %q", actual, substring)
		}
	}
}

func TestFmt_manyArgs(t *testing.T) {
	tempDir := fmtFixtureWriteDir(t)
	// Add a second file
	secondSrc := `locals { x = 1 }`

	err := ioutil.WriteFile(filepath.Join(tempDir, "second.tf"), []byte(secondSrc), 0644)
	if err != nil {
		t.Fatal(err)
	}

	ui := new(cli.MockUi)
	c := &FmtCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{
		filepath.Join(tempDir, "main.tf"),
		filepath.Join(tempDir, "second.tf"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("wrong exit code. errors: \n%s", ui.ErrorWriter.String())
	}

	got, err := filepath.Abs(strings.TrimSpace(ui.OutputWriter.String()))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(tempDir, fmtFixture.filename)

	if got != want {
		t.Fatalf("wrong output\ngot:  %s\nwant: %s", got, want)
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

	got, err := filepath.Abs(strings.TrimSpace(ui.OutputWriter.String()))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(tempDir, fmtFixture.filename)

	if got != want {
		t.Fatalf("wrong output\ngot:  %s\nwant: %s", got, want)
	}
}

func TestFmt_fileArg(t *testing.T) {
	tempDir := fmtFixtureWriteDir(t)

	ui := new(cli.MockUi)
	c := &FmtCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{filepath.Join(tempDir, fmtFixture.filename)}
	if code := c.Run(args); code != 0 {
		t.Fatalf("wrong exit code. errors: \n%s", ui.ErrorWriter.String())
	}

	got, err := filepath.Abs(strings.TrimSpace(ui.OutputWriter.String()))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(tempDir, fmtFixture.filename)

	if got != want {
		t.Fatalf("wrong output\ngot:  %s\nwant: %s", got, want)
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

	// Given that we give relative paths back to the user, normalize this temp
	// dir so that we're comparing against a relative-ized (normalized) path
	tempDir = c.normalizePath(tempDir)

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

func TestFmt_providerReqs(t *testing.T) {
	runFmt := func(t *testing.T, args ...string) string {
		ui := cli.NewMockUi()
		c := &FmtCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(testProvider()),
				Ui:               ui,
			},
		}
		if code := c.Run(args); code != 0 {
			t.Fatalf("unexpected failure\nerrors:\n%s", ui.ErrorWriter.String())
		}
		return ui.OutputWriter.String()
	}

	t.Run("no existing terraform blocks at all and no versions.tf", func(t *testing.T) {
		workDir := t.TempDir()
		err := os.WriteFile(filepath.Join(workDir, "main.tf"), []byte(`
			resource "tls_thingy" "placeholder" {
			}
			resource "local_thingy" "placeholder" {
			}
		`), os.ModePerm)
		if err != nil {
			t.Fatal(err)
		}

		runFmt(t, "-write", workDir)

		got, err := os.ReadFile(filepath.Join(workDir, "versions.tf"))
		if err != nil {
			t.Fatalf("can't open versions.tf: %s", err)
		}
		want := `terraform {
  required_providers {
    local = {
      source = "hashicorp/local"
    }
    tls = {
      source = "hashicorp/tls"
    }
  }
}
`
		if diff := cmp.Diff(want, string(got)); diff != "" {
			t.Errorf("wrong versions.tf content\n%s", diff)
		}
	})
	t.Run("versions.tf has existing terraform block but no required_providers", func(t *testing.T) {
		workDir := t.TempDir()
		err := os.WriteFile(filepath.Join(workDir, "main.tf"), []byte(`
			resource "tls_thingy" "placeholder" {
			}
		`), os.ModePerm)
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(filepath.Join(workDir, "versions.tf"), []byte(`
			terraform {
				required_version = ">= 1.0.0"
			}
		`), os.ModePerm)
		if err != nil {
			t.Fatal(err)
		}

		runFmt(t, "-write", workDir)

		got, err := os.ReadFile(filepath.Join(workDir, "versions.tf"))
		if err != nil {
			t.Fatalf("can't open versions.tf: %s", err)
		}
		want := `terraform {
  required_version = ">= 1.0.0"
  required_providers {
    tls = {
      source = "hashicorp/tls"
    }
  }
}`
		if diff := cmp.Diff(want, string(bytes.TrimSpace(got))); diff != "" {
			t.Errorf("wrong versions.tf content\n%s", diff)
		}
	})
	t.Run("another file has existing terraform block but no required_providers", func(t *testing.T) {
		workDir := t.TempDir()
		err := os.WriteFile(filepath.Join(workDir, "main.tf"), []byte(`
			terraform {
				required_version = ">= 1.0.0"
			}

			resource "tls_thingy" "placeholder" {
			}
		`), os.ModePerm)
		if err != nil {
			t.Fatal(err)
		}

		runFmt(t, "-write", workDir)

		// Because the existing terraform block wasn't in versions.tf and
		// didn't have a nested required_providers block already, we
		// don't presume that the existing block would be a good home for
		// our new required_providers entries. (Not all "terraform" blocks
		// are intended for dependency-related declarations.)
		got, err := os.ReadFile(filepath.Join(workDir, "versions.tf"))
		if err != nil {
			t.Fatalf("can't open versions.tf: %s", err)
		}
		want := `terraform {
  required_providers {
    tls = {
      source = "hashicorp/tls"
    }
  }
}`
		if diff := cmp.Diff(want, string(bytes.TrimSpace(got))); diff != "" {
			t.Errorf("wrong versions.tf content\n%s", diff)
		}
	})
	t.Run("versions.tf already declares another provider", func(t *testing.T) {
		workDir := t.TempDir()
		err := os.WriteFile(filepath.Join(workDir, "main.tf"), []byte(`
			resource "tls_thingy" "placeholder" {
			}
		`), os.ModePerm)
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(filepath.Join(workDir, "versions.tf"), []byte(`
			terraform {
				required_providers {
					anyother = {
						source = "example.com/foo/bar"
					}
				}
			}
		`), os.ModePerm)
		if err != nil {
			t.Fatal(err)
		}

		runFmt(t, "-write", workDir)

		got, err := os.ReadFile(filepath.Join(workDir, "versions.tf"))
		if err != nil {
			t.Fatalf("can't open versions.tf: %s", err)
		}
		want := `terraform {
  required_providers {
    anyother = {
      source = "example.com/foo/bar"
    }
    tls = {
      source = "hashicorp/tls"
    }
  }
}`
		if diff := cmp.Diff(want, string(bytes.TrimSpace(got))); diff != "" {
			t.Errorf("wrong versions.tf content\n%s", diff)
		}
	})
	t.Run("versions.tf already declares another provider in an unconventional file", func(t *testing.T) {
		workDir := t.TempDir()
		err := os.WriteFile(filepath.Join(workDir, "main.tf"), []byte(`
			terraform {
				required_providers {
					anyother = {
						source = "example.com/foo/bar"
					}
				}
			}

			resource "tls_thingy" "placeholder" {
			}
		`), os.ModePerm)
		if err != nil {
			t.Fatal(err)
		}

		runFmt(t, "-write", workDir)

		// If the author decided to use a different file for required_providers
		// then we preserve that preference and skip generating version.tf.
		_, err = os.ReadFile(filepath.Join(workDir, "versions.tf"))
		if !os.IsNotExist(err) {
			t.Error("versions.tf was generated, but should not have been")
		}

		got, err := os.ReadFile(filepath.Join(workDir, "main.tf"))
		if err != nil {
			t.Fatalf("can't open main.tf: %s", err)
		}
		want := `terraform {
  required_providers {
    anyother = {
      source = "example.com/foo/bar"
    }
    tls = {
      source = "hashicorp/tls"
    }
  }
}

resource "tls_thingy" "placeholder" {
}`
		if diff := cmp.Diff(want, string(bytes.TrimSpace(got))); diff != "" {
			t.Errorf("wrong main.tf content\n%s", diff)
		}
	})
	t.Run("module already declares all of the providers it should", func(t *testing.T) {
		workDir := t.TempDir()
		err := os.WriteFile(filepath.Join(workDir, "main.tf"), []byte(`
			terraform {
				required_providers {
					tls = {
						source = "hashicorp/tls"
					}
				}
			}

			resource "tls_thingy" "placeholder" {
			}
		`), os.ModePerm)
		if err != nil {
			t.Fatal(err)
		}

		runFmt(t, "-write", workDir)

		_, err = os.ReadFile(filepath.Join(workDir, "versions.tf"))
		if !os.IsNotExist(err) {
			t.Error("versions.tf was generated, but should not have been")
		}

		got, err := os.ReadFile(filepath.Join(workDir, "main.tf"))
		if err != nil {
			t.Fatalf("can't open main.tf: %s", err)
		}
		want := `terraform {
  required_providers {
    tls = {
      source = "hashicorp/tls"
    }
  }
}

resource "tls_thingy" "placeholder" {
}`
		if diff := cmp.Diff(want, string(bytes.TrimSpace(got))); diff != "" {
			t.Errorf("wrong main.tf content\n%s", diff)
		}
	})
	t.Run("required_providers is already in a .tf.json file", func(t *testing.T) {
		workDir := t.TempDir()
		err := os.WriteFile(filepath.Join(workDir, "main.tf"), []byte(`
			resource "tls_thingy" "placeholder" {
			}
		`), os.ModePerm)
		if err != nil {
			t.Fatal(err)
		}
		versionsJSON := []byte(`
			{
				"terraform": {
					"required_providers": {
					}
				}
			}
		`)
		err = os.WriteFile(filepath.Join(workDir, "versions.tf.json"), versionsJSON, os.ModePerm)
		if err != nil {
			t.Fatal(err)
		}

		runFmt(t, "-write", workDir)

		// No versions.tf should have been generated, because a module can
		// only have one required_providers block and this one is in a file
		// we cannot format.
		_, err = os.ReadFile(filepath.Join(workDir, "versions.tf"))
		if !os.IsNotExist(err) {
			t.Error("versions.tf was generated, but should not have been")
		}

		// The JSON file should not have been modified either.
		got, err := os.ReadFile(filepath.Join(workDir, "versions.tf.json"))
		if err != nil {
			t.Fatalf("can't open main.tf: %s", err)
		}
		want := versionsJSON
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong versions.tf.json content\n%s", diff)
		}
	})

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
