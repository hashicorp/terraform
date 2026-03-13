// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package ast

import (
	"testing"

	"github.com/zclconf/go-cty/cty"
)

// TestMigrationPatterns_MergeAttributes exercises the merge_attributes_into_one
// migration pattern. This covers the Azure SQL provider upgrade where the
// separate edition and requested_service_objective_name attributes were
// consolidated into a single sku_name attribute (e.g. "S0", "P1", "Basic").
func TestMigrationPatterns_MergeAttributes(t *testing.T) {
	tests := []struct {
		name   string
		files  map[string]string
		mutate func(t *testing.T, mod *Module)
		want   map[string]string
	}{
		// ── Category: Merge Attributes Into One ───────────────────────
		// Real: Azure SQL edition + requested_service_objective_name → sku_name
		// Behavior: Remove multiple attributes, add one consolidated attribute
		{
			name: "merge_attributes_into_one",
			files: map[string]string{
				"main.tf": `# Primary SQL database with literal values
resource "test_sql_database" "primary" {
  name                              = "primary-db"
  server_name                       = "sql-server-01" # the target server
  edition                           = "Standard"
  requested_service_objective_name  = "S0" // service tier
}

/* Secondary database on the same server */
resource "test_sql_database" "secondary" {
  name                             = "secondary-db"
  server_name                      = "sql-server-01"
  edition                          = "Premium" # premium edition
  requested_service_objective_name = "P1"
}

output "primary_id" {
  value = test_sql_database.primary.id
}
`,
				"expressions.tf": `# SQL database with conditional edition
resource "test_sql_database" "conditional" {
  name                             = var.env == "prod" ? "prod-db" : "dev-db" // env-based name
  server_name                      = var.server_name
  edition                          = var.env == "prod" ? "Premium" : "Standard"
  requested_service_objective_name = var.env == "prod" ? "P1" : "S0"
}

/* SQL database with string interpolation */
resource "test_sql_database" "interpolated" {
  name                             = "${var.project}-db-${var.suffix}"
  server_name                      = "${var.project}-server"
  edition                          = var.db_edition
  requested_service_objective_name = var.db_objective # from variable
}
`,
				"complex.tf": `# SQL database with map index and nested object access
resource "test_sql_database" "map_lookup" {
  name                             = var.databases[var.env].name
  server_name                      = var.servers[var.env].hostname
  edition                          = var.db_config[var.env].edition
  requested_service_objective_name = var.db_config[var.env].objective
}

// SQL database with try, for expression, and nested object access
resource "test_sql_database" "complex_expr" {
  name                             = try(var.db_overrides[var.region], "${var.project}-db")
  server_name                      = coalesce(var.server_override, var.config.sql.server)
  edition                          = try(var.edition_map[var.tier], "Standard")
  requested_service_objective_name = [for o in var.objectives : o.name if o.tier == var.tier][0]
}
`,
			},
			mutate: func(t *testing.T, mod *Module) {
				for _, r := range mod.FindBlocks("resource", "test_sql_database") {
					r.Block.RemoveAttribute("edition")
					r.Block.RemoveAttribute("requested_service_objective_name")
					r.Block.SetAttributeValue("sku_name", cty.StringVal("S0"))
				}
			},
			want: map[string]string{
				"main.tf": `# Primary SQL database with literal values
resource "test_sql_database" "primary" {
  name        = "primary-db"
  server_name = "sql-server-01" # the target server
  sku_name    = "S0"
}

/* Secondary database on the same server */
resource "test_sql_database" "secondary" {
  name        = "secondary-db"
  server_name = "sql-server-01"
  sku_name    = "S0"
}

output "primary_id" {
  value = test_sql_database.primary.id
}
`,
				"expressions.tf": `# SQL database with conditional edition
resource "test_sql_database" "conditional" {
  name        = var.env == "prod" ? "prod-db" : "dev-db" // env-based name
  server_name = var.server_name
  sku_name    = "S0"
}

/* SQL database with string interpolation */
resource "test_sql_database" "interpolated" {
  name        = "${var.project}-db-${var.suffix}"
  server_name = "${var.project}-server"
  sku_name    = "S0"
}
`,
				"complex.tf": `# SQL database with map index and nested object access
resource "test_sql_database" "map_lookup" {
  name        = var.databases[var.env].name
  server_name = var.servers[var.env].hostname
  sku_name    = "S0"
}

// SQL database with try, for expression, and nested object access
resource "test_sql_database" "complex_expr" {
  name        = try(var.db_overrides[var.region], "${var.project}-db")
  server_name = coalesce(var.server_override, var.config.sql.server)
  sku_name    = "S0"
}
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
