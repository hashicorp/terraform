// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestEngineApply(t *testing.T) {
	dir := t.TempDir()

	original := `resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
  acl    = "private"

  tags = {
    Name = "my-bucket"
  }
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	migrations := awsMigrations()
	results, err := Apply(dir, migrations[0])
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(results) == 0 {
		t.Fatal("expected non-empty results")
	}

	// Verify the original file on disk is unchanged (Apply doesn't write)
	onDisk, err := os.ReadFile(filepath.Join(dir, "main.tf"))
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(original, string(onDisk)); diff != "" {
		t.Fatalf("Apply should not modify files on disk (-want +got):\n%s", diff)
	}

	// Verify the first result is the acl sub-migration
	if results[0].SubMigration.Name != "s3-bucket-acl" {
		t.Fatalf("expected first sub-migration to be s3-bucket-acl, got %s", results[0].SubMigration.Name)
	}

	// Verify the result contains transformed content
	if len(results[0].Files) == 0 {
		t.Fatal("expected files in first result")
	}
	after := string(results[0].Files[0].After)
	if !strings.Contains(after, `aws_s3_bucket_acl`) {
		t.Fatalf("expected transformed content to contain aws_s3_bucket_acl, got:\n%s", after)
	}
}

func TestEngineApplyNoMatch(t *testing.T) {
	dir := t.TempDir()

	content := `resource "aws_instance" "example" {
  ami           = "ami-123456"
  instance_type = "t2.micro"
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	migrations := awsMigrations()
	results, err := Apply(dir, migrations[0])
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(results) != 0 {
		t.Fatalf("expected empty results for non-matching file, got %d results", len(results))
	}
}

func TestEngineIgnoresNonTfFiles(t *testing.T) {
	dir := t.TempDir()

	content := `# This is a markdown file with acl = "private" in it
resource "aws_s3_bucket" "example" {
  acl = "private"
}
`
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	migrations := awsMigrations()
	results, err := Apply(dir, migrations[0])
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(results) != 0 {
		t.Fatalf("expected empty results for non-.tf files, got %d results", len(results))
	}
}

func TestEngineWriteResults(t *testing.T) {
	dir := t.TempDir()

	original := `resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
  acl    = "private"
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	migrations := awsMigrations()
	results, err := Apply(dir, migrations[0])
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if err := WriteResults(dir, results); err != nil {
		t.Fatalf("unexpected error writing results: %s", err)
	}

	// Verify the file on disk has been updated
	onDisk, err := os.ReadFile(filepath.Join(dir, "main.tf"))
	if err != nil {
		t.Fatal(err)
	}

	if string(onDisk) == original {
		t.Fatal("expected file to be updated after WriteResults")
	}
	if !strings.Contains(string(onDisk), `aws_s3_bucket_acl`) {
		t.Fatalf("expected written file to contain aws_s3_bucket_acl, got:\n%s", string(onDisk))
	}
}

func TestEngineChaining(t *testing.T) {
	dir := t.TempDir()

	original := `resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
  acl    = "private"

  cors_rule {
    allowed_headers = ["*"]
    allowed_methods = ["PUT", "POST"]
    allowed_origins = ["https://example.com"]
  }
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	migrations := awsMigrations()
	results, err := Apply(dir, migrations[0])
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// We expect at least two sub-migration results: acl and cors
	if len(results) < 2 {
		t.Fatalf("expected at least 2 sub-migration results, got %d", len(results))
	}

	// Verify sub-migration names
	names := make([]string, len(results))
	for i, r := range results {
		names[i] = r.SubMigration.Name
	}

	if names[0] != "s3-bucket-acl" {
		t.Fatalf("expected first result to be s3-bucket-acl, got %s", names[0])
	}
	if names[1] != "s3-bucket-cors" {
		t.Fatalf("expected second result to be s3-bucket-cors, got %s", names[1])
	}

	// The final sub-migration result should have the output after both transforms.
	// Get the last file result for main.tf from the cors sub-migration.
	finalAfter := string(results[1].Files[0].After)
	if !strings.Contains(finalAfter, `aws_s3_bucket_acl`) {
		t.Fatalf("expected chained output to contain aws_s3_bucket_acl, got:\n%s", finalAfter)
	}
	if !strings.Contains(finalAfter, `aws_s3_bucket_cors_configuration`) {
		t.Fatalf("expected chained output to contain aws_s3_bucket_cors_configuration, got:\n%s", finalAfter)
	}

	// Verify the original bucket resource no longer contains cors_rule.
	// Find the aws_s3_bucket block and check it does not have cors_rule.
	bucketIdx := strings.Index(finalAfter, `resource "aws_s3_bucket" "example"`)
	corsIdx := strings.Index(finalAfter, `resource "aws_s3_bucket_cors_configuration"`)
	if bucketIdx < 0 || corsIdx < 0 {
		t.Fatal("expected both aws_s3_bucket and aws_s3_bucket_cors_configuration in output")
	}
	bucketBlock := finalAfter[bucketIdx:corsIdx]
	if strings.Contains(bucketBlock, "cors_rule") {
		t.Fatalf("expected cors_rule to be extracted from aws_s3_bucket block, got:\n%s", bucketBlock)
	}
}
