// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package ast

import (
	"testing"

	"github.com/zclconf/go-cty/cty"
)

// TestMigrationPatterns_TypeChanges exercises realistic provider migration
// scenarios where an attribute changes type. Each test case represents a
// category of breaking change observed across real Terraform provider major
// version upgrades:
//
//   - AzureRM provider v3→v4 address_prefix (string) → address_prefixes (list)
//
// Each test has multiple files and resource instances with varying expression
// complexity: literals, variable/local references, ternary conditionals, and
// string interpolation mixing literals with resource references. All HCL
// comment styles (#, //, /* */) are used to verify comment preservation.
func TestMigrationPatterns_TypeChanges(t *testing.T) {
	tests := []struct {
		name   string
		files  map[string]string
		mutate func(t *testing.T, mod *Module)
		want   map[string]string
	}{
		// ── Category: Scalar Attribute to List ────────────────────────
		// Real: AzureRM v4 address_prefix (string) → address_prefixes ([]string)
		// Behavior: Remove scalar attr, add list attr wrapping the value
		{
			name: "scalar_attribute_to_list",
			files: map[string]string{
				"main.tf": `# Primary subnet with literal CIDR
resource "test_subnet" "example" {
  name           = "subnet-a" # the subnet name
  address_prefix = "10.0.1.0/24"
}

// Subnet referencing locals
resource "test_subnet" "from_local" {
  name           = local.subnet_name
  address_prefix = local.subnet_cidr // local ref
}

/* Reference the subnet to prove surrounding code is preserved */
output "subnet_id" {
  value = test_subnet.example.id
}
`,
				"expressions.tf": `# Subnet with conditional CIDR expression
resource "test_subnet" "conditional" {
  name           = var.subnet_name
  address_prefix = var.env == "prod" ? var.prod_cidr : var.dev_cidr // conditional
}

/* Subnet mixing string literal with resource reference */
resource "test_subnet" "interpolated" {
  name           = "${var.project}-subnet"
  address_prefix = "10.${var.octet}.1.0/24"
}

// Subnet with variable reference
resource "test_subnet" "from_var" {
  name           = var.name
  address_prefix = var.cidr_block # from variable
}
`,
				"complex.tf": `# Subnet with map index access
resource "test_subnet" "map_lookup" {
  name           = var.subnets[var.env].name
  address_prefix = var.cidr_map[var.region]
}

// Subnet with try and nested object access
resource "test_subnet" "complex_expr" {
  name           = try(var.subnet_overrides[var.env], "${var.project}-subnet")
  address_prefix = try(var.override_cidr, var.network_config.subnets.primary.cidr)
}

/* Subnet with for expression and list index */
resource "test_subnet" "for_expr" {
  name           = tostring(var.subnet_names[0])
  address_prefix = [for s in var.subnet_configs : s.cidr if s.primary][0]
}
`,
			},
			mutate: func(t *testing.T, mod *Module) {
				for _, r := range mod.FindBlocks("resource", "test_subnet") {
					r.Block.RemoveAttribute("address_prefix")
					r.Block.SetAttributeValue("address_prefixes", cty.ListVal([]cty.Value{
						cty.StringVal("10.0.1.0/24"),
					}))
				}
			},
			want: map[string]string{
				"main.tf": `# Primary subnet with literal CIDR
resource "test_subnet" "example" {
  name             = "subnet-a" # the subnet name
  address_prefixes = ["10.0.1.0/24"]
}

// Subnet referencing locals
resource "test_subnet" "from_local" {
  name             = local.subnet_name
  address_prefixes = ["10.0.1.0/24"]
}

/* Reference the subnet to prove surrounding code is preserved */
output "subnet_id" {
  value = test_subnet.example.id
}
`,
				"expressions.tf": `# Subnet with conditional CIDR expression
resource "test_subnet" "conditional" {
  name             = var.subnet_name
  address_prefixes = ["10.0.1.0/24"]
}

/* Subnet mixing string literal with resource reference */
resource "test_subnet" "interpolated" {
  name             = "${var.project}-subnet"
  address_prefixes = ["10.0.1.0/24"]
}

// Subnet with variable reference
resource "test_subnet" "from_var" {
  name             = var.name
  address_prefixes = ["10.0.1.0/24"]
}
`,
				"complex.tf": `# Subnet with map index access
resource "test_subnet" "map_lookup" {
  name             = var.subnets[var.env].name
  address_prefixes = ["10.0.1.0/24"]
}

// Subnet with try and nested object access
resource "test_subnet" "complex_expr" {
  name             = try(var.subnet_overrides[var.env], "${var.project}-subnet")
  address_prefixes = ["10.0.1.0/24"]
}

/* Subnet with for expression and list index */
resource "test_subnet" "for_expr" {
  name             = tostring(var.subnet_names[0])
  address_prefixes = ["10.0.1.0/24"]
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
