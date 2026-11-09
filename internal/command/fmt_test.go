// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFmt_MockDataFiles(t *testing.T) {
	t.Parallel()
	const inSuffix = "_in.tfmock.hcl"
	const outSuffix = "_out.tfmock.hcl"
	const gotSuffix = "_got.tfmock.hcl"
	entries, err := os.ReadDir("testdata/tfmock-fmt")
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
			t.Parallel()
			inFile := filepath.Join("testdata", "tfmock-fmt", testName+inSuffix)
			wantFile := filepath.Join("testdata", "tfmock-fmt", testName+outSuffix)
			gotFile := filepath.Join(tmpDir, testName+gotSuffix)
			input, err := os.ReadFile(inFile)
			if err != nil {
				t.Fatal(err)
			}
			want, err := os.ReadFile(wantFile)
			if err != nil {
				t.Fatal(err)
			}
			err = os.WriteFile(gotFile, input, 0700)
			if err != nil {
				t.Fatal(err)
			}

			view, done := testView(t)
			c := &FmtCommand{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(testProvider()),
					View:             view,
				},
			}
			args := []string{gotFile}
			if code := c.Run(args); code != 0 {
				t.Fatalf("fmt command was unsuccessful:\n%s", done(t).Stderr())
			}

			got, err := os.ReadFile(gotFile)
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
	t.Parallel()
	const inSuffix = "_in.tftest.hcl"
	const outSuffix = "_out.tftest.hcl"
	const gotSuffix = "_got.tftest.hcl"
	entries, err := os.ReadDir("testdata/tftest-fmt")
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
			t.Parallel()
			inFile := filepath.Join("testdata", "tftest-fmt", testName+inSuffix)
			wantFile := filepath.Join("testdata", "tftest-fmt", testName+outSuffix)
			gotFile := filepath.Join(tmpDir, testName+gotSuffix)
			input, err := os.ReadFile(inFile)
			if err != nil {
				t.Fatal(err)
			}
			want, err := os.ReadFile(wantFile)
			if err != nil {
				t.Fatal(err)
			}
			err = os.WriteFile(gotFile, input, 0700)
			if err != nil {
				t.Fatal(err)
			}

			view, done := testView(t)
			c := &FmtCommand{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(testProvider()),
					View:             view,
				},
			}
			args := []string{gotFile}
			if code := c.Run(args); code != 0 {
				t.Fatalf("fmt command was unsuccessful:\n%s", done(t).Stderr())
			}

			got, err := os.ReadFile(gotFile)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(string(want), string(got)); diff != "" {
				t.Errorf("wrong result\n%s", diff)
			}
		})
	}
}

