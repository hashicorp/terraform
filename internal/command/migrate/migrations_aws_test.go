// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package migrate

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func findSubMigration(t *testing.T, migrations []Migration, migrationID, subName string) SubMigration {
	t.Helper()
	for _, m := range migrations {
		if m.ID() != migrationID {
			continue
		}
		for _, s := range m.SubMigrations {
			if s.Name == subName {
				return s
			}
		}
	}
	t.Fatalf("sub-migration %q not found in migration %q", subName, migrationID)
	return SubMigration{}
}

func TestAWSS3BucketACL(t *testing.T) {
	sub := findSubMigration(t, awsMigrations(), "hashicorp/aws/v3-to-v4", "s3-bucket-acl")

	tests := map[string]struct {
		input    string
		expected string
	}{
		"extracts acl into separate resource": {
			input: `resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
  acl    = "private"

  tags = {
    Name = "my-bucket"
  }
}
`,
			expected: `resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"

  tags = {
    Name = "my-bucket"
  }
}

resource "aws_s3_bucket_acl" "example" {
  bucket = aws_s3_bucket.example.id
  acl    = "private"
}
`,
		},
		"no match leaves input unchanged": {
			input: `resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
}
`,
			expected: `resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
}
`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := sub.Apply("main.tf", []byte(tc.input))
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if diff := cmp.Diff(tc.expected, string(got)); diff != "" {
				t.Fatalf("unexpected result (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAWSS3BucketCORS(t *testing.T) {
	sub := findSubMigration(t, awsMigrations(), "hashicorp/aws/v3-to-v4", "s3-bucket-cors")

	tests := map[string]struct {
		input    string
		expected string
	}{
		"extracts cors_rule into separate resource": {
			input: `resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"

  cors_rule {
    allowed_headers = ["*"]
    allowed_methods = ["PUT", "POST"]
    allowed_origins = ["https://example.com"]
  }
}
`,
			expected: `resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
}

resource "aws_s3_bucket_cors_configuration" "example" {
  bucket = aws_s3_bucket.example.id

  cors_rule {
    allowed_headers = ["*"]
    allowed_methods = ["PUT", "POST"]
    allowed_origins = ["https://example.com"]
  }
}
`,
		},
		"no match leaves input unchanged": {
			input: `resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
}
`,
			expected: `resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
}
`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := sub.Apply("main.tf", []byte(tc.input))
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if diff := cmp.Diff(tc.expected, string(got)); diff != "" {
				t.Fatalf("unexpected result (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAWSS3BucketLogging(t *testing.T) {
	sub := findSubMigration(t, awsMigrations(), "hashicorp/aws/v3-to-v4", "s3-bucket-logging")

	tests := map[string]struct {
		input    string
		expected string
	}{
		"extracts logging into separate resource": {
			input: `resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"

  logging {
    target_bucket = "log-bucket"
    target_prefix = "log/"
  }
}
`,
			expected: `resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
}

resource "aws_s3_bucket_logging" "example" {
  bucket = aws_s3_bucket.example.id

  logging {
    target_bucket = "log-bucket"
    target_prefix = "log/"
  }
}
`,
		},
		"no match leaves input unchanged": {
			input: `resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
}
`,
			expected: `resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
}
`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := sub.Apply("main.tf", []byte(tc.input))
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if diff := cmp.Diff(tc.expected, string(got)); diff != "" {
				t.Fatalf("unexpected result (-want +got):\n%s", diff)
			}
		})
	}
}
