// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"testing"
)

func TestModuleSourceRemote_VersionHint(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected string
	}{
		// Git URLs with version parameters
		{
			name:     "git URL with ref parameter",
			source:   "git::https://example.com/repo.git?ref=v1.2.3",
			expected: "v1.2.3",
		},
		{
			name:     "git URL with tag parameter (not supported)",
			source:   "git::https://example.com/repo.git?tag=v2.0.0",
			expected: "",
		},
		{
			name:     "git URL with branch parameter (not supported)",
			source:   "git::https://example.com/repo.git?branch=main",
			expected: "",
		},
		{
			name:     "git URL with commit parameter (not supported)",
			source:   "git::https://example.com/repo.git?commit=abc123def456",
			expected: "",
		},
		{
			name:     "GitHub shorthand with ref",
			source:   "github.com/hashicorp/terraform-aws-vpc?ref=v3.0.0",
			expected: "v3.0.0",
		},
		{
			name:     "BitBucket shorthand with ref",
			source:   "bitbucket.org/hashicorp/terraform-consul?ref=v0.1.0",
			expected: "v0.1.0",
		},
		{
			name:     "SSH git URL with ref",
			source:   "git@github.com:hashicorp/terraform.git?ref=v1.0.0",
			expected: "v1.0.0",
		},
		{
			name:     "URL ending in .git with ref",
			source:   "https://gitlab.com/example/repo.git?ref=feature-branch",
			expected: "feature-branch",
		},
		{
			name:     "Multiple parameters - ref takes precedence",
			source:   "git::https://example.com/repo.git?depth=1&ref=v1.0.0&other=value",
			expected: "v1.0.0",
		},
		{
			name:     "only ref parameter is recognized when multiple version params present",
			source:   "git::https://example.com/repo.git?branch=main&tag=v1.0.0&commit=abc123&ref=v2.0.0",
			expected: "v2.0.0",
		},
		{
			name:     "non-ref version parameters are ignored",
			source:   "git::https://example.com/repo.git?branch=main&commit=abc123",
			expected: "",
		},

		// Non-git URLs should return empty
		{
			name:     "HTTP archive URL",
			source:   "https://example.com/module.zip?version=1.0.0",
			expected: "",
		},
		{
			name:     "S3 URL",
			source:   "s3::https://s3.amazonaws.com/bucket/module.zip?version=1.0.0",
			expected: "",
		},
		{
			name:     "GCS URL",
			source:   "gcs::https://storage.googleapis.com/bucket/module.zip?version=1.0.0",
			expected: "",
		},

		// Git URLs without version parameters
		{
			name:     "git URL without parameters",
			source:   "git::https://example.com/repo.git",
			expected: "",
		},
		{
			name:     "git URL with non-version parameters only",
			source:   "git::https://example.com/repo.git?depth=1&sshkey=mykey",
			expected: "",
		},
		{
			name:     "GitHub shorthand without parameters",
			source:   "github.com/hashicorp/terraform-aws-vpc",
			expected: "",
		},

		// Edge cases
		{
			name:     "invalid URL",
			source:   "not-a-valid-url",
			expected: "",
		},
		{
			name:     "empty source",
			source:   "",
			expected: "",
		},

		// Additional edge cases
		{
			name:     "git URL with multiple query parameters",
			source:   "git::https://example.com/repo.git?depth=1&ref=v1.0.0&sshkey=mykey",
			expected: "v1.0.0",
		},
		{
			name:     "git URL with fragment and query",
			source:   "git::https://example.com/repo.git?ref=v2.0.0#somefragment",
			expected: "v2.0.0", // Fragment should be excluded from version hint
		},
		{
			name:     "case sensitivity - uppercase REF",
			source:   "git::https://example.com/repo.git?REF=v1.0.0",
			expected: "", // Should not match due to case sensitivity
		},
		{
			name:     "malformed git URL with version parameters",
			source:   "git::not-a-valid-url?ref=v1.0.0",
			expected: "v1.0.0", // Should still extract parameter even from malformed URL
		},
		{
			name:     "git URL with empty ref parameter value",
			source:   "git::https://example.com/repo.git?ref=&tag=v1.0.0",
			expected: "", // Empty ref returns empty, tag parameter is ignored
		},
		{
			name:     "git URL with URL-encoded version",
			source:   "git::https://example.com/repo.git?ref=v1.0.0%2Bbuild",
			expected: "v1.0.0+build", // Should be URL decoded
		},
		{
			name:     "SSH git URL with complex parameters",
			source:   "git@github.com:owner/repo.git?ref=feature/new-feature&depth=1",
			expected: "feature/new-feature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := ModuleSourceRemote{
				Package: ModulePackage(tt.source),
			}
			got := source.VersionHint()
			if got != tt.expected {
				t.Errorf("VersionHint() = %q, expected %q", got, tt.expected)
			}
		})
	}
}

func TestModuleSourceRemote_isGitURL(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected bool
	}{
		// Git URLs - should return true
		{
			name:     "explicit git:: prefix",
			source:   "git::https://example.com/repo.git",
			expected: true,
		},
		{
			name:     "GitHub shorthand",
			source:   "github.com/hashicorp/terraform",
			expected: true,
		},
		{
			name:     "BitBucket shorthand",
			source:   "bitbucket.org/hashicorp/terraform",
			expected: true,
		},
		{
			name:     "SSH git URL",
			source:   "git@github.com:hashicorp/terraform.git",
			expected: true,
		},
		{
			name:     "HTTPS URL ending in .git",
			source:   "https://gitlab.com/example/repo.git",
			expected: true,
		},
		{
			name:     "HTTP URL ending in .git",
			source:   "http://example.com/repo.git",
			expected: true,
		},

		// Non-git URLs - should return false
		{
			name:     "HTTPS archive",
			source:   "https://example.com/module.zip",
			expected: false,
		},
		{
			name:     "S3 URL",
			source:   "s3::https://s3.amazonaws.com/bucket/module.zip",
			expected: false,
		},
		{
			name:     "GCS URL",
			source:   "gcs::https://storage.googleapis.com/bucket/module.zip",
			expected: false,
		},
		{
			name:     "file:// URL",
			source:   "file:///path/to/module",
			expected: false,
		},
		{
			name:     "URL containing .git but not ending with it",
			source:   "https://example.com/.git/module.zip",
			expected: false,
		},
		{
			name:     "invalid URL",
			source:   "not-a-url",
			expected: false,
		},
		{
			name:     "empty source",
			source:   "",
			expected: false,
		},

		// Additional edge cases for isGitURL
		{
			name:     "GitHub with subdirectory",
			source:   "github.com/hashicorp/terraform//modules/vpc",
			expected: true,
		},
		{
			name:     "git URL with .git in middle of path",
			source:   "https://example.com/.git-something/repo",
			expected: false,
		},
		{
			name:     "SSH with non-standard port",
			source:   "git@example.com:2222/repo.git",
			expected: true,
		},
		{
			name:     "git URL with query parameters",
			source:   "git::https://example.com/repo.git?ref=main",
			expected: true,
		},
		{
			name:     "URL ending with .git.zip (not git)",
			source:   "https://example.com/archive.git.zip",
			expected: false,
		},
		{
			name:     "BitBucket with subdirectory",
			source:   "bitbucket.org/owner/repo//modules/test",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := ModuleSourceRemote{
				Package: ModulePackage(tt.source),
			}
			got := source.isGitURL()
			if got != tt.expected {
				t.Errorf("isGitURL() = %v, expected %v", got, tt.expected)
			}
		})
	}
}