func TestFmt_QueryFiles(t *testing.T) {
	t.Parallel()
	const inSuffix = "_in.tfquery.hcl"
	const outSuffix = "_out.tfquery.hcl"
	const gotSuffix = "_got.tfquery.hcl"
	entries, err := os.ReadDir("testdata/tfquery-fmt")
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
			t.Parallel()
			inFile := filepath.Join("testdata", "tfquery-fmt", testName+inSuffix)
			wantFile := filepath.Join("testdata", "tfquery-fmt", testName+outSuffix)
			gotFile := filepath.Join(tmpDir, testName+gotSuffix)
			input, err := os.ReadFile(inFile)
			if err != nil {
				t.Fatal(err)
			}
			want, err := os.ReadFile(wantFile)
			if err != nil {
				t.Fatal(err)
			}
			err = os.WriteFile(gotFile, input, 0700)
			if err != nil {
				t.Fatal(err)
			}

			view, done := testView(t)
			c := &FmtCommand{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(testProvider()),
					View:             view,
				},
			}
			args := []string{gotFile}
			if code := c.Run(args); code != 0 {
				t.Fatalf("fmt command was unsuccessful:\n%s", done(t).Stderr())
			}

			got, err := os.ReadFile(gotFile)
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
	t.Parallel()
	const inSuffix = "_in.tf"
	const outSuffix = "_out.tf"
	const gotSuffix = "_got.tf"
	entries, err := os.ReadDir("testdata/fmt")
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
			t.Parallel()
			inFile := filepath.Join("testdata", "fmt", testName+inSuffix)
			wantFile := filepath.Join("testdata", "fmt", testName+outSuffix)
			gotFile := filepath.Join(tmpDir, testName+gotSuffix)
			input, err := os.ReadFile(inFile)
			if err != nil {
				t.Fatal(err)
			}
			want, err := os.ReadFile(wantFile)
			if err != nil {
				t.Fatal(err)
			}
			err = os.WriteFile(gotFile, input, 0700)
			if err != nil {
				t.Fatal(err)
			}

			view, done := testView(t)
			c := &FmtCommand{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(testProvider()),
					View:             view,
				},
			}
			args := []string{gotFile}
			if code := c.Run(args); code != 0 {
				t.Fatalf("fmt command was unsuccessful:\n%s", done(t).Stderr())
			}

			got, err := os.ReadFile(gotFile)
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
	t.Parallel()

	tempDir := fmtFixtureWriteDir(t)

	view, done := testView(t)
	c := &FmtCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			View:             view,
		},
	}

	missingDir := filepath.Join(tempDir, "doesnotexist")
	args := []string{missingDir}
	if code := c.Run(args); code != 2 {
		t.Fatalf("wrong exit code. errors: \n%s", done(t).Stderr())
	}
	output := done(t)

	expected := "No file or directory at"
	if actual := output.Stderr(); !strings.Contains(actual, expected) {
		t.Fatalf("expected:\n%s\n\nto include: %q", actual, expected)
	}
}

func TestFmt_syntaxError(t *testing.T) {
	t.Parallel()

	tempDir := testTempDir(t)

	invalidSrc := `
a = 1 +
`

	err := os.WriteFile(filepath.Join(tempDir, "invalid.tf"), []byte(invalidSrc), 0644)
	if err != nil {
		t.Fatal(err)
	}

	view, done := testView(t)
	c := &FmtCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			View:             view,
		},
	}

	args := []string{tempDir}
	if code := c.Run(args); code != 2 {
		t.Fatalf("wrong exit code. errors: \n%s", done(t).Stderr())
	}
	output := done(t)

	expected := "Invalid expression"
	if actual := output.Stderr(); !strings.Contains(actual, expected) {
		t.Fatalf("expected:\n%s\n\nto include: %q", actual, expected)
	}
}

func TestFmt_snippetInError(t *testing.T) {
	t.Parallel()

	tempDir := testTempDir(t)

	backendSrc := `terraform {backend "s3" {}}`

	err := os.WriteFile(filepath.Join(tempDir, "backend.tf"), []byte(backendSrc), 0644)
	if err != nil {
		t.Fatal(err)
	}

	view, done := testView(t)
	c := &FmtCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			View:             view,
		},
	}

	args := []string{"-no-color", tempDir}
	if code := c.Run(args); code != 2 {
		t.Fatalf("wrong exit code. errors: \n%s", done(t).Stderr())
	}
	output := done(t)

	substrings := []string{
		"Argument definition required",
		"line 1, in terraform",
		`terraform {backend "s3" {}}`,
	}
	for _, substring := range substrings {
		if actual := output.Stderr(); !strings.Contains(actual, substring) {
			t.Errorf("expected:\n%s\n\nto include: %q", actual, substring)
		}
	}
}

func TestFmt_manyArgs(t *testing.T) {
	t.Parallel()

	tempDir := fmtFixtureWriteDir(t)
	// Add a second file
	secondSrc := `locals { x = 1 }`

	err := os.WriteFile(filepath.Join(tempDir, "second.tf"), []byte(secondSrc), 0644)
	if err != nil {
		t.Fatal(err)
	}

	view, done := testView(t)
	c := &FmtCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			View:             view,
		},
	}

	args := []string{
		filepath.Join(tempDir, "main.tf"),
		filepath.Join(tempDir, "second.tf"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("wrong exit code. errors: \n%s", done(t).Stderr())
	}

	output := done(t)
	got, err := filepath.Abs(strings.TrimSpace(output.Stdout()))
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

	view, done := testView(t)
	c := &FmtCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			View:             view,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("wrong exit code. errors: \n%s", done(t).Stderr())
	}

	expected := fmt.Sprintf("%s\n", fmtFixture.filename)
	if actual := done(t).Stdout(); actual != expected {
		t.Fatalf("got: %q\nexpected: %q", actual, expected)
	}
}

