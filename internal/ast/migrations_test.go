// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package ast

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// TestMigrationPatterns exercises realistic provider migration scenarios using
// the AST library. Each test case represents a category of breaking change
// observed across real Terraform provider major version upgrades:
//
//   - AWS provider v3→v4, v4→v5
//   - Google Cloud provider v5→v6
//   - AzureRM provider v3→v4
//   - Terraform core backend→cloud migration
//
// Test cases use a fictional "test" provider so examples are self-contained.
// Where a mutation requires reading attribute values from the source config
// (which the current API does not support), the test hardcodes known values
// and documents the limitation.
func TestMigrationPatterns(t *testing.T) {
	tests := []struct {
		name   string
		files  map[string]string
		mutate func(t *testing.T, mod *Module)
		want   map[string]string
	}{
		// ── Category: Attribute Rename ───────────────────────────────
		// Real: AWS v5 name→db_name, Azure v4 enable_auto_scaling→auto_scaling_enabled
		{
			name: "rename_attribute",
			files: map[string]string{
				"main.tf": `resource "test_instance" "example" {
  ami = "abc-123"
  instance_type = "t2.micro"
}
`,
			},
			mutate: func(t *testing.T, mod *Module) {
				for _, r := range mod.FindBlocks("resource", "test_instance") {
					r.Block.RenameAttribute("ami", "image_id")
				}
			},
			want: map[string]string{
				"main.tf": `resource "test_instance" "example" {
  image_id      = "abc-123"
  instance_type = "t2.micro"
}
`,
			},
		},

		// ── Category: Batch Attribute Renames (naming convention) ────
		// Real: Azure v4 renamed ~20 enable_* → *_enabled across resources
		{
			name: "rename_multiple_attributes_convention",
			files: map[string]string{
				"main.tf": `resource "test_cluster" "example" {
  name = "my-cluster"
  enable_auto_scaling = true
  enable_host_encryption = true
  enable_node_public_ip = false
}
`,
			},
			mutate: func(t *testing.T, mod *Module) {
				renames := [][2]string{
					{"enable_auto_scaling", "auto_scaling_enabled"},
					{"enable_host_encryption", "host_encryption_enabled"},
					{"enable_node_public_ip", "node_public_ip_enabled"},
				}
				for _, r := range mod.FindBlocks("resource", "test_cluster") {
					for _, rn := range renames {
						r.Block.RenameAttribute(rn[0], rn[1])
					}
				}
			},
			want: map[string]string{
				"main.tf": `resource "test_cluster" "example" {
  name                    = "my-cluster"
  auto_scaling_enabled    = true
  host_encryption_enabled = true
  node_public_ip_enabled  = false
}
`,
			},
		},

		// ── Category: Attribute Removal ──────────────────────────────
		// Real: AWS v5 removed enable_classiclink, GCP v6 removed multi_region_auxiliary
		{
			name: "remove_deprecated_attribute",
			files: map[string]string{
				"main.tf": `resource "test_vpc" "example" {
  cidr_block = "10.0.0.0/16"
  enable_classiclink = true
  enable_dns_support = true
}
`,
			},
			mutate: func(t *testing.T, mod *Module) {
				for _, r := range mod.FindBlocks("resource", "test_vpc") {
					r.Block.RemoveAttribute("enable_classiclink")
				}
			},
			want: map[string]string{
				"main.tf": `resource "test_vpc" "example" {
  cidr_block         = "10.0.0.0/16"
  enable_dns_support = true
}
`,
			},
		},

		// ── Category: Add Required Attribute ─────────────────────────
		// Real: GCP v5/v6 added deletion_protection=true on many resources
		{
			name: "add_required_attribute",
			files: map[string]string{
				"main.tf": `resource "test_database" "example" {
  name = "my-db"
  engine = "postgres"
}
`,
			},
			mutate: func(t *testing.T, mod *Module) {
				for _, r := range mod.FindBlocks("resource", "test_database") {
					r.Block.SetAttributeValue("deletion_protection", cty.True)
				}
			},
			want: map[string]string{
				"main.tf": `resource "test_database" "example" {
  name                = "my-db"
  engine              = "postgres"
  deletion_protection = true
}
`,
			},
		},

		// ── Category: Resource Type Rename ───────────────────────────
		// Real: AWS v4 aws_s3_bucket_object→aws_s3_object,
		//       Azure v4 azurerm_sql_server→azurerm_mssql_server
		{
			name: "rename_resource_type_labels",
			files: map[string]string{
				"main.tf": `resource "test_bucket_object" "file" {
  bucket = "my-bucket"
  key = "index.html"
}
`,
			},
			mutate: func(t *testing.T, mod *Module) {
				for _, r := range mod.FindBlocks("resource", "test_bucket_object") {
					labels := r.Block.Labels()
					labels[0] = "test_object"
					r.Block.SetLabels(labels)
				}
			},
			want: map[string]string{
				"main.tf": `resource "test_object" "file" {
  bucket = "my-bucket"
  key    = "index.html"
}
`,
			},
		},

		// ── Category: Resource Type Rename + Reference Update ────────
		// When a resource type changes, all references must update too.
		// Real: AWS v4 aws_s3_bucket_object→aws_s3_object across files
		{
			name: "rename_resource_type_with_references",
			files: map[string]string{
				"main.tf": `resource "test_bucket_object" "file" {
  bucket = "my-bucket"
  key = "index.html"
}
`,
				"outputs.tf": `output "object_id" {
  value = test_bucket_object.file.id
}
`,
			},
			mutate: func(t *testing.T, mod *Module) {
				for _, r := range mod.FindBlocks("resource", "test_bucket_object") {
					labels := r.Block.Labels()
					labels[0] = "test_object"
					r.Block.SetLabels(labels)
				}
				mod.RenameReferencePrefix(
					makeTraversal("test_bucket_object"),
					makeTraversal("test_object"),
				)
			},
			want: map[string]string{
				"main.tf": `resource "test_object" "file" {
  bucket = "my-bucket"
  key    = "index.html"
}
`,
				"outputs.tf": `output "object_id" {
  value = test_object.file.id
}
`,
			},
		},

		// ── Category: Nested Block Flattening ────────────────────────
		// A nested block is dissolved and its attributes promoted to the parent.
		// Real: AWS v5 ElastiCache cluster_mode block dissolved,
		//       attrs promoted to top-level
		//
		// NOTE: A real migration needs to read attribute values from the
		// nested block before removing it. This test uses known values
		// because the current API does not expose attribute reading.
		{
			name: "flatten_nested_block_into_parent",
			files: map[string]string{
				"main.tf": `resource "test_cache_cluster" "example" {
  description = "my-cluster"
  cluster_config {
    num_shards = 3
    replicas_per_shard = 2
  }
}
`,
			},
			mutate: func(t *testing.T, mod *Module) {
				for _, r := range mod.FindBlocks("resource", "test_cache_cluster") {
					// In a real migration, read values from the nested block first.
					r.Block.SetAttributeValue("num_shards", cty.NumberIntVal(3))
					r.Block.SetAttributeValue("replicas_per_shard", cty.NumberIntVal(2))
					r.Block.RemoveBlock("cluster_config")
				}
			},
			want: map[string]string{
				"main.tf": `resource "test_cache_cluster" "example" {
  description        = "my-cluster"
  num_shards         = 3
  replicas_per_shard = 2
}
`,
			},
		},

		// ── Category: Extract Nested Block to Standalone Resource ────
		// An inline nested block is removed from a resource and replaced
		// by a new dedicated resource that references the parent.
		// Real: AWS v4 extracted 13 sub-resources from aws_s3_bucket
		//       (versioning, logging, cors_rule, lifecycle_rule, etc.)
		//
		// NOTE: Same limitation as above — values are hardcoded because
		// the current API does not expose attribute reading.
		{
			name: "extract_nested_block_to_standalone_resource",
			files: map[string]string{
				"main.tf": `resource "test_bucket" "example" {
  name = "my-bucket"
  versioning {
    enabled = true
  }
}
`,
			},
			mutate: func(t *testing.T, mod *Module) {
				for _, r := range mod.FindBlocks("resource", "test_bucket") {
					r.Block.RemoveBlock("versioning")

					nb := r.File.AddBlock("resource", []string{"test_bucket_versioning", "example"})
					nb.SetAttributeRaw("bucket", hclwrite.TokensForTraversal(hcl.Traversal{
						hcl.TraverseRoot{Name: "test_bucket"},
						hcl.TraverseAttr{Name: "example"},
						hcl.TraverseAttr{Name: "id"},
					}))
					nb.SetAttributeValue("status", cty.StringVal("Enabled"))
				}
			},
			want: map[string]string{
				"main.tf": `resource "test_bucket" "example" {
  name = "my-bucket"
}
resource "test_bucket_versioning" "example" {
  bucket = test_bucket.example.id
  status = "Enabled"
}
`,
			},
		},

		// ── Category: Block to Single Attribute ─────────────────────
		// A single-instance nested block is replaced by a scalar attribute.
		// Real: Azure v4 retention_policy {} → retention_policy_in_days,
		//       trust_policy {} → trust_policy_enabled
		{
			name: "single_block_to_attribute",
			files: map[string]string{
				"main.tf": `resource "test_registry" "example" {
  name = "my-registry"
  retention_policy {
    days = 30
  }
}
`,
			},
			mutate: func(t *testing.T, mod *Module) {
				for _, r := range mod.FindBlocks("resource", "test_registry") {
					// In a real migration, read "days" from the block first.
					r.Block.RemoveBlock("retention_policy")
					r.Block.SetAttributeValue("retention_policy_in_days", cty.NumberIntVal(30))
				}
			},
			want: map[string]string{
				"main.tf": `resource "test_registry" "example" {
  name                     = "my-registry"
  retention_policy_in_days = 30
}
`,
			},
		},

		// ── Category: Attribute to Block ─────────────────────────────
		// A boolean or scalar attribute is replaced by a nested block,
		// sometimes with different semantics.
		// Real: GCP v5 replication { automatic = true } → replication { auto {} },
		//       GCP v5 enable_binary_authorization → binary_authorization { eval_mode = "..." }
		{
			name: "attribute_to_block",
			files: map[string]string{
				"main.tf": `resource "test_secret" "example" {
  name = "my-secret"
  replication {
    automatic = true
  }
}
`,
			},
			mutate: func(t *testing.T, mod *Module) {
				for _, r := range mod.FindBlocks("resource", "test_secret") {
					repl := r.Block.BlockAtPath(cty.Path{cty.GetAttrStep{Name: "replication"}})
					if repl != nil {
						repl.RemoveAttribute("automatic")
						repl.AddBlock("auto")
					}
				}
			},
			want: map[string]string{
				"main.tf": `resource "test_secret" "example" {
  name = "my-secret"
  replication {
    auto {
    }
  }
}
`,
			},
		},

		// ── Category: Move Attribute into Nested Block ───────────────
		// A flat top-level attribute moves into a new nested block.
		// Real: GCP v6 google_alloydb_cluster network → network_config { network = ... }
		{
			name: "move_attribute_into_nested_block",
			files: map[string]string{
				"main.tf": `resource "test_cluster" "example" {
  name = "my-cluster"
  network = "projects/p/global/networks/default"
}
`,
			},
			mutate: func(t *testing.T, mod *Module) {
				for _, r := range mod.FindBlocks("resource", "test_cluster") {
					// In a real migration, read the value of "network" first.
					r.Block.RemoveAttribute("network")
					nc := r.Block.AddBlock("network_config")
					nc.SetAttributeValue("network", cty.StringVal("projects/p/global/networks/default"))
				}
			},
			want: map[string]string{
				"main.tf": `resource "test_cluster" "example" {
  name = "my-cluster"
  network_config {
    network = "projects/p/global/networks/default"
  }
}
`,
			},
		},

		// ── Category: Remote Backend to Cloud Block ──────────────────
		// Terraform core migration: backend "remote" → cloud block.
		// Real: Terraform v1.1+ migration from remote backend to HCP Terraform.
		//
		// NOTE: In a real migration, attribute values would be read from
		// the old backend block. This test hardcodes the known values.
		{
			name: "remote_backend_to_cloud_block",
			files: map[string]string{
				"main.tf": `terraform {
  backend "remote" {
    hostname = "app.terraform.io"
    organization = "my-org"
    workspaces {
      name = "my-workspace"
    }
  }
}
`,
			},
			mutate: func(t *testing.T, mod *Module) {
				for _, r := range mod.FindBlocks("terraform", "") {
					r.Block.RemoveBlock("backend")
					cloud := r.Block.AddBlock("cloud")
					cloud.SetAttributeValue("hostname", cty.StringVal("app.terraform.io"))
					cloud.SetAttributeValue("organization", cty.StringVal("my-org"))
					ws := cloud.AddBlock("workspaces")
					ws.SetAttributeValue("name", cty.StringVal("my-workspace"))
				}
			},
			want: map[string]string{
				"main.tf": `terraform {
  cloud {
    hostname     = "app.terraform.io"
    organization = "my-org"
    workspaces {
      name = "my-workspace"
    }
  }
}
`,
			},
		},

		// ── Category: Rename Attribute in Nested Block ───────────────
		// An attribute inside a nested block is renamed.
		// Real: Azure v4 sentinel_alert_rule_scheduled
		//       incident_configuration.group_by_entities → incident.by_entities
		{
			name: "rename_nested_block_attribute",
			files: map[string]string{
				"main.tf": `resource "test_alert_rule" "example" {
  name = "my-rule"
  incident_config {
    create_incident = true
    group_by_entities = ["Host"]
  }
}
`,
			},
			mutate: func(t *testing.T, mod *Module) {
				for _, r := range mod.FindBlocks("resource", "test_alert_rule") {
					ic := r.Block.BlockAtPath(cty.Path{cty.GetAttrStep{Name: "incident_config"}})
					if ic != nil {
						ic.RenameAttribute("group_by_entities", "by_entities")
					}
				}
			},
			want: map[string]string{
				"main.tf": `resource "test_alert_rule" "example" {
  name = "my-rule"
  incident_config {
    create_incident = true
    by_entities     = ["Host"]
  }
}
`,
			},
		},

		// ── Category: Repeated Blocks to List Attribute ──────────────
		// Multiple repeated blocks are collapsed into a single list attribute.
		// Real: Azure v4 container_app_job secrets {} → secret {},
		//       conceptually similar to tags blocks → tags map
		//
		// NOTE: A real migration must read values from each block.
		// This test hardcodes the replacement value.
		{
			name: "repeated_blocks_to_list_attribute",
			files: map[string]string{
				"main.tf": `resource "test_container" "example" {
  name = "my-app"
  allowed_ip {
    cidr = "10.0.0.0/8"
  }
  allowed_ip {
    cidr = "172.16.0.0/12"
  }
}
`,
			},
			mutate: func(t *testing.T, mod *Module) {
				for _, r := range mod.FindBlocks("resource", "test_container") {
					// Remove all allowed_ip blocks.
					for r.Block.RemoveBlock("allowed_ip") {
					}
					// Replace with a list attribute (values hardcoded).
					r.Block.SetAttributeValue("allowed_ips", cty.ListVal([]cty.Value{
						cty.StringVal("10.0.0.0/8"),
						cty.StringVal("172.16.0.0/12"),
					}))
				}
			},
			want: map[string]string{
				"main.tf": `resource "test_container" "example" {
  name        = "my-app"
  allowed_ips = ["10.0.0.0/8", "172.16.0.0/12"]
}
`,
			},
		},

		// ── Category: Enum/Value Change ──────────────────────────────
		// An attribute's allowed values change.
		// Real: AWS v5 kinesis_firehose destination="s3" → "extended_s3",
		//       GCP v5 skip_delete → deletion_policy="ABANDON"
		{
			name: "enum_value_change",
			files: map[string]string{
				"main.tf": `resource "test_stream" "example" {
  name = "my-stream"
  destination = "s3"
}
`,
			},
			mutate: func(t *testing.T, mod *Module) {
				for _, r := range mod.FindBlocks("resource", "test_stream") {
					r.Block.SetAttributeValue("destination", cty.StringVal("extended_s3"))
				}
			},
			want: map[string]string{
				"main.tf": `resource "test_stream" "example" {
  name        = "my-stream"
  destination = "extended_s3"
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

// TestCrossModuleMigrationPatterns tests migration scenarios that span
// module boundaries, where changes in a child module require coordinated
// updates in the calling (parent) module.
func TestCrossModuleMigrationPatterns(t *testing.T) {
	tests := []struct {
		name       string
		rootFiles  map[string]string
		childName  string
		childFiles map[string]string
		mutate     func(t *testing.T, cfg *Config)
		wantRoot   map[string]string
		wantChild  map[string]string
	}{
		// ── Category: Rename Module Input Variable ───────────────────
		// A module renames a variable. Callers must update the argument
		// name in their module block.
		// Real: Any module major version that renames an input variable
		{
			name: "rename_module_input",
			rootFiles: map[string]string{
				"main.tf": `module "network" {
  source   = "./modules/network"
  vpc_cidr = "10.0.0.0/16"
}
`,
			},
			childName: "network",
			childFiles: map[string]string{
				"main.tf": `variable "vpc_cidr" {
  type = string
}

resource "test_vpc" "main" {
  cidr_block = var.vpc_cidr
}
`,
			},
			mutate: func(t *testing.T, cfg *Config) {
				rootMod := cfg.Root.Module
				childMod := cfg.Root.Child("network").Module

				// Rename the variable in the child module.
				for _, r := range childMod.FindBlocks("variable", "vpc_cidr") {
					r.Block.SetLabels([]string{"cidr_block"})
				}
				// Update references in the child module: var.vpc_cidr → var.cidr_block
				childMod.RenameReferencePrefix(
					makeTraversal("var", "vpc_cidr"),
					makeTraversal("var", "cidr_block"),
				)
				// Update the argument name in the parent module call.
				for _, r := range rootMod.FindBlocks("module", "network") {
					r.Block.RenameAttribute("vpc_cidr", "cidr_block")
				}
			},
			wantRoot: map[string]string{
				"main.tf": `module "network" {
  source     = "./modules/network"
  cidr_block = "10.0.0.0/16"
}
`,
			},
			wantChild: map[string]string{
				"main.tf": `variable "cidr_block" {
  type = string
}

resource "test_vpc" "main" {
  cidr_block = var.cidr_block
}
`,
			},
		},

		// ── Category: Rename Module Output ───────────────────────────
		// A module renames an output. Callers must update their references
		// from module.<name>.<old_output> to module.<name>.<new_output>.
		// Real: Any module major version that renames an output
		{
			name: "rename_module_output",
			rootFiles: map[string]string{
				"main.tf": `module "network" {
  source = "./modules/network"
}

resource "test_instance" "web" {
  subnet_id = module.network.subnet_id
}
`,
			},
			childName: "network",
			childFiles: map[string]string{
				"outputs.tf": `output "subnet_id" {
  value = test_subnet.main.id
}
`,
			},
			mutate: func(t *testing.T, cfg *Config) {
				rootMod := cfg.Root.Module
				childMod := cfg.Root.Child("network").Module

				// Rename the output in the child module.
				for _, r := range childMod.FindBlocks("output", "subnet_id") {
					r.Block.SetLabels([]string{"primary_subnet_id"})
				}
				// Update references in the parent: module.network.subnet_id → module.network.primary_subnet_id
				rootMod.RenameReferencePrefix(
					makeTraversal("module", "network", "subnet_id"),
					makeTraversal("module", "network", "primary_subnet_id"),
				)
			},
			wantRoot: map[string]string{
				"main.tf": `module "network" {
  source = "./modules/network"
}

resource "test_instance" "web" {
  subnet_id = module.network.primary_subnet_id
}
`,
			},
			wantChild: map[string]string{
				"outputs.tf": `output "primary_subnet_id" {
  value = test_subnet.main.id
}
`,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Build root module.
			var rootFiles []*File
			for name, content := range tc.rootFiles {
				f, err := ParseFile([]byte(content), name, nil)
				if err != nil {
					t.Fatalf("parsing root %s: %s", name, err)
				}
				rootFiles = append(rootFiles, f)
			}
			rootMod := NewModule(rootFiles, "", true, nil)

			// Build child module.
			var childFiles []*File
			for name, content := range tc.childFiles {
				f, err := ParseFile([]byte(content), name, nil)
				if err != nil {
					t.Fatalf("parsing child %s: %s", name, err)
				}
				childFiles = append(childFiles, f)
			}
			childMod := NewModule(childFiles, tc.childName, true, nil)

			// Build config tree.
			childNode := &ModuleNode{Module: childMod, Children: map[string]*ModuleNode{}}
			rootNode := &ModuleNode{
				Module:   rootMod,
				Children: map[string]*ModuleNode{tc.childName: childNode},
			}
			childNode.Parent = rootNode
			cfg := &Config{Root: rootNode}

			tc.mutate(t, cfg)

			// Check root module output.
			rootBytes := rootMod.Bytes()
			for name, wantContent := range tc.wantRoot {
				gotContent, ok := rootBytes[name]
				if !ok {
					t.Errorf("missing root output file %s", name)
					continue
				}
				if string(gotContent) != wantContent {
					t.Errorf("root %s mismatch\n--- want ---\n%s\n--- got ---\n%s", name, wantContent, string(gotContent))
				}
			}

			// Check child module output.
			childBytes := childMod.Bytes()
			for name, wantContent := range tc.wantChild {
				gotContent, ok := childBytes[name]
				if !ok {
					t.Errorf("missing child output file %s", name)
					continue
				}
				if string(gotContent) != wantContent {
					t.Errorf("child %s mismatch\n--- want ---\n%s\n--- got ---\n%s", name, wantContent, string(gotContent))
				}
			}
		})
	}
}
