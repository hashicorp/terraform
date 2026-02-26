// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/cli"
)

func TestInit2_versionConstraintAdded(t *testing.T) {
	// This test is for what happens when there is a version constraint added
	// to a module that previously didn't have one.
	td := t.TempDir()
	testCopyDir(t, testFixturePath(filepath.Join("dynamic-module-sources", "add-version-constraint")), td)
	t.Chdir(td)

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{"-get=false"}
	code := c.Run(args)
	testOutput := done(t)
	if code != 1 {
		t.Fatalf("got exit status %d; want 1\nstderr:\n%s\n\nstdout:\n%s", code, testOutput.Stderr(), testOutput.Stdout())
	}
	got := testOutput.All()

	want := "Module version requirements have changed"
	if !strings.Contains(got, want) {
		t.Fatalf("wrong error\ngot:\n%s\n\nwant: containing %q", got, want)
	}
}

func TestInit2_invalidRegistrySourceWithModule(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(filepath.Join("dynamic-module-sources", "invalid-registry-source-with-module")), td)
	t.Chdir(td)

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{}
	code := c.Run(args)
	testOutput := done(t)
	if code != 1 {
		t.Fatalf("got exit status %d; want 1\nstderr:\n%s\n\nstdout:\n%s", code, testOutput.Stderr(), testOutput.Stdout())
	}
	got := testOutput.All()

	want := "Invalid registry module source address"
	if !strings.Contains(got, want) {
		t.Fatalf("wrong error\ngot:\n%s\n\nwant: containing %q", got, want)
	}
}

func TestInit2_localSourceWithVersion(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(filepath.Join("dynamic-module-sources", "local-source-with-version")), td)
	t.Chdir(td)

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{}
	code := c.Run(args)
	testOutput := done(t)
	if code != 1 {
		t.Fatalf("got exit status %d; want 1\nstderr:\n%s\n\nstdout:\n%s", code, testOutput.Stderr(), testOutput.Stdout())
	}
	got := testOutput.All()

	want := "Invalid registry module source address"
	if !strings.Contains(got, want) {
		t.Fatalf("wrong error\ngot:\n%s\n\nwant: containing %q", got, want)
	}
}