func TestFmt_directoryArg(t *testing.T) {
	t.Parallel()
	tempDir := fmtFixtureWriteDir(t)

	view, done := testView(t)
	c := &FmtCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			View:             view,
		},
	}

	args := []string{tempDir}
	if code := c.Run(args); code != 0 {
		t.Fatalf("wrong exit code. errors: \n%s", done(t).Stderr())
	}

	output := done(t)
	got, err := filepath.Abs(strings.TrimSpace(output.Stdout()))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(tempDir, fmtFixture.filename)

	if got != want {
		t.Fatalf("wrong output\ngot:  %s\nwant: %s", got, want)
	}
}

func TestFmt_fileArg(t *testing.T) {
	t.Parallel()
	tempDir := fmtFixtureWriteDir(t)

	view, done := testView(t)
	c := &FmtCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			View:             view,
		},
	}

	args := []string{filepath.Join(tempDir, fmtFixture.filename)}
	if code := c.Run(args); code != 0 {
		t.Fatalf("wrong exit code. errors: \n%s", done(t).Stderr())
	}

	output := done(t)
	got, err := filepath.Abs(strings.TrimSpace(output.Stdout()))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(tempDir, fmtFixture.filename)

	if got != want {
		t.Fatalf("wrong output\ngot:  %s\nwant: %s", got, want)
	}
}

func TestFmt_stdinArg(t *testing.T) {
	t.Parallel()
	input := new(bytes.Buffer)
	input.Write(fmtFixture.input)

	view, done := testView(t)
	c := &FmtCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			View:             view,
		},
		input: input,
	}

	args := []string{"-"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("wrong exit code. errors: \n%s", done(t).Stderr())
	}

	expected := fmtFixture.golden
	actual := done(t).Stdout()
	if actual := []byte(actual); !bytes.Equal(actual, expected) {
		t.Fatalf("got: %q\nexpected: %q", actual, expected)
	}
}

// This test defines current behaviour, and doesn't mean this is correct behaviour.
// In future we should add validation in the arguments packages to prevent this being
// silently ignored, but that will be a breaking change.
func TestFmt_stdinArgIgnoresFurtherArguments(t *testing.T) {
	t.Parallel()
	input := new(bytes.Buffer)
	input.Write(fmtFixture.input)

	view, done := testView(t)
	c := &FmtCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			View:             view,
		},
		input: input,
	}

	args := []string{
		"-",
		"does-not-exist.tf", // Ignored, so it doesn't cause usual error "No file or directory at does-not-exist.tf"
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("wrong exit code. errors: \n%s", done(t).Stderr())
	}

	if actual := []byte(done(t).Stdout()); !bytes.Equal(actual, fmtFixture.golden) {
		t.Fatalf("got: %q\nexpected: %q", actual, fmtFixture.golden)
	}
}

func TestFmt_nonDefaultOptions(t *testing.T) {
	t.Parallel()
	tempDir := fmtFixtureWriteDir(t)

	view, done := testView(t)
	c := &FmtCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			View:             view,
		},
	}

	args := []string{
		"-list=false",
		"-write=false",
		"-diff",
		tempDir,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("wrong exit code. errors: \n%s", done(t).Stderr())
	}

	expected := fmt.Sprintf("-%s+%s", fmtFixture.input, fmtFixture.golden)
	if actual := done(t).Stdout(); !strings.Contains(actual, expected) {
		t.Fatalf("expected:\n%s\n\nto include: %q", actual, expected)
	}
}

