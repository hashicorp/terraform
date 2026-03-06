// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrateList(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.tf"), []byte(`resource "aws_s3_bucket" "test" {
  bucket = "my-bucket"
  acl    = "private"
}
`), 0644)

	t.Chdir(dir)

	view, done := testView(t)
	c := &MigrateListCommand{
		Meta: Meta{View: view},
	}

	code := c.Run(nil)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}

	output := done(t)
	got := output.Stdout()
	if !strings.Contains(got, "hashicorp/aws") {
		t.Errorf("expected hashicorp/aws in output:\n%s", got)
	}
}

func TestMigrateList_noMatches(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.tf"), []byte(`resource "null_resource" "test" {
}
`), 0644)

	t.Chdir(dir)

	view, done := testView(t)
	c := &MigrateListCommand{
		Meta: Meta{View: view},
	}

	code := c.Run(nil)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}

	output := done(t)
	got := output.Stdout()
	if !strings.Contains(got, "No applicable migrations") {
		t.Errorf("expected 'No applicable migrations' in output:\n%s", got)
	}
}
