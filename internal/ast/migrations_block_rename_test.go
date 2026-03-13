// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package ast

import (
	"testing"
)

// TestMigrationPatterns_NestedBlockRename exercises realistic provider migration
// scenarios where a nested block type is renamed. Each test case represents a
// category of breaking change observed across real Terraform provider major
// version upgrades:
//
//   - AzureRM provider v3→v4 log {} → enabled_log {}, metric {} → enabled_metric {}
//     in azurerm_monitor_diagnostic_setting
//
// Each test has multiple files and resource instances with varying expression
// complexity: literals, variable/local references, ternary conditionals, and
// string interpolation mixing literals with resource references. All HCL
// comment styles (#, //, /* */) are used to verify comment preservation.
func TestMigrationPatterns_NestedBlockRename(t *testing.T) {
	tests := []struct {
		name   string
		files  map[string]string
		mutate func(t *testing.T, mod *Module)
		want   map[string]string
	}{
		// ── Category: Nested Block Type Rename ────────────────────────
		// Real: AzureRM v4 log {} → enabled_log {}, metric {} → enabled_metric {}
		// Behavior: Nested block type name changes; all content preserved
		{
			name: "rename_nested_block_type",
			files: map[string]string{
				"main.tf": `# Primary diagnostic setting with literal values
resource "test_diagnostic_setting" "example" {
  name               = "diag-primary" # the diagnostic name
  target_resource_id = "/subscriptions/abc/resourceGroups/rg/providers/Microsoft.Storage/storageAccounts/sa"

  log {
    category = "AuditEvent"
    enabled  = true // always on
  }

  # Second log category
  log {
    category = "RequestResponse"
    enabled  = false
  }

  metric {
    category = "AllMetrics"
    enabled  = true
  }
}

/* Another diagnostic setting with only log blocks */
resource "test_diagnostic_setting" "logs_only" {
  name               = "diag-logs" // logs only
  target_resource_id = "/subscriptions/abc/resourceGroups/rg/providers/Microsoft.Compute/virtualMachines/vm"

  log {
    category = "Administrative"
    enabled  = true # always audited
  }
}

output "diag_id" {
  value = test_diagnostic_setting.example.id
}
`,
				"expressions.tf": `# Diagnostic setting with variable references
resource "test_diagnostic_setting" "from_ref" {
  name               = var.diag_name
  target_resource_id = var.target_resource_id

  log {
    category = var.log_category // from variable
    enabled  = var.log_enabled
  }

  metric {
    category = var.metric_category
    enabled  = var.env == "prod" ? true : false # conditional
  }
}

/* Diagnostic setting with interpolated name and conditional */
resource "test_diagnostic_setting" "interpolated" {
  name               = "${var.project}-diag-${var.env}"
  target_resource_id = "${var.resource_id_prefix}/providers/Microsoft.Storage/storageAccounts/${var.storage_name}"

  log {
    category = var.env == "prod" ? "AuditEvent" : "RequestResponse" // env-based
    enabled  = var.audit_enabled
  }

  log {
    category = "Policy"
    enabled  = var.policy_enabled
  }

  metric {
    category = "AllMetrics"
    enabled  = var.metrics_enabled
  }
}
`,
				"complex.tf": `# Diagnostic setting with map index and nested object access
resource "test_diagnostic_setting" "map_lookup" {
  name               = var.diag_configs[var.env].name
  target_resource_id = var.resources[var.region].id

  log {
    category = var.log_categories[var.env]
    enabled  = var.diag_config.logging.enabled
  }

  metric {
    category = var.metric_config[var.tier].category
    enabled  = try(var.diag_config.metrics.enabled, true)
  }
}

// Diagnostic setting with try, for expression, and nested object access
resource "test_diagnostic_setting" "complex_expr" {
  name               = try(var.diag_name_override, "${var.project}-diag")
  target_resource_id = try(var.resource_overrides[var.region], var.default_resource_id)

  log {
    category = try(var.log_category_overrides[var.env], "AuditEvent")
    enabled  = alltrue([var.logging_enabled, var.compliance_mode])
  }

  /* Metrics block with for-derived value */
  metric {
    category = [for c in var.metric_categories : c.name if c.primary][0]
    enabled  = !var.metrics_disabled
  }
}
`,
			},
			mutate: func(t *testing.T, mod *Module) {
				for _, r := range mod.FindBlocks("resource", "test_diagnostic_setting") {
					for _, log := range r.Block.NestedBlocks("log") {
						log.SetType("enabled_log")
					}
					for _, metric := range r.Block.NestedBlocks("metric") {
						metric.SetType("enabled_metric")
					}
				}
			},
			want: map[string]string{
				"main.tf": `# Primary diagnostic setting with literal values
resource "test_diagnostic_setting" "example" {
  name               = "diag-primary" # the diagnostic name
  target_resource_id = "/subscriptions/abc/resourceGroups/rg/providers/Microsoft.Storage/storageAccounts/sa"

  enabled_log {
    category = "AuditEvent"
    enabled  = true // always on
  }

  # Second log category
  enabled_log {
    category = "RequestResponse"
    enabled  = false
  }

  enabled_metric {
    category = "AllMetrics"
    enabled  = true
  }
}

/* Another diagnostic setting with only log blocks */
resource "test_diagnostic_setting" "logs_only" {
  name               = "diag-logs" // logs only
  target_resource_id = "/subscriptions/abc/resourceGroups/rg/providers/Microsoft.Compute/virtualMachines/vm"

  enabled_log {
    category = "Administrative"
    enabled  = true # always audited
  }
}

output "diag_id" {
  value = test_diagnostic_setting.example.id
}
`,
				"expressions.tf": `# Diagnostic setting with variable references
resource "test_diagnostic_setting" "from_ref" {
  name               = var.diag_name
  target_resource_id = var.target_resource_id

  enabled_log {
    category = var.log_category // from variable
    enabled  = var.log_enabled
  }

  enabled_metric {
    category = var.metric_category
    enabled  = var.env == "prod" ? true : false # conditional
  }
}

/* Diagnostic setting with interpolated name and conditional */
resource "test_diagnostic_setting" "interpolated" {
  name               = "${var.project}-diag-${var.env}"
  target_resource_id = "${var.resource_id_prefix}/providers/Microsoft.Storage/storageAccounts/${var.storage_name}"

  enabled_log {
    category = var.env == "prod" ? "AuditEvent" : "RequestResponse" // env-based
    enabled  = var.audit_enabled
  }

  enabled_log {
    category = "Policy"
    enabled  = var.policy_enabled
  }

  enabled_metric {
    category = "AllMetrics"
    enabled  = var.metrics_enabled
  }
}
`,
				"complex.tf": `# Diagnostic setting with map index and nested object access
resource "test_diagnostic_setting" "map_lookup" {
  name               = var.diag_configs[var.env].name
  target_resource_id = var.resources[var.region].id

  enabled_log {
    category = var.log_categories[var.env]
    enabled  = var.diag_config.logging.enabled
  }

  enabled_metric {
    category = var.metric_config[var.tier].category
    enabled  = try(var.diag_config.metrics.enabled, true)
  }
}

// Diagnostic setting with try, for expression, and nested object access
resource "test_diagnostic_setting" "complex_expr" {
  name               = try(var.diag_name_override, "${var.project}-diag")
  target_resource_id = try(var.resource_overrides[var.region], var.default_resource_id)

  enabled_log {
    category = try(var.log_category_overrides[var.env], "AuditEvent")
    enabled  = alltrue([var.logging_enabled, var.compliance_mode])
  }

  /* Metrics block with for-derived value */
  enabled_metric {
    category = [for c in var.metric_categories : c.name if c.primary][0]
    enabled  = !var.metrics_disabled
  }
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