// If a user has set options to prevent the command from producing output, either by:
// - list=false : Won't list files that need formatting
// - diff=false : Won't show diffs for files that changed
// - write=false : Won't write changes to files
// In these cases, the command falls back to printing the formatted content to stdout.
func TestFmt_formattedResultFallbackOutput(t *testing.T) {
	t.Parallel()
	tempDir := fmtFixtureWriteDir(t)

	view, done := testView(t)
	c := &FmtCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			View:             view,
		},
	}

	args := []string{
		"-list=false",
		"-diff=false",
		"-write=false",
		tempDir,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("wrong exit code. expected 0")
	}

	expectedOutput := "foo = \"bar\"\n"

	output := done(t)
	if actual := output.Stdout(); actual != expectedOutput {
		t.Fatalf("expected:\n%s\n\nto be:\n%s", actual, expectedOutput)
	}
	if actual := output.Stderr(); actual != "" {
		t.Fatalf("expected:\n%s\n\nto be empty.", actual)
	}
}

func TestFmt_check(t *testing.T) {
	t.Run("check option", func(t *testing.T) {
		t.Parallel()
		tempDir := fmtFixtureWriteDir(t)

		view, done := testView(t)
		c := &FmtCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(testProvider()),
				View:             view,
			},
		}

		args := []string{
			"-check",
			"-list=true", // -list defaults to true, but set explicitly here
			tempDir,
		}
		if code := c.Run(args); code != 3 {
			t.Fatalf("wrong exit code. expected 3")
		}

		// Given that we give relative paths back to the user, normalize this temp
		// dir so that we're comparing against a relative-ized (normalized) path
		tempDir = c.normalizePath(tempDir)
		expectedOutput := fmt.Sprintf("%s/main.tf\n", tempDir)

		output := done(t)
		if actual := output.Stdout(); actual != expectedOutput {
			t.Fatalf("expected:\n%s\n\nto be:\n%s", actual, expectedOutput)
		}
		if actual := output.Stderr(); actual != "" {
			t.Fatalf("expected:\n%s\n\nto be empty.", actual)
		}
	})

	t.Run("with list disabled", func(t *testing.T) {
		t.Parallel()
		tempDir := fmtFixtureWriteDir(t)

		view, done := testView(t)
		c := &FmtCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(testProvider()),
				View:             view,
			},
		}

		args := []string{
			"-check",
			"-list=false",
			tempDir,
		}
		if code := c.Run(args); code != 3 {
			t.Fatalf("wrong exit code. expected 3")
		}

		output := done(t)
		if actual := output.Stdout(); actual != "" {
			t.Fatalf("expected:\n%s\n\nto be empty.", actual)
		}
		if actual := output.Stderr(); actual != "" {
			t.Fatalf("expected:\n%s\n\nto be empty.", actual)
		}
	})

	t.Run("with diff option", func(t *testing.T) {
		t.Parallel()
		tempDir := fmtFixtureWriteDir(t)

		view, done := testView(t)
		c := &FmtCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(testProvider()),
				View:             view,
			},
		}

		args := []string{
			"-check",
			"-diff",
			tempDir,
		}
		if code := c.Run(args); code != 3 {
			t.Fatalf("wrong exit code. expected 3")
		}

		tempDir = c.normalizePath(tempDir)
		expectedOutput := fmt.Sprintf(`%s/main.tf
--- old/%s/main.tf
+++ new/%s/main.tf
@@ -1 +1 @@
-  foo  =  "bar"
+foo = "bar"
`, tempDir, tempDir, tempDir)

		output := done(t)
		if actual := output.Stdout(); actual != expectedOutput {
			t.Fatalf("expected:\n%s\n\ngot:\n%s", expectedOutput, actual)
		}
		if actual := output.Stderr(); actual != "" {
			t.Fatalf("expected:\n%s\n\nto be empty.", actual)
		}
	})

	t.Run("combining -diff and -list=false options", func(t *testing.T) {
		t.Parallel()
		tempDir := fmtFixtureWriteDir(t)

		view, done := testView(t)
		c := &FmtCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(testProvider()),
				View:             view,
			},
		}

		args := []string{
			"-check",
			"-diff",
			"-list=false",
			tempDir,
		}
		if code := c.Run(args); code != 3 {
			t.Fatalf("wrong exit code. expected 3")
		}

		tempDir = c.normalizePath(tempDir)
		// output doesn't have the file path printed before the diff, which is the -list contribution to output
		expectedOutput := fmt.Sprintf(`--- old/%s/main.tf
+++ new/%s/main.tf
@@ -1 +1 @@
-  foo  =  "bar"
+foo = "bar"
`, tempDir, tempDir)

		output := done(t)
		if actual := output.Stdout(); actual != expectedOutput {
			t.Fatalf("expected:\n%s\n\ngot:\n%s", expectedOutput, actual)
		}
		if actual := output.Stderr(); actual != "" {
			t.Fatalf("expected:\n%s\n\nto be empty.", actual)
		}
	})

	t.Run("explicit -write=true option is silently ignored when using the check option", func(t *testing.T) {
		t.Parallel()
		tempDir := fmtFixtureWriteDir(t)

		view, done := testView(t)
		c := &FmtCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(testProvider()),
				View:             view,
			},
		}

		args := []string{
			"-check",
			"-write=true", // Setting the default value, which is overridden when -check is used
			tempDir,
		}
		if code := c.Run(args); code != 3 {
			t.Fatalf("wrong exit code. expected 3")
		}

		tempDir = c.normalizePath(tempDir)
		expectedOutput := fmt.Sprintf("%s/main.tf\n", tempDir)

		output := done(t)
		if actual := output.Stdout(); actual != expectedOutput {
			t.Fatalf("expected:\n%s\n\ngot:\n%s", expectedOutput, actual)
		}
		if actual := output.Stderr(); actual != "" {
			t.Fatalf("expected:\n%s\n\nto be empty.", actual)
		}
	})

	t.Run("all outputs disabled, alongside -check option", func(t *testing.T) {
		t.Parallel()
		tempDir := fmtFixtureWriteDir(t)

		view, done := testView(t)
		c := &FmtCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(testProvider()),
				View:             view,
			},
		}

		args := []string{
			"-check",
			"-write=false",
			"-list=false",
			"-diff=false",
			tempDir,
		}
		if code := c.Run(args); code != 3 {
			t.Fatalf("wrong exit code. expected 3")
		}

		expectedOutput := ""

		output := done(t)
		if actual := output.Stdout(); actual != expectedOutput {
			t.Fatalf("expected:\n%s\n\ngot:\n%s", expectedOutput, actual)
		}
		if actual := output.Stderr(); actual != "" {
			t.Fatalf("expected:\n%s\n\nto be empty.", actual)
		}
	})
}

