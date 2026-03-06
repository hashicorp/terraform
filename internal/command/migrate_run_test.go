// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrateRun_default(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.tf"), []byte(`resource "aws_s3_bucket" "test" {
  bucket = "my-bucket"
  acl    = "private"
}
`), 0644)
	t.Chdir(dir)

	view, done := testView(t)
	c := &MigrateRunCommand{
		Meta: Meta{View: view},
	}

	code := c.Run([]string{"hashicorp/aws/v3-to-v4"})
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}

	output := done(t)
	got := output.Stdout()
	if !strings.Contains(got, "Applying") {
		t.Errorf("expected Applying header:\n%s", got)
	}

	// Verify file was modified
	content, err := os.ReadFile(filepath.Join(dir, "main.tf"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "aws_s3_bucket_acl") {
		t.Error("expected file to contain aws_s3_bucket_acl")
	}
}

func TestMigrateRun_dryRun(t *testing.T) {
	dir := t.TempDir()
	input := `resource "aws_s3_bucket" "test" {
  bucket = "my-bucket"
  acl    = "private"
}
`
	os.WriteFile(filepath.Join(dir, "main.tf"), []byte(input), 0644)
	t.Chdir(dir)

	view, done := testView(t)
	c := &MigrateRunCommand{
		Meta: Meta{View: view},
	}

	code := c.Run([]string{"-dry-run", "hashicorp/aws/v3-to-v4"})
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}

	output := done(t)
	got := output.Stdout()
	if !strings.Contains(got, "Planning") {
		t.Errorf("expected Planning header:\n%s", got)
	}
	if !strings.Contains(got, "would be applied") {
		t.Errorf("expected dry-run summary:\n%s", got)
	}

	// Verify file was NOT modified
	content, err := os.ReadFile(filepath.Join(dir, "main.tf"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != input {
		t.Error("dry-run should not modify files")
	}
}

func TestMigrateRun_notFound(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	view, done := testView(t)
	c := &MigrateRunCommand{
		Meta: Meta{View: view},
	}

	code := c.Run([]string{"nonexistent/provider/migration"})
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}

	output := done(t)
	got := output.Stderr()
	if !strings.Contains(got, "Migration not found") {
		t.Errorf("expected 'Migration not found' in error output:\n%s", got)
	}
}

func TestMigrateRun_noChanges(t *testing.T) {
	dir := t.TempDir()
	// A file with no applicable changes for the AWS migration
	os.WriteFile(filepath.Join(dir, "main.tf"), []byte(`resource "null_resource" "test" {
}
`), 0644)
	t.Chdir(dir)

	view, done := testView(t)
	c := &MigrateRunCommand{
		Meta: Meta{View: view},
	}

	code := c.Run([]string{"hashicorp/aws/v3-to-v4"})
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}

	output := done(t)
	got := output.Stdout()
	if !strings.Contains(got, "Applied 0 changes") {
		t.Errorf("expected zero changes summary:\n%s", got)
	}
}
