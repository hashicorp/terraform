// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/cli"
)

func TestStateMigrate_basic(t *testing.T) {
	ui := cli.NewMockUi()
	view, done := testView(t)
	c := &StateMigrateCommand{
		Meta: Meta{
			Ui:   ui,
			View: view,
		},
	}

	tmpDir := t.TempDir()
	_, err := os.Create(filepath.Join(tmpDir, ".terraform.lock.hcl"))
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(tmpDir)

	args := []string{}
	code := c.Run(args)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	out := done(t)

	expectedMsg := "Not implemented yet"
	if !strings.Contains(out.Stderr(), expectedMsg) {
		t.Fatalf("expected output %q, got %q", expectedMsg, out.All())
	}
}

func TestStateMigrate_nonExistentLockFiles(t *testing.T) {
	ui := cli.NewMockUi()
	view, done := testView(t)
	c := &StateMigrateCommand{
		Meta: Meta{
			Ui:   ui,
			View: view,
		},
	}

	// use temporary directory to ensure the lock files certainly do not exist
	tmpDir := t.TempDir()

	args := []string{
		"-input=false",
		"-source-provider-lock-file", filepath.Join(tmpDir, ".terraform.lock.hcl"),
		"-destination-provider-lock-file", filepath.Join(tmpDir, ".terraform.lock.hcl"),
	}
	code := c.Run(args)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	out := done(t)

	expectedMsg := "Unreadable source provider lock file"
	if !strings.Contains(out.Stderr(), expectedMsg) {
		t.Fatalf("expected output %q, got %q", expectedMsg, out.All())
	}

	expectedMsg = "Unreadable destination provider lock file"
	if !strings.Contains(out.Stderr(), expectedMsg) {
		t.Fatalf("expected output %q, got %q", expectedMsg, out.All())
	}
}
