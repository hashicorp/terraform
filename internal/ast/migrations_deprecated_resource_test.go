// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package ast

import (
	"fmt"
	"testing"
)

// TestMigrationPatterns_DeprecatedResource exercises the deprecated_resource_add_fixme
// migration category: resources that cannot be auto-migrated are left in place and
// a FIXME comment is appended to the file to signal that manual intervention is required.
//
// Real-world examples:
//   - GCP Cloud Source Repos deprecated → Secure Source Manager (no automated path)
//   - AWS EC2-Classic resources removed in favour of VPC-only equivalents
func TestMigrationPatterns_DeprecatedResource(t *testing.T) {
	tests := []struct {
		name   string
		files  map[string]string
		mutate func(t *testing.T, mod *Module)
		want   map[string]string
	}{
		// ── Category: Deprecated Resource Add FIXME ───────────────────
		// Real: GCP Cloud Source Repos deprecated → Secure Source Manager (manual),
		//       AWS EC2-Classic resources removed
		// Behavior: resource is left unchanged; a # FIXME: comment is appended to
		// the end of each file that contains at least one matching resource block.
		{
			name: "deprecated_resource_add_fixme",
			files: map[string]string{
				"main.tf": `# Source repository with literal project and name
resource "test_source_repo" "example" {
  project = "my-project" # the GCP project
  name    = "my-repo"
}

// Second repo in the same project
resource "test_source_repo" "from_local" {
  project = "my-project"
  name    = local.secondary_repo_name
}

/* Reference the repos to prove surrounding code is preserved */
output "repo_id" {
  value = test_source_repo.example.id
}
`,
				"expressions.tf": `# Repo with conditional project
resource "test_source_repo" "conditional" {
  project = var.env == "prod" ? var.prod_project : var.dev_project // env-based
  name    = "shared-repo"
}

/* Repo with string interpolation mixing literal and resource reference */
resource "test_source_repo" "interpolated" {
  project = "${var.org}-${var.env}"
  name    = "${test_random_id.suffix.hex}-repo"
}

// Repo driven entirely by variable references
resource "test_source_repo" "from_ref" {
  project = var.project
  name    = var.repo_name
}
`,
				"complex.tf": `# Repo with map index and nested object access
resource "test_source_repo" "map_lookup" {
  project = var.projects[var.env]
  name    = var.repo_config[var.env].name
}

// Repo with try, for expression, and nested object access
resource "test_source_repo" "complex_expr" {
  project = try(var.project_overrides[var.region], var.default_project)
  name    = coalesce(var.repo_name_override, var.config.source.repo_name)
  labels  = { for k, v in var.base_labels : k => v if v != "" }
}
`,
			},
			mutate: func(t *testing.T, mod *Module) {
				for _, r := range mod.FindBlocks("resource", "test_source_repo") {
					labels := r.Block.Labels()
					r.File.AppendComment(fmt.Sprintf("FIXME: %s.%s is deprecated. Migrate to test_secure_source manually.", labels[0], labels[1]))
				}
			},
			want: map[string]string{
				"main.tf": `# Source repository with literal project and name
resource "test_source_repo" "example" {
  project = "my-project" # the GCP project
  name    = "my-repo"
}

// Second repo in the same project
resource "test_source_repo" "from_local" {
  project = "my-project"
  name    = local.secondary_repo_name
}

/* Reference the repos to prove surrounding code is preserved */
output "repo_id" {
  value = test_source_repo.example.id
}

# FIXME: test_source_repo.example is deprecated. Migrate to test_secure_source manually.

# FIXME: test_source_repo.from_local is deprecated. Migrate to test_secure_source manually.
`,
				"expressions.tf": `# Repo with conditional project
resource "test_source_repo" "conditional" {
  project = var.env == "prod" ? var.prod_project : var.dev_project // env-based
  name    = "shared-repo"
}

/* Repo with string interpolation mixing literal and resource reference */
resource "test_source_repo" "interpolated" {
  project = "${var.org}-${var.env}"
  name    = "${test_random_id.suffix.hex}-repo"
}

// Repo driven entirely by variable references
resource "test_source_repo" "from_ref" {
  project = var.project
  name    = var.repo_name
}

# FIXME: test_source_repo.conditional is deprecated. Migrate to test_secure_source manually.

# FIXME: test_source_repo.interpolated is deprecated. Migrate to test_secure_source manually.

# FIXME: test_source_repo.from_ref is deprecated. Migrate to test_secure_source manually.
`,
				"complex.tf": `# Repo with map index and nested object access
resource "test_source_repo" "map_lookup" {
  project = var.projects[var.env]
  name    = var.repo_config[var.env].name
}

// Repo with try, for expression, and nested object access
resource "test_source_repo" "complex_expr" {
  project = try(var.project_overrides[var.region], var.default_project)
  name    = coalesce(var.repo_name_override, var.config.source.repo_name)
  labels  = { for k, v in var.base_labels : k => v if v != "" }
}

# FIXME: test_source_repo.map_lookup is deprecated. Migrate to test_secure_source manually.

# FIXME: test_source_repo.complex_expr is deprecated. Migrate to test_secure_source manually.
`,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var files []*File
			for name, content := range tc.files {
				f, err := ParseFile([]byte(content), name, nil)
				if err != nil {
					t.Fatalf("parsing %s: %s", name, err)
				}
				files = append(files, f)
			}
			mod := NewModule(files, "", true, nil)

			tc.mutate(t, mod)

			got := mod.Bytes()
			for name, wantContent := range tc.want {
				gotContent, ok := got[name]
				if !ok {
					t.Errorf("missing output file %s", name)
					continue
				}
				if string(gotContent) != wantContent {
					t.Errorf("file %s mismatch\n--- want ---\n%s\n--- got ---\n%s", name, wantContent, string(gotContent))
				}
			}
		})
	}
}