// When `fmt` encounters a symlink:
// * The target is edited
// * The symlink itself remains a symlink
// * The target file's permissions are unchanged
func TestFmt_symlinkedFileAreEdited(t *testing.T) {
	t.Run("root directory symlink", func(t *testing.T) {
		workingDir := tempWorkingDir(t)
		t.Chdir(workingDir.RootModuleDir())

		// Create the target of the symlink.
		otherDir := t.TempDir()
		target := filepath.Join(otherDir, "main.tf")
		cfg := `variable "a" {
default="x"
  type =    string
}

locals {
    b = var.a
c =   "y"
}`
		err := os.WriteFile(target, []byte(cfg), 0644)
		if err != nil {
			t.Fatal(err)
		}
		var perms os.FileMode = 0744
		if err := os.Chmod(target, perms); err != nil {
			t.Fatal(err)
		}

		// Create the symlink pointing to the target file.
		// Use a relative path for the target of the symlink.
		link := filepath.Join(workingDir.RootModuleDir(), "main.tf")
		relTarget, err := filepath.Rel(workingDir.RootModuleDir(), target)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(relTarget, link); err != nil {
			t.Fatal(err)
		}

		view, done := testView(t)
		c := &FmtCommand{
			Meta: Meta{
				View:       view,
				WorkingDir: workingDir,
			},
		}

		if code := c.Run([]string{"-no-color"}); code != 0 {
			t.Fatalf("unexpected failure with code %d: %s", code, done(t).Stderr())
		}

		// The symlink is still a symlink
		fi, err := os.Lstat(link)
		if err != nil {
			t.Fatalf("unexpected error %s", err)
		}
		if fi.Mode().Type() != os.ModeSymlink {
			t.Fatal("expected the working directory main.tf to still be a symlink")
		}

		// The target file's permissions are unchanged
		fi, err = os.Stat(target)
		if err != nil {
			t.Fatalf("unexpected error stating target file: %s", err)
		}
		if fi.Mode().Perm() != perms {
			t.Fatalf("expected target file permissions to be %o, got: %o", perms, fi.Mode().Perm())
		}

		// The symlinked file has been formatted
		view, done = testView(t)
		c = &FmtCommand{
			Meta: Meta{
				View:       view,
				WorkingDir: workingDir,
			},
		}
		args := []string{
			"-no-color",
			"-check",
		}
		if code := c.Run(args); code != 0 {
			output := done(t)
			t.Fatalf("expected `terraform fmt -check` to exit with code 0 due to the file being formatted, got code %d: stdout: %s\nstderr: %s", code, output.Stdout(), output.Stderr())
		}
	})

	t.Run("formatting a symlink from outside the root directory", func(t *testing.T) {
		workingDir := tempWorkingDir(t)

		// We don't change directory into workingDir

		// Create the target of the symlink.
		otherDir := t.TempDir()
		target := filepath.Join(otherDir, "main.tf")
		cfg := `variable "a" {
default="x"
  type =    string
}

locals {
    b = var.a
c =   "y"
}`
		err := os.WriteFile(target, []byte(cfg), 0644)
		if err != nil {
			t.Fatal(err)
		}
		var perms os.FileMode = 0744
		if err := os.Chmod(target, perms); err != nil {
			t.Fatal(err)
		}

		// Create the symlink pointing to the target file.
		// Use a relative path for the target of the symlink.
		link := filepath.Join(workingDir.RootModuleDir(), "main.tf")
		relTarget, err := filepath.Rel(workingDir.RootModuleDir(), target)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(relTarget, link); err != nil {
			t.Fatal(err)
		}

		view, done := testView(t)
		c := &FmtCommand{
			Meta: Meta{
				View: view,
				// Not setting WorkingDir
			},
		}

		args := []string{
			"-no-color",
			workingDir.RootModuleDir(),
		}
		if code := c.Run(args); code != 0 {
			t.Fatalf("unexpected failure with code %d: %s", code, done(t).Stderr())
		}

		// The symlink is still a symlink
		fi, err := os.Lstat(link)
		if err != nil {
			t.Fatalf("unexpected error %s", err)
		}
		if fi.Mode().Type() != os.ModeSymlink {
			t.Fatal("expected the working directory main.tf to still be a symlink")
		}

		// The target file's permissions are unchanged
		fi, err = os.Stat(target)
		if err != nil {
			t.Fatalf("unexpected error stating target file: %s", err)
		}
		if fi.Mode().Perm() != perms {
			t.Fatalf("expected target file permissions to be %o, got: %o", perms, fi.Mode().Perm())
		}

		// The symlinked file has been formatted
		view, done = testView(t)
		c = &FmtCommand{
			Meta: Meta{
				View: view,
				// Not setting WorkingDir
			},
		}
		args = []string{
			"-no-color",
			"-check",
			filepath.Join(workingDir.RootModuleDir(), "main.tf"),
		}
		if code := c.Run(args); code != 0 {
			output := done(t)
			t.Fatalf("expected `terraform fmt -check` to exit with code 0 due to the file being formatted, got code %d: stdout: %s\nstderr: %s", code, output.Stdout(), output.Stderr())
		}
	})

	t.Run("relative symlink in a nested directory", func(t *testing.T) {
		// This test ensures that a relative symlink is handled correctly and avoids an error
		// where the relative path is used relative to the root module instead of the directory
		// it is in.

		workingDir := tempWorkingDir(t)
		t.Chdir(workingDir.RootModuleDir())

		// Instead of using a test fixture, we create this structure using code
		// to ensure the symlink is correct:
		//
		// root/
		// └── main.tf
		// └── mod/
		//     └── target/
		//         └── main.tf
		//     └── main.tf (symlink to target/main.tf)
		rootDir := workingDir.RootModuleDir()
		moduleDir := filepath.Join(rootDir, "mod")
		targetDir := filepath.Join(moduleDir, "target")
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			t.Fatal(err)
		}

		cfg := `variable "a" {
default="x"
  type =    string
}

locals {
    b = var.a
c =   "y"
}`
		targetFile := filepath.Join(targetDir, "main.tf")
		var perms os.FileMode = 0744
		if err := os.WriteFile(targetFile, []byte(cfg), perms); err != nil {
			t.Fatal(err)
		}

		symlink := filepath.Join(moduleDir, "main.tf")
		targetPath, err := filepath.Rel(filepath.Dir(symlink), targetFile)
		if err != nil {
			t.Fatal(err)
		}
		// Symlink made in the mod directory, with a relative link of "target/main.tf"
		if err := os.Symlink(targetPath, symlink); err != nil {
			t.Fatal(err)
		}

		rootDirFile := filepath.Join(rootDir, "main.tf")
		if err := os.MkdirAll(filepath.Dir(rootDirFile), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(rootDirFile, []byte(cfg), 0644); err != nil {
			t.Fatal(err)
		}

		view, done := testView(t)
		c := &FmtCommand{
			Meta: Meta{
				View:       view,
				WorkingDir: workingDir,
			},
		}

		// Format "mod" directory, which contains a symlink
		if code := c.Run([]string{"-no-color", "mod"}); code != 0 {
			output := done(t)
			t.Fatalf("unexpected failure with code %d: %s", code, output.Stderr())
		}

		// Ensure that the main.tf file in the root directory is unchanged
		got, err := os.ReadFile(rootDirFile)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, []byte(cfg)) {
			t.Fatalf("unrelated file was modified:\n%s", got)
		}

		// The symlink in mod/ should still be a symlink
		linkInfo, err := os.Lstat(symlink)
		if err != nil {
			t.Fatal(err)
		}
		if linkInfo.Mode().Type() != os.ModeSymlink {
			t.Fatalf("expected %s to be a symlink", symlink)
		}

		// The symlink target in mod/target/...
		// 1. Should have permissions unchanged.
		// 2. Should be formatted.
		fi, err := os.Stat(targetFile)
		if err != nil {
			t.Fatalf("unexpected error stating target file: %s", err)
		}
		if fi.Mode().Perm() != perms {
			t.Fatalf("expected target file permissions to be %o, got: %o", perms, fi.Mode().Perm())
		}

		view, done = testView(t)
		c = &FmtCommand{
			Meta: Meta{
				View:       view,
				WorkingDir: workingDir,
			},
		}
		if code := c.Run([]string{"-no-color", "-check", "mod"}); code != 0 {
			output := done(t)
			t.Fatalf("expected `terraform fmt -check` to exit with code 0 due to the file being formatted, got code %d: stdout: %s\nstderr: %s", code, output.Stdout(), output.Stderr())
		}
	})
}

func TestFmt_checkStdin(t *testing.T) {
	t.Parallel()
	input := new(bytes.Buffer)
	input.Write(fmtFixture.input)

	view, done := testView(t)
	c := &FmtCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			View:             view,
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

	output := done(t)
	if output.Stdout() != "" {
		t.Fatalf("expected no output, got: %q", output.Stdout())
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

	err := os.WriteFile(filepath.Join(dir, fmtFixture.filename), fmtFixture.input, 0644)
	if err != nil {
		t.Fatal(err)
	}

	return dir
}
