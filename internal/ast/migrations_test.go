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
// Each test has multiple files and resource instances with varying expression
// complexity: literals, variable/local references, ternary conditionals, and
// string interpolation mixing literals with resource references. All HCL
// comment styles (#, //, /* */) are used to verify comment preservation.
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
				"main.tf": `# Primary instance with literal values
resource "test_instance" "example" {
  ami           = "abc-123" # the base image
  instance_type = "t2.micro"
}

// Instance referencing a local
resource "test_instance" "from_ref" {
  ami           = local.base_ami
  instance_type = var.instance_type
}

/* Reference the instance to prove surrounding code is preserved */
output "instance_id" {
  value = test_instance.example.id
}
`,
				"expressions.tf": `# Instance with conditional expression value
resource "test_instance" "conditional" {
  ami           = var.env == "prod" ? var.prod_ami : var.dev_ami // conditional
  instance_type = "t2.micro"
}

/* Instance mixing string literal with resource reference */
resource "test_instance" "interpolated" {
  ami           = "${test_base_ami.default.id}-custom"
  instance_type = "t2.micro"
}
`,
				"complex.tf": `# Instance with map index and nested object access
resource "test_instance" "map_lookup" {
  ami           = var.ami_map[var.region]
  instance_type = var.sizing[var.env].type
}

// Instance with try, for expression, and list value
resource "test_instance" "complex_expr" {
  ami             = try(var.custom_images[var.region], var.default_ami)
  instance_type   = coalesce(var.override_type, var.config.compute.type)
  security_groups = [for sg in var.security_groups : sg.id]
}
`,
			},
			mutate: func(t *testing.T, mod *Module) {
				for _, r := range mod.FindBlocks("resource", "test_instance") {
					r.Block.RenameAttribute("ami", "image_id")
				}
			},
			want: map[string]string{
				"main.tf": `# Primary instance with literal values
resource "test_instance" "example" {
  image_id      = "abc-123" # the base image
  instance_type = "t2.micro"
}

// Instance referencing a local
resource "test_instance" "from_ref" {
  image_id      = local.base_ami
  instance_type = var.instance_type
}

/* Reference the instance to prove surrounding code is preserved */
output "instance_id" {
  value = test_instance.example.id
}
`,
				"expressions.tf": `# Instance with conditional expression value
resource "test_instance" "conditional" {
  image_id      = var.env == "prod" ? var.prod_ami : var.dev_ami // conditional
  instance_type = "t2.micro"
}

/* Instance mixing string literal with resource reference */
resource "test_instance" "interpolated" {
  image_id      = "${test_base_ami.default.id}-custom"
  instance_type = "t2.micro"
}
`,
				"complex.tf": `# Instance with map index and nested object access
resource "test_instance" "map_lookup" {
  image_id      = var.ami_map[var.region]
  instance_type = var.sizing[var.env].type
}

// Instance with try, for expression, and list value
resource "test_instance" "complex_expr" {
  image_id        = try(var.custom_images[var.region], var.default_ami)
  instance_type   = coalesce(var.override_type, var.config.compute.type)
  security_groups = [for sg in var.security_groups : sg.id]
}
`,
			},
		},

		// ── Category: Batch Attribute Renames (naming convention) ────
		// Real: Azure v4 renamed ~20 enable_* → *_enabled across resources
		{
			name: "rename_multiple_attributes_convention",
			files: map[string]string{
				"main.tf": `# AKS cluster with literal flags
resource "test_cluster" "primary" {
  name = "primary-cluster"

  # Feature flags (old naming convention)
  enable_auto_scaling    = true
  enable_host_encryption = true // encrypt at rest
  enable_node_public_ip  = false
}

output "cluster_name" {
  value = test_cluster.primary.name # track the cluster
}
`,
				"expressions.tf": `/* Cluster using variable-driven flags */
resource "test_cluster" "from_ref" {
  name                   = var.cluster_name
  enable_auto_scaling    = var.auto_scaling
  enable_host_encryption = var.host_encryption
  enable_node_public_ip  = var.public_ip
}

# Cluster with conditional flags
resource "test_cluster" "conditional" {
  name                   = "cond-cluster"
  enable_auto_scaling    = var.env == "prod" ? true : false // prod only
  enable_host_encryption = var.env == "prod" ? true : false
  enable_node_public_ip  = var.env != "prod" ? true : false
}

/* Cluster with function-call expressions */
resource "test_cluster" "with_func" {
  name                   = "${var.project}-cluster"
  enable_auto_scaling    = lookup(var.features, "auto_scaling", false)
  enable_host_encryption = lookup(var.features, "host_encryption", false)
  enable_node_public_ip  = lookup(var.features, "public_ip", false) # feature map
}
`,
				"complex.tf": `// Cluster with nested object and map index access
resource "test_cluster" "nested_access" {
  name                   = var.clusters["primary"].name
  enable_auto_scaling    = var.cluster_config.features.auto_scaling
  enable_host_encryption = var.cluster_config.features["host_encryption"]
  enable_node_public_ip  = try(var.settings.network.public_ip, false)
}

# Cluster with for expression and type constructor
resource "test_cluster" "with_for" {
  name                   = tostring(var.cluster_names[0])
  enable_auto_scaling    = contains(var.enabled_features, "auto_scaling")
  enable_host_encryption = anytrue([var.encrypt_hosts, var.compliance_mode])
  enable_node_public_ip  = !var.private_cluster
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
				"main.tf": `# AKS cluster with literal flags
resource "test_cluster" "primary" {
  name = "primary-cluster"

  # Feature flags (old naming convention)
  auto_scaling_enabled    = true
  host_encryption_enabled = true // encrypt at rest
  node_public_ip_enabled  = false
}

output "cluster_name" {
  value = test_cluster.primary.name # track the cluster
}
`,
				"expressions.tf": `/* Cluster using variable-driven flags */
resource "test_cluster" "from_ref" {
  name                    = var.cluster_name
  auto_scaling_enabled    = var.auto_scaling
  host_encryption_enabled = var.host_encryption
  node_public_ip_enabled  = var.public_ip
}

# Cluster with conditional flags
resource "test_cluster" "conditional" {
  name                    = "cond-cluster"
  auto_scaling_enabled    = var.env == "prod" ? true : false // prod only
  host_encryption_enabled = var.env == "prod" ? true : false
  node_public_ip_enabled  = var.env != "prod" ? true : false
}

/* Cluster with function-call expressions */
resource "test_cluster" "with_func" {
  name                    = "${var.project}-cluster"
  auto_scaling_enabled    = lookup(var.features, "auto_scaling", false)
  host_encryption_enabled = lookup(var.features, "host_encryption", false)
  node_public_ip_enabled  = lookup(var.features, "public_ip", false) # feature map
}
`,
				"complex.tf": `// Cluster with nested object and map index access
resource "test_cluster" "nested_access" {
  name                    = var.clusters["primary"].name
  auto_scaling_enabled    = var.cluster_config.features.auto_scaling
  host_encryption_enabled = var.cluster_config.features["host_encryption"]
  node_public_ip_enabled  = try(var.settings.network.public_ip, false)
}

# Cluster with for expression and type constructor
resource "test_cluster" "with_for" {
  name                    = tostring(var.cluster_names[0])
  auto_scaling_enabled    = contains(var.enabled_features, "auto_scaling")
  host_encryption_enabled = anytrue([var.encrypt_hosts, var.compliance_mode])
  node_public_ip_enabled  = !var.private_cluster
}
`,
			},
		},

		// ── Category: Attribute Removal ──────────────────────────────
		// Real: AWS v5 removed enable_classiclink, GCP v6 removed multi_region_auxiliary
		{
			name: "remove_deprecated_attribute",
			files: map[string]string{
				"main.tf": `# VPC with deprecated attribute
resource "test_vpc" "example" {
  cidr_block = "10.0.0.0/16"

  enable_classiclink = true # DEPRECATED: removed in v5
  enable_dns_support = true
}

// downstream reference
output "vpc_id" {
  value = test_vpc.example.id
}
`,
				"expressions.tf": `/* VPC with variable reference for deprecated attr */
resource "test_vpc" "from_ref" {
  cidr_block         = var.vpc_cidr
  enable_classiclink = var.classiclink # will be removed
  enable_dns_support = var.dns_support
}

# VPC with conditional deprecated attribute
resource "test_vpc" "conditional" {
  cidr_block         = var.vpc_cidr
  enable_classiclink = var.legacy_mode ? true : false // conditional
  enable_dns_support = true
}

/* VPC with function call for deprecated attribute */
resource "test_vpc" "with_func" {
  cidr_block         = "${var.cidr_prefix}.0.0/16"
  enable_classiclink = tobool(var.classiclink_str)
  enable_dns_support = var.dns_support
}
`,
				"complex.tf": `// VPC with nested object access
resource "test_vpc" "nested" {
  cidr_block         = cidrsubnet(var.base_cidr, 8, 0)
  enable_classiclink = var.legacy_settings.vpc.classiclink
  enable_dns_support = try(var.dns_settings.support, true)
}

# VPC with list index and contains
resource "test_vpc" "indexed" {
  cidr_block         = var.cidrs[var.region][0]
  enable_classiclink = contains(keys(var.legacy_features), "classiclink")
  enable_dns_support = alltrue([var.dns_enabled, var.vpc_dns])
}
`,
			},
			mutate: func(t *testing.T, mod *Module) {
				for _, r := range mod.FindBlocks("resource", "test_vpc") {
					r.Block.RemoveAttribute("enable_classiclink")
				}
			},
			want: map[string]string{
				"main.tf": `# VPC with deprecated attribute
resource "test_vpc" "example" {
  cidr_block = "10.0.0.0/16"

  enable_dns_support = true
}

// downstream reference
output "vpc_id" {
  value = test_vpc.example.id
}
`,
				"expressions.tf": `/* VPC with variable reference for deprecated attr */
resource "test_vpc" "from_ref" {
  cidr_block         = var.vpc_cidr
  enable_dns_support = var.dns_support
}

# VPC with conditional deprecated attribute
resource "test_vpc" "conditional" {
  cidr_block         = var.vpc_cidr
  enable_dns_support = true
}

/* VPC with function call for deprecated attribute */
resource "test_vpc" "with_func" {
  cidr_block         = "${var.cidr_prefix}.0.0/16"
  enable_dns_support = var.dns_support
}
`,
				"complex.tf": `// VPC with nested object access
resource "test_vpc" "nested" {
  cidr_block         = cidrsubnet(var.base_cidr, 8, 0)
  enable_dns_support = try(var.dns_settings.support, true)
}

# VPC with list index and contains
resource "test_vpc" "indexed" {
  cidr_block         = var.cidrs[var.region][0]
  enable_dns_support = alltrue([var.dns_enabled, var.vpc_dns])
}
`,
			},
		},

		// ── Category: Add Required Attribute ─────────────────────────
		// Real: GCP v5/v6 added deletion_protection=true on many resources
		{
			name: "add_required_attribute",
			files: map[string]string{
				"main.tf": `# Database resource missing the new required attribute
resource "test_database" "example" {
  name   = "my-db"
  engine = "postgres" // the DB engine
}

/* Reference the database */
output "db_endpoint" {
  value = test_database.example.endpoint
}
`,
				"expressions.tf": `// Database with variable references
resource "test_database" "from_ref" {
  name   = var.db_name
  engine = var.db_engine
}

# Database with conditional name
resource "test_database" "conditional" {
  name   = var.env == "prod" ? "prod-db" : "dev-db" // env-based
  engine = "postgres"
}

/* Database with interpolated name mixing literal and resource ref */
resource "test_database" "interpolated" {
  name   = "${var.project}-db-${test_random_id.db.hex}"
  engine = "postgres"
}
`,
				"complex.tf": `# Database with map index and nested object access
resource "test_database" "complex" {
  name   = var.databases["primary"].name
  engine = var.config.db.engine
}

// Database with for expression and try
resource "test_database" "for_expr" {
  name   = [for db in var.db_configs : db.name if db.primary][0]
  engine = try(var.engine_overrides[var.region], "postgres")
}
`,
			},
			mutate: func(t *testing.T, mod *Module) {
				for _, r := range mod.FindBlocks("resource", "test_database") {
					r.Block.SetAttributeValue("deletion_protection", cty.True)
				}
			},
			want: map[string]string{
				"main.tf": `# Database resource missing the new required attribute
resource "test_database" "example" {
  name                = "my-db"
  engine              = "postgres" // the DB engine
  deletion_protection = true
}

/* Reference the database */
output "db_endpoint" {
  value = test_database.example.endpoint
}
`,
				"expressions.tf": `// Database with variable references
resource "test_database" "from_ref" {
  name                = var.db_name
  engine              = var.db_engine
  deletion_protection = true
}

# Database with conditional name
resource "test_database" "conditional" {
  name                = var.env == "prod" ? "prod-db" : "dev-db" // env-based
  engine              = "postgres"
  deletion_protection = true
}

/* Database with interpolated name mixing literal and resource ref */
resource "test_database" "interpolated" {
  name                = "${var.project}-db-${test_random_id.db.hex}"
  engine              = "postgres"
  deletion_protection = true
}
`,
				"complex.tf": `# Database with map index and nested object access
resource "test_database" "complex" {
  name                = var.databases["primary"].name
  engine              = var.config.db.engine
  deletion_protection = true
}

// Database with for expression and try
resource "test_database" "for_expr" {
  name                = [for db in var.db_configs : db.name if db.primary][0]
  engine              = try(var.engine_overrides[var.region], "postgres")
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
				"main.tf": `# S3 object with literal values
resource "test_bucket_object" "file" {
  bucket = "my-bucket"
  key    = "index.html" # the object key
}

/* Another object in the same bucket */
resource "test_bucket_object" "config" {
  bucket = "my-bucket"
  key    = "config.json"
}
`,
				"expressions.tf": `// S3 object with variable references
resource "test_bucket_object" "from_ref" {
  bucket = var.bucket_name # from variable
  key    = var.object_key
}

# S3 object with conditional key
resource "test_bucket_object" "conditional" {
  bucket = var.bucket_name
  key    = var.env == "staging" ? "staging/${var.name}" : var.name // env path
}

/* S3 object with interpolation mixing literal and resource ref */
resource "test_bucket_object" "interpolated" {
  bucket = "${var.project}-${var.env}-bucket"
  key    = "${test_template.path.rendered}/index.html"
}
`,
				"complex.tf": `# S3 object with for expression in key
resource "test_bucket_object" "for_expr" {
  bucket = tolist(var.bucket_ids)[0]
  key    = join("/", [for p in var.path_segments : lower(p)])
}

// S3 object with nested object and map index
resource "test_bucket_object" "nested" {
  bucket = var.buckets[var.env].name
  key    = try(var.key_overrides[var.name], "${var.prefix}/${var.name}")
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
				"main.tf": `# S3 object with literal values
resource "test_object" "file" {
  bucket = "my-bucket"
  key    = "index.html" # the object key
}

/* Another object in the same bucket */
resource "test_object" "config" {
  bucket = "my-bucket"
  key    = "config.json"
}
`,
				"expressions.tf": `// S3 object with variable references
resource "test_object" "from_ref" {
  bucket = var.bucket_name # from variable
  key    = var.object_key
}

# S3 object with conditional key
resource "test_object" "conditional" {
  bucket = var.bucket_name
  key    = var.env == "staging" ? "staging/${var.name}" : var.name // env path
}

/* S3 object with interpolation mixing literal and resource ref */
resource "test_object" "interpolated" {
  bucket = "${var.project}-${var.env}-bucket"
  key    = "${test_template.path.rendered}/index.html"
}
`,
				"complex.tf": `# S3 object with for expression in key
resource "test_object" "for_expr" {
  bucket = tolist(var.bucket_ids)[0]
  key    = join("/", [for p in var.path_segments : lower(p)])
}

// S3 object with nested object and map index
resource "test_object" "nested" {
  bucket = var.buckets[var.env].name
  key    = try(var.key_overrides[var.name], "${var.prefix}/${var.name}")
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
				"main.tf": `# The bucket object being renamed
resource "test_bucket_object" "file" {
  bucket = "my-bucket"
  key    = "index.html"
}
`,
				"expressions.tf": `// IAM policy with direct reference
resource "test_iam_policy" "read" {
  name         = "read-policy"
  resource_arn = test_bucket_object.file.arn # direct ref
}

# IAM policy with reference inside conditional
resource "test_iam_policy" "conditional" {
  name         = "cond-policy"
  resource_arn = var.use_object ? test_bucket_object.file.arn : var.default_arn // ternary
}

/* Output with interpolation containing the reference */
output "object_url" {
  value = "https://storage.example.com/${test_bucket_object.file.id}"
}
`,
				"outputs.tf": `/* Expose object details */
output "object_id" {
  value = test_bucket_object.file.id // the object ID
}

# Also expose the version
output "object_version" {
  value = test_bucket_object.file.version_id
}
`,
				"complex.tf": `# Reference inside a for expression
output "object_arns" {
  value = [for o in [test_bucket_object.file] : o.arn]
}

// Reference inside map literal value
output "object_tags" {
  value = { id = test_bucket_object.file.id, arn = test_bucket_object.file.arn }
}

/* Reference with nested access and try */
output "object_meta" {
  value = try(test_bucket_object.file.metadata["content-type"], "application/octet-stream")
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
				"main.tf": `# The bucket object being renamed
resource "test_object" "file" {
  bucket = "my-bucket"
  key    = "index.html"
}
`,
				"expressions.tf": `// IAM policy with direct reference
resource "test_iam_policy" "read" {
  name         = "read-policy"
  resource_arn = test_object.file.arn # direct ref
}

# IAM policy with reference inside conditional
resource "test_iam_policy" "conditional" {
  name         = "cond-policy"
  resource_arn = var.use_object ? test_object.file.arn : var.default_arn // ternary
}

/* Output with interpolation containing the reference */
output "object_url" {
  value = "https://storage.example.com/${test_object.file.id}"
}
`,
				"outputs.tf": `/* Expose object details */
output "object_id" {
  value = test_object.file.id // the object ID
}

# Also expose the version
output "object_version" {
  value = test_object.file.version_id
}
`,
				"complex.tf": `# Reference inside a for expression
output "object_arns" {
  value = [for o in [test_object.file] : o.arn]
}

// Reference inside map literal value
output "object_tags" {
  value = { id = test_object.file.id, arn = test_object.file.arn }
}

/* Reference with nested access and try */
output "object_meta" {
  value = try(test_object.file.metadata["content-type"], "application/octet-stream")
}
`,
			},
		},

		// ── Category: Resource Type Rename with Moved Block ──────────
		// A resource type is renamed across the provider and a moved block
		// is emitted so Terraform can migrate state without destroy/create.
		// Real: Azure v4 azurerm_sql_server → azurerm_mssql_server,
		//       azurerm_sql_database → azurerm_mssql_database
		{
			name: "resource_type_rename_with_moved_block",
			files: map[string]string{
				"main.tf": `# SQL Server with literal values
resource "test_sql_server" "primary" {
  name    = "sql-primary"
  version = "12.0" # SQL version
}

// SQL Database on the server
resource "test_sql_database" "app" {
  name      = "app-db"
  server_id = test_sql_server.primary.id
}

/* Outputs referencing both resources */
output "server_fqdn" {
  value = test_sql_server.primary.fqdn
}

output "database_id" {
  value = test_sql_database.app.id
}
`,
				"expressions.tf": `// SQL Server with variable reference
resource "test_sql_server" "from_ref" {
  name    = var.sql_server_name
  version = var.sql_version # configurable
}

# SQL Database with conditional
resource "test_sql_database" "conditional" {
  name      = var.env == "prod" ? "prod-db" : "dev-db"
  server_id = test_sql_server.from_ref.id // server reference
}

/* SQL Server with interpolation referencing a resource */
resource "test_sql_server" "interpolated" {
  name    = "${var.project}-sql-${test_random_id.sql.hex}"
  version = "12.0"
}
`,
				"complex.tf": `# SQL Server with map index and nested object access
resource "test_sql_server" "map_lookup" {
  name    = var.servers[var.env].name
  version = try(var.sql_versions[var.region], "12.0")
}

// SQL Database with try and nested object reference
resource "test_sql_database" "complex_expr" {
  name      = try(var.db_overrides[var.env], "${var.project}-db")
  server_id = test_sql_server.map_lookup.id
  sku_name  = coalesce(var.sku_override, var.config.database.sku)
}
`,
			},
			mutate: func(t *testing.T, mod *Module) {
				// Rename test_sql_server → test_mssql_server and add moved blocks
				for _, r := range mod.FindBlocks("resource", "test_sql_server") {
					labels := r.Block.Labels()
					r.Block.SetLabels([]string{"test_mssql_server", labels[1]})
					moved := r.File.AddBlock("moved", nil)
					moved.SetAttributeRaw("from", hclwrite.TokensForTraversal(hcl.Traversal{
						hcl.TraverseRoot{Name: "test_sql_server"},
						hcl.TraverseAttr{Name: labels[1]},
					}))
					moved.SetAttributeRaw("to", hclwrite.TokensForTraversal(hcl.Traversal{
						hcl.TraverseRoot{Name: "test_mssql_server"},
						hcl.TraverseAttr{Name: labels[1]},
					}))
				}
				// Rename test_sql_database → test_mssql_database and add moved blocks
				for _, r := range mod.FindBlocks("resource", "test_sql_database") {
					labels := r.Block.Labels()
					r.Block.SetLabels([]string{"test_mssql_database", labels[1]})
					moved := r.File.AddBlock("moved", nil)
					moved.SetAttributeRaw("from", hclwrite.TokensForTraversal(hcl.Traversal{
						hcl.TraverseRoot{Name: "test_sql_database"},
						hcl.TraverseAttr{Name: labels[1]},
					}))
					moved.SetAttributeRaw("to", hclwrite.TokensForTraversal(hcl.Traversal{
						hcl.TraverseRoot{Name: "test_mssql_database"},
						hcl.TraverseAttr{Name: labels[1]},
					}))
				}
				// Update all references across the module
				mod.RenameReferencePrefix(
					makeTraversal("test_sql_server"),
					makeTraversal("test_mssql_server"),
				)
				mod.RenameReferencePrefix(
					makeTraversal("test_sql_database"),
					makeTraversal("test_mssql_database"),
				)
			},
			want: map[string]string{
				"main.tf": `# SQL Server with literal values
resource "test_mssql_server" "primary" {
  name    = "sql-primary"
  version = "12.0" # SQL version
}

// SQL Database on the server
resource "test_mssql_database" "app" {
  name      = "app-db"
  server_id = test_mssql_server.primary.id
}

/* Outputs referencing both resources */
output "server_fqdn" {
  value = test_mssql_server.primary.fqdn
}

output "database_id" {
  value = test_mssql_database.app.id
}
moved {
  from = test_sql_server.primary
  to   = test_mssql_server.primary
}
moved {
  from = test_sql_database.app
  to   = test_mssql_database.app
}
`,
				"expressions.tf": `// SQL Server with variable reference
resource "test_mssql_server" "from_ref" {
  name    = var.sql_server_name
  version = var.sql_version # configurable
}

# SQL Database with conditional
resource "test_mssql_database" "conditional" {
  name      = var.env == "prod" ? "prod-db" : "dev-db"
  server_id = test_mssql_server.from_ref.id // server reference
}

/* SQL Server with interpolation referencing a resource */
resource "test_mssql_server" "interpolated" {
  name    = "${var.project}-sql-${test_random_id.sql.hex}"
  version = "12.0"
}
moved {
  from = test_sql_server.from_ref
  to   = test_mssql_server.from_ref
}
moved {
  from = test_sql_server.interpolated
  to   = test_mssql_server.interpolated
}
moved {
  from = test_sql_database.conditional
  to   = test_mssql_database.conditional
}
`,
				"complex.tf": `# SQL Server with map index and nested object access
resource "test_mssql_server" "map_lookup" {
  name    = var.servers[var.env].name
  version = try(var.sql_versions[var.region], "12.0")
}

// SQL Database with try and nested object reference
resource "test_mssql_database" "complex_expr" {
  name      = try(var.db_overrides[var.env], "${var.project}-db")
  server_id = test_mssql_server.map_lookup.id
  sku_name  = coalesce(var.sku_override, var.config.database.sku)
}
moved {
  from = test_sql_server.map_lookup
  to   = test_mssql_server.map_lookup
}
moved {
  from = test_sql_database.complex_expr
  to   = test_mssql_database.complex_expr
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
				"main.tf": `# Cache cluster with nested cluster_config
resource "test_cache_cluster" "example" {
  description = "my-cluster" // the cluster description

  # Cluster configuration (to be flattened)
  cluster_config {
    num_shards         = 3
    replicas_per_shard = 2
  }
}

output "cluster_id" {
  value = test_cache_cluster.example.id
}
`,
				"expressions.tf": `/* Cluster with variable-driven description */
resource "test_cache_cluster" "from_ref" {
  description = var.cluster_description

  cluster_config {
    num_shards         = var.shard_count # driven by variable
    replicas_per_shard = var.replica_count
  }
}

// Cluster with conditional description
resource "test_cache_cluster" "conditional" {
  description = var.env == "prod" ? "Production Cache" : "Dev Cache"

  cluster_config {
    num_shards         = var.env == "prod" ? 5 : 2 // scale by env
    replicas_per_shard = max(var.replicas, 1)
  }
}

# Cluster with interpolated description referencing a resource
resource "test_cache_cluster" "interpolated" {
  description = "${var.project}-cache-${test_random_id.cache.hex}"

  cluster_config {
    num_shards         = 3
    replicas_per_shard = 2
  }
}
`,
				"complex.tf": `# Cluster with try, map index, and nested object access
resource "test_cache_cluster" "complex" {
  description = try(var.descriptions[var.region], "default-cache")

  cluster_config {
    num_shards         = var.shard_configs[var.env].count
    replicas_per_shard = max(var.env == "prod" ? 3 : 1, var.min_replicas)
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
				"main.tf": `# Cache cluster with nested cluster_config
resource "test_cache_cluster" "example" {
  description = "my-cluster" // the cluster description

  num_shards         = 3
  replicas_per_shard = 2
}

output "cluster_id" {
  value = test_cache_cluster.example.id
}
`,
				"expressions.tf": `/* Cluster with variable-driven description */
resource "test_cache_cluster" "from_ref" {
  description = var.cluster_description

  num_shards         = 3
  replicas_per_shard = 2
}

// Cluster with conditional description
resource "test_cache_cluster" "conditional" {
  description = var.env == "prod" ? "Production Cache" : "Dev Cache"

  num_shards         = 3
  replicas_per_shard = 2
}

# Cluster with interpolated description referencing a resource
resource "test_cache_cluster" "interpolated" {
  description = "${var.project}-cache-${test_random_id.cache.hex}"

  num_shards         = 3
  replicas_per_shard = 2
}
`,
				"complex.tf": `# Cluster with try, map index, and nested object access
resource "test_cache_cluster" "complex" {
  description = try(var.descriptions[var.region], "default-cache")

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
				"main.tf": `# The bucket that will lose its versioning block
resource "test_bucket" "example" {
  name = "my-bucket" // the bucket name

  # Versioning config (to be extracted)
  versioning {
    enabled = true
  }
}

/* Downstream reference */
output "bucket_arn" {
  value = test_bucket.example.arn
}
`,
				"expressions.tf": `// Bucket with variable name
resource "test_bucket" "from_ref" {
  name = var.bucket_name # from variable

  versioning {
    enabled = var.versioning_enabled
  }
}

# Bucket with conditional name
resource "test_bucket" "conditional" {
  name = var.env == "prod" ? "prod-bucket" : "dev-bucket"

  versioning {
    enabled = var.env == "prod" ? true : false // prod only
  }
}

/* Bucket with interpolation referencing a resource */
resource "test_bucket" "interpolated" {
  name = "${var.project}-${test_random_id.bucket.hex}"

  versioning {
    enabled = true
  }
}
`,
				"complex.tf": `# Bucket with map index and nested object access
resource "test_bucket" "map_lookup" {
  name = var.bucket_names[var.env]

  versioning {
    enabled = var.versioning_config[var.env].enabled
  }
}

// Bucket with try and for expression
resource "test_bucket" "complex_expr" {
  name = try(var.bucket_overrides[var.region], "${var.project}-${var.region}")

  versioning {
    enabled = coalesce(var.force_versioning, var.env == "prod")
  }
}
`,
			},
			mutate: func(t *testing.T, mod *Module) {
				for _, r := range mod.FindBlocks("resource", "test_bucket") {
					r.Block.RemoveBlock("versioning")

					nb := r.File.AddBlock("resource", []string{"test_bucket_versioning", r.Block.Labels()[1]})
					nb.SetAttributeRaw("bucket", hclwrite.TokensForTraversal(hcl.Traversal{
						hcl.TraverseRoot{Name: "test_bucket"},
						hcl.TraverseAttr{Name: r.Block.Labels()[1]},
						hcl.TraverseAttr{Name: "id"},
					}))
					nb.SetAttributeValue("status", cty.StringVal("Enabled"))
				}
			},
			want: map[string]string{
				"main.tf": `# The bucket that will lose its versioning block
resource "test_bucket" "example" {
  name = "my-bucket" // the bucket name

}

/* Downstream reference */
output "bucket_arn" {
  value = test_bucket.example.arn
}
resource "test_bucket_versioning" "example" {
  bucket = test_bucket.example.id
  status = "Enabled"
}
`,
				"expressions.tf": `// Bucket with variable name
resource "test_bucket" "from_ref" {
  name = var.bucket_name # from variable

}

# Bucket with conditional name
resource "test_bucket" "conditional" {
  name = var.env == "prod" ? "prod-bucket" : "dev-bucket"

}

/* Bucket with interpolation referencing a resource */
resource "test_bucket" "interpolated" {
  name = "${var.project}-${test_random_id.bucket.hex}"

}
resource "test_bucket_versioning" "from_ref" {
  bucket = test_bucket.from_ref.id
  status = "Enabled"
}
resource "test_bucket_versioning" "conditional" {
  bucket = test_bucket.conditional.id
  status = "Enabled"
}
resource "test_bucket_versioning" "interpolated" {
  bucket = test_bucket.interpolated.id
  status = "Enabled"
}
`,
				"complex.tf": `# Bucket with map index and nested object access
resource "test_bucket" "map_lookup" {
  name = var.bucket_names[var.env]

}

// Bucket with try and for expression
resource "test_bucket" "complex_expr" {
  name = try(var.bucket_overrides[var.region], "${var.project}-${var.region}")

}
resource "test_bucket_versioning" "map_lookup" {
  bucket = test_bucket.map_lookup.id
  status = "Enabled"
}
resource "test_bucket_versioning" "complex_expr" {
  bucket = test_bucket.complex_expr.id
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
				"main.tf": `# Container registry with retention policy block
resource "test_registry" "example" {
  name = "my-registry"

  # Retention configuration (being simplified to an attribute)
  retention_policy {
    days = 30 // keep images for 30 days
  }
}

output "registry_url" {
  value = test_registry.example.login_server /* the login URL */
}
`,
				"expressions.tf": `// Registry with variable name
resource "test_registry" "from_ref" {
  name = var.registry_name

  retention_policy {
    days = var.retention_days # configurable retention
  }
}

# Registry with conditional name
resource "test_registry" "conditional" {
  name = var.env == "prod" ? "prod-registry" : "dev-registry"

  retention_policy {
    days = var.env == "prod" ? 90 : 30 // longer in prod
  }
}

/* Registry with interpolation referencing a resource */
resource "test_registry" "interpolated" {
  name = "${var.project}-registry-${test_random_id.reg.hex}"

  retention_policy {
    days = 30
  }
}
`,
				"complex.tf": `# Registry with map index access
resource "test_registry" "map_lookup" {
  name = var.registries[var.env].name

  retention_policy {
    days = var.retention_configs[var.env].days
  }
}

// Registry with try and nested object access
resource "test_registry" "complex_expr" {
  name = try(var.registry_overrides[var.region], "${var.project}-registry")

  retention_policy {
    days = coalesce(var.override_days, var.config.retention.default_days)
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
				"main.tf": `# Container registry with retention policy block
resource "test_registry" "example" {
  name = "my-registry"

  retention_policy_in_days = 30
}

output "registry_url" {
  value = test_registry.example.login_server /* the login URL */
}
`,
				"expressions.tf": `// Registry with variable name
resource "test_registry" "from_ref" {
  name = var.registry_name

  retention_policy_in_days = 30
}

# Registry with conditional name
resource "test_registry" "conditional" {
  name = var.env == "prod" ? "prod-registry" : "dev-registry"

  retention_policy_in_days = 30
}

/* Registry with interpolation referencing a resource */
resource "test_registry" "interpolated" {
  name = "${var.project}-registry-${test_random_id.reg.hex}"

  retention_policy_in_days = 30
}
`,
				"complex.tf": `# Registry with map index access
resource "test_registry" "map_lookup" {
  name = var.registries[var.env].name

  retention_policy_in_days = 30
}

// Registry with try and nested object access
resource "test_registry" "complex_expr" {
  name = try(var.registry_overrides[var.region], "${var.project}-registry")

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
				"main.tf": `# Secret with old-style replication
resource "test_secret" "example" {
  name = "my-secret" // the secret name

  /* Old replication format using boolean */
  replication {
    automatic = true
  }
}

output "secret_id" {
  value = test_secret.example.id
}
`,
				"expressions.tf": `// Secret with variable name
resource "test_secret" "from_ref" {
  name = var.secret_name

  replication {
    automatic = var.auto_replicate # from variable
  }
}

# Secret with conditional name
resource "test_secret" "conditional" {
  name = var.env == "prod" ? "prod-secret" : "dev-secret"

  replication {
    automatic = var.env == "prod" // prod auto-replicates
  }
}

/* Secret with interpolation referencing a resource */
resource "test_secret" "interpolated" {
  name = "${var.project}-secret-${test_random_id.sec.hex}"

  replication {
    automatic = true
  }
}
`,
				"complex.tf": `# Secret with map index and nested object access
resource "test_secret" "map_lookup" {
  name = var.secrets[var.env].name

  replication {
    automatic = var.replication_config[var.env].automatic
  }
}

// Secret with try and type constructor
resource "test_secret" "complex_expr" {
  name = try(var.secret_overrides[var.region], tostring(var.default_secret_id))

  replication {
    automatic = alltrue([var.auto_replicate, var.region_config.allow_auto])
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
				"main.tf": `# Secret with old-style replication
resource "test_secret" "example" {
  name = "my-secret" // the secret name

  /* Old replication format using boolean */
  replication {
    auto {
    }
  }
}

output "secret_id" {
  value = test_secret.example.id
}
`,
				"expressions.tf": `// Secret with variable name
resource "test_secret" "from_ref" {
  name = var.secret_name

  replication {
    auto {
    }
  }
}

# Secret with conditional name
resource "test_secret" "conditional" {
  name = var.env == "prod" ? "prod-secret" : "dev-secret"

  replication {
    auto {
    }
  }
}

/* Secret with interpolation referencing a resource */
resource "test_secret" "interpolated" {
  name = "${var.project}-secret-${test_random_id.sec.hex}"

  replication {
    auto {
    }
  }
}
`,
				"complex.tf": `# Secret with map index and nested object access
resource "test_secret" "map_lookup" {
  name = var.secrets[var.env].name

  replication {
    auto {
    }
  }
}

// Secret with try and type constructor
resource "test_secret" "complex_expr" {
  name = try(var.secret_overrides[var.region], tostring(var.default_secret_id))

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
				"main.tf": `# Cluster with flat network attribute
resource "test_cluster" "example" {
  name    = "my-cluster"
  network = "projects/p/global/networks/default" # the VPC network
}

/* Reference cluster endpoint */
output "cluster_endpoint" {
  value = test_cluster.example.endpoint
}
`,
				"expressions.tf": `// Cluster with variable reference for network
resource "test_cluster" "from_ref" {
  name    = var.cluster_name
  network = var.network_id # from variable
}

# Cluster with conditional network
resource "test_cluster" "conditional" {
  name    = var.env == "prod" ? "prod-cluster" : "dev-cluster"
  network = var.env == "prod" ? var.prod_network : var.dev_network // env-based
}

/* Cluster with interpolation for network, referencing a resource */
resource "test_cluster" "interpolated" {
  name    = "${var.project}-cluster"
  network = "projects/${var.project}/global/networks/${test_network.main.name}"
}
`,
				"complex.tf": `# Cluster with map index and nested object access
resource "test_cluster" "map_lookup" {
  name    = var.clusters[var.env].name
  network = var.network_map[var.region].id
}

// Cluster with try and for expression
resource "test_cluster" "complex_expr" {
  name    = try(var.cluster_overrides[var.region], "${var.project}-${var.region}")
  network = coalesce(var.network_override, var.networks[var.env].self_link)
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
				"main.tf": `# Cluster with flat network attribute
resource "test_cluster" "example" {
  name = "my-cluster"
  network_config {
    network = "projects/p/global/networks/default"
  }
}

/* Reference cluster endpoint */
output "cluster_endpoint" {
  value = test_cluster.example.endpoint
}
`,
				"expressions.tf": `// Cluster with variable reference for network
resource "test_cluster" "from_ref" {
  name = var.cluster_name
  network_config {
    network = "projects/p/global/networks/default"
  }
}

# Cluster with conditional network
resource "test_cluster" "conditional" {
  name = var.env == "prod" ? "prod-cluster" : "dev-cluster"
  network_config {
    network = "projects/p/global/networks/default"
  }
}

/* Cluster with interpolation for network, referencing a resource */
resource "test_cluster" "interpolated" {
  name = "${var.project}-cluster"
  network_config {
    network = "projects/p/global/networks/default"
  }
}
`,
				"complex.tf": `# Cluster with map index and nested object access
resource "test_cluster" "map_lookup" {
  name = var.clusters[var.env].name
  network_config {
    network = "projects/p/global/networks/default"
  }
}

// Cluster with try and for expression
resource "test_cluster" "complex_expr" {
  name = try(var.cluster_overrides[var.region], "${var.project}-${var.region}")
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
				"backend.tf": `# Terraform settings with remote backend
terraform {
  /* This backend is being migrated to cloud */
  backend "remote" {
    hostname     = "app.terraform.io"
    organization = "my-org" // the org

    workspaces {
      name = "my-workspace"
    }
  }
}
`,
				"main.tf": `# Main configuration is untouched
resource "test_instance" "web" {
  ami           = var.web_ami
  instance_type = var.env == "prod" ? "t2.large" : "t2.micro" // scale by env
}

/* Output with interpolation referencing a resource */
output "web_url" {
  value = "https://${test_instance.web.public_ip}:443"
}
`,
				"complex.tf": `# Resources unaffected by backend migration
resource "test_database" "main" {
  name   = var.databases[var.env].name
  engine = try(var.engine_overrides[var.region], "postgres")
}

// Data source with complex expression
data "test_ami" "latest" {
  owners      = [for o in var.ami_owners : tostring(o)]
  name_filter = coalesce(var.ami_filter, "ubuntu-*")
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
				"backend.tf": `# Terraform settings with remote backend
terraform {
  /* This backend is being migrated to cloud */
  cloud {
    hostname     = "app.terraform.io"
    organization = "my-org"
    workspaces {
      name = "my-workspace"
    }
  }
}
`,
				"main.tf": `# Main configuration is untouched
resource "test_instance" "web" {
  ami           = var.web_ami
  instance_type = var.env == "prod" ? "t2.large" : "t2.micro" // scale by env
}

/* Output with interpolation referencing a resource */
output "web_url" {
  value = "https://${test_instance.web.public_ip}:443"
}
`,
				"complex.tf": `# Resources unaffected by backend migration
resource "test_database" "main" {
  name   = var.databases[var.env].name
  engine = try(var.engine_overrides[var.region], "postgres")
}

// Data source with complex expression
data "test_ami" "latest" {
  owners      = [for o in var.ami_owners : tostring(o)]
  name_filter = coalesce(var.ami_filter, "ubuntu-*")
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
				"main.tf": `# Alert rule with literal nested attribute
resource "test_alert_rule" "example" {
  name = "my-rule"

  # Incident configuration
  incident_config {
    create_incident   = true
    group_by_entities = ["Host"] // the grouping
  }
}

output "rule_id" {
  value = test_alert_rule.example.id /* the rule ID */
}
`,
				"expressions.tf": `// Alert rule with variable reference in nested block
resource "test_alert_rule" "from_ref" {
  name = var.rule_name

  incident_config {
    create_incident   = var.create_incident
    group_by_entities = var.group_entities # from variable
  }
}

# Alert rule with conditional in nested block
resource "test_alert_rule" "conditional" {
  name = var.env == "prod" ? "prod-rule" : "dev-rule"

  incident_config {
    create_incident   = true
    group_by_entities = var.use_custom ? var.custom_entities : ["Host"] // conditional
  }
}

/* Alert rule with function call in nested block */
resource "test_alert_rule" "with_func" {
  name = "${var.project}-rule"

  incident_config {
    create_incident   = true
    group_by_entities = concat(var.base_entities, ["Host"])
  }
}
`,
				"complex.tf": `# Alert rule with map index in nested block
resource "test_alert_rule" "map_lookup" {
  name = var.rules[var.env].name

  incident_config {
    create_incident   = var.incident_config[var.env].create
    group_by_entities = var.entity_groups[var.alert_level]
  }
}

// Alert rule with try and for expression in nested block
resource "test_alert_rule" "complex_expr" {
  name = try(var.rule_overrides[var.region], "${var.project}-rule")

  incident_config {
    create_incident   = coalesce(var.force_incident, var.env == "prod")
    group_by_entities = [for e in var.entities : e.name if e.groupable]
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
				"main.tf": `# Alert rule with literal nested attribute
resource "test_alert_rule" "example" {
  name = "my-rule"

  # Incident configuration
  incident_config {
    create_incident = true
    by_entities     = ["Host"] // the grouping
  }
}

output "rule_id" {
  value = test_alert_rule.example.id /* the rule ID */
}
`,
				"expressions.tf": `// Alert rule with variable reference in nested block
resource "test_alert_rule" "from_ref" {
  name = var.rule_name

  incident_config {
    create_incident = var.create_incident
    by_entities     = var.group_entities # from variable
  }
}

# Alert rule with conditional in nested block
resource "test_alert_rule" "conditional" {
  name = var.env == "prod" ? "prod-rule" : "dev-rule"

  incident_config {
    create_incident = true
    by_entities     = var.use_custom ? var.custom_entities : ["Host"] // conditional
  }
}

/* Alert rule with function call in nested block */
resource "test_alert_rule" "with_func" {
  name = "${var.project}-rule"

  incident_config {
    create_incident = true
    by_entities     = concat(var.base_entities, ["Host"])
  }
}
`,
				"complex.tf": `# Alert rule with map index in nested block
resource "test_alert_rule" "map_lookup" {
  name = var.rules[var.env].name

  incident_config {
    create_incident = var.incident_config[var.env].create
    by_entities     = var.entity_groups[var.alert_level]
  }
}

// Alert rule with try and for expression in nested block
resource "test_alert_rule" "complex_expr" {
  name = try(var.rule_overrides[var.region], "${var.project}-rule")

  incident_config {
    create_incident = coalesce(var.force_incident, var.env == "prod")
    by_entities     = [for e in var.entities : e.name if e.groupable]
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
				"main.tf": `# Container with repeated IP allow blocks
resource "test_container" "example" {
  name = "my-app" // the app name

  # First allowed range
  allowed_ip {
    cidr = "10.0.0.0/8"
  }

  /* Second allowed range */
  allowed_ip {
    cidr = "172.16.0.0/12"
  }
}

output "container_id" {
  value = test_container.example.id
}
`,
				"expressions.tf": `// Container with variable name
resource "test_container" "from_ref" {
  name = var.app_name

  allowed_ip {
    cidr = var.cidr_primary # primary range
  }

  allowed_ip {
    cidr = var.cidr_secondary
  }
}

# Container with conditional name
resource "test_container" "conditional" {
  name = var.env == "prod" ? "prod-app" : "dev-app"

  allowed_ip {
    cidr = var.env == "prod" ? "10.0.0.0/8" : "172.16.0.0/12" // env-based
  }
}

/* Container with interpolated name referencing a resource */
resource "test_container" "interpolated" {
  name = "${var.project}-app-${test_random_id.app.hex}"

  allowed_ip {
    cidr = "10.0.0.0/8"
  }
}
`,
				"complex.tf": `# Container with map index for name
resource "test_container" "map_lookup" {
  name = var.containers[var.env].name

  allowed_ip {
    cidr = var.cidr_map[var.region].primary
  }

  allowed_ip {
    cidr = var.cidr_map[var.region].secondary
  }
}

// Container with try and nested object access
resource "test_container" "complex_expr" {
  name = try(var.container_overrides[var.region], "${var.project}-app")

  allowed_ip {
    cidr = cidrsubnet(var.base_cidr, 8, 0)
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
				"main.tf": `# Container with repeated IP allow blocks
resource "test_container" "example" {
  name = "my-app" // the app name


  /* Second allowed range */
  allowed_ips = ["10.0.0.0/8", "172.16.0.0/12"]
}

output "container_id" {
  value = test_container.example.id
}
`,
				"expressions.tf": `// Container with variable name
resource "test_container" "from_ref" {
  name = var.app_name


  allowed_ips = ["10.0.0.0/8", "172.16.0.0/12"]
}

# Container with conditional name
resource "test_container" "conditional" {
  name = var.env == "prod" ? "prod-app" : "dev-app"

  allowed_ips = ["10.0.0.0/8", "172.16.0.0/12"]
}

/* Container with interpolated name referencing a resource */
resource "test_container" "interpolated" {
  name = "${var.project}-app-${test_random_id.app.hex}"

  allowed_ips = ["10.0.0.0/8", "172.16.0.0/12"]
}
`,
				"complex.tf": `# Container with map index for name
resource "test_container" "map_lookup" {
  name = var.containers[var.env].name


  allowed_ips = ["10.0.0.0/8", "172.16.0.0/12"]
}

// Container with try and nested object access
resource "test_container" "complex_expr" {
  name = try(var.container_overrides[var.region], "${var.project}-app")

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
				"main.tf": `# Stream with old enum value
resource "test_stream" "example" {
  name        = "my-stream" // the stream name
  destination = "s3"        # old value, must become extended_s3
}

/* Downstream reference */
output "stream_arn" {
  value = test_stream.example.arn
}
`,
				"expressions.tf": `// Stream with variable name
resource "test_stream" "from_ref" {
  name        = var.stream_name
  destination = var.destination_type # from variable
}

# Stream with conditional name
resource "test_stream" "conditional" {
  name        = var.env == "prod" ? "prod-stream" : "dev-stream"
  destination = var.legacy ? "s3" : "kinesis" // conditional
}

/* Stream with interpolation referencing a resource */
resource "test_stream" "interpolated" {
  name        = "${var.project}-stream-${test_random_id.stream.hex}"
  destination = "s3"
}
`,
				"complex.tf": `# Stream with map index access
resource "test_stream" "map_lookup" {
  name        = var.streams[var.env].name
  destination = var.destination_map[var.region]
}

// Stream with try and nested object access
resource "test_stream" "complex_expr" {
  name        = try(var.stream_overrides[var.region], "${var.project}-stream")
  destination = coalesce(var.dest_override, var.config.streaming.destination)
}
`,
			},
			mutate: func(t *testing.T, mod *Module) {
				for _, r := range mod.FindBlocks("resource", "test_stream") {
					r.Block.SetAttributeValue("destination", cty.StringVal("extended_s3"))
				}
			},
			want: map[string]string{
				"main.tf": `# Stream with old enum value
resource "test_stream" "example" {
  name        = "my-stream"   // the stream name
  destination = "extended_s3" # old value, must become extended_s3
}

/* Downstream reference */
output "stream_arn" {
  value = test_stream.example.arn
}
`,
				"expressions.tf": `// Stream with variable name
resource "test_stream" "from_ref" {
  name        = var.stream_name
  destination = "extended_s3" # from variable
}

# Stream with conditional name
resource "test_stream" "conditional" {
  name        = var.env == "prod" ? "prod-stream" : "dev-stream"
  destination = "extended_s3" // conditional
}

/* Stream with interpolation referencing a resource */
resource "test_stream" "interpolated" {
  name        = "${var.project}-stream-${test_random_id.stream.hex}"
  destination = "extended_s3"
}
`,
				"complex.tf": `# Stream with map index access
resource "test_stream" "map_lookup" {
  name        = var.streams[var.env].name
  destination = "extended_s3"
}

// Stream with try and nested object access
resource "test_stream" "complex_expr" {
  name        = try(var.stream_overrides[var.region], "${var.project}-stream")
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
				"main.tf": `# Root module calling the network module
module "network" {
  source   = "./modules/network"
  vpc_cidr = "10.0.0.0/16" // the CIDR range
}

/* Use the module output */
resource "test_instance" "web" {
  subnet_id = module.network.subnet_id
}
`,
				"expressions.tf": `// Module call with variable reference
module "network_staging" {
  source   = "./modules/network"
  vpc_cidr = var.staging_cidr # from tfvars
}

# Module call with conditional
module "network_conditional" {
  source   = "./modules/network"
  vpc_cidr = var.env == "prod" ? "10.0.0.0/16" : "172.16.0.0/16"
}

/* Module call with interpolation referencing a resource */
module "network_dynamic" {
  source   = "./modules/network"
  vpc_cidr = "${var.cidr_prefix}.0.0/16"
}
`,
				"complex.tf": `# Module call with map index for CIDR
module "network_map" {
  source   = "./modules/network"
  vpc_cidr = var.cidr_map[var.region]
}

// Module call with try and nested object access
module "network_complex" {
  source   = "./modules/network"
  vpc_cidr = try(var.network_config[var.env].cidr, "10.0.0.0/16")
}
`,
			},
			childName: "network",
			childFiles: map[string]string{
				"main.tf": `# Network module resources
variable "vpc_cidr" {
  type = string // CIDR for the VPC
}

/* Create the VPC */
resource "test_vpc" "main" {
  cidr_block = var.vpc_cidr
}
`,
				"outputs.tf": `# Network module outputs
output "subnet_id" {
  value = test_subnet.main.id // the primary subnet
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
				// Update the argument name in all parent module calls.
				for _, name := range []string{"network", "network_staging", "network_conditional", "network_dynamic", "network_map", "network_complex"} {
					for _, r := range rootMod.FindBlocks("module", name) {
						r.Block.RenameAttribute("vpc_cidr", "cidr_block")
					}
				}
			},
			wantRoot: map[string]string{
				"main.tf": `# Root module calling the network module
module "network" {
  source     = "./modules/network"
  cidr_block = "10.0.0.0/16" // the CIDR range
}

/* Use the module output */
resource "test_instance" "web" {
  subnet_id = module.network.subnet_id
}
`,
				"expressions.tf": `// Module call with variable reference
module "network_staging" {
  source     = "./modules/network"
  cidr_block = var.staging_cidr # from tfvars
}

# Module call with conditional
module "network_conditional" {
  source     = "./modules/network"
  cidr_block = var.env == "prod" ? "10.0.0.0/16" : "172.16.0.0/16"
}

/* Module call with interpolation referencing a resource */
module "network_dynamic" {
  source     = "./modules/network"
  cidr_block = "${var.cidr_prefix}.0.0/16"
}
`,
				"complex.tf": `# Module call with map index for CIDR
module "network_map" {
  source     = "./modules/network"
  cidr_block = var.cidr_map[var.region]
}

// Module call with try and nested object access
module "network_complex" {
  source     = "./modules/network"
  cidr_block = try(var.network_config[var.env].cidr, "10.0.0.0/16")
}
`,
			},
			wantChild: map[string]string{
				"main.tf": `# Network module resources
variable "cidr_block" {
  type = string // CIDR for the VPC
}

/* Create the VPC */
resource "test_vpc" "main" {
  cidr_block = var.cidr_block
}
`,
				"outputs.tf": `# Network module outputs
output "subnet_id" {
  value = test_subnet.main.id // the primary subnet
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
				"main.tf": `# Root module calling network
module "network" {
  source = "./modules/network"
}

/* Instance using the module output directly */
resource "test_instance" "web" {
  subnet_id = module.network.subnet_id // old output name
}
`,
				"expressions.tf": `// Resource with reference inside conditional
resource "test_instance" "conditional" {
  subnet_id = var.custom_subnet != "" ? var.custom_subnet : module.network.subnet_id
}

# Output with interpolation containing module reference
output "subnet_info" {
  value = "Subnet: ${module.network.subnet_id}" /* inline doc */
}

/* Security group with direct reference */
resource "test_security_group" "web" {
  name   = "web-sg"
  vpc_id = module.network.subnet_id # same old output
}
`,
				"complex.tf": `# Reference inside a for expression
output "all_subnets" {
  value = [for s in [module.network.subnet_id] : "subnet-${s}"]
}

// Reference inside map literal
output "network_info" {
  value = { primary = module.network.subnet_id, backup = var.backup_subnet }
}

/* Reference with try */
output "safe_subnet" {
  value = try(module.network.subnet_id, var.fallback_subnet)
}
`,
			},
			childName: "network",
			childFiles: map[string]string{
				"outputs.tf": `# Network module outputs
output "subnet_id" {
  value       = test_subnet.main.id
  description = "The primary subnet" /* inline doc */
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
				"main.tf": `# Root module calling network
module "network" {
  source = "./modules/network"
}

/* Instance using the module output directly */
resource "test_instance" "web" {
  subnet_id = module.network.primary_subnet_id // old output name
}
`,
				"expressions.tf": `// Resource with reference inside conditional
resource "test_instance" "conditional" {
  subnet_id = var.custom_subnet != "" ? var.custom_subnet : module.network.primary_subnet_id
}

# Output with interpolation containing module reference
output "subnet_info" {
  value = "Subnet: ${module.network.primary_subnet_id}" /* inline doc */
}

/* Security group with direct reference */
resource "test_security_group" "web" {
  name   = "web-sg"
  vpc_id = module.network.primary_subnet_id # same old output
}
`,
				"complex.tf": `# Reference inside a for expression
output "all_subnets" {
  value = [for s in [module.network.primary_subnet_id] : "subnet-${s}"]
}

// Reference inside map literal
output "network_info" {
  value = { primary = module.network.primary_subnet_id, backup = var.backup_subnet }
}

/* Reference with try */
output "safe_subnet" {
  value = try(module.network.primary_subnet_id, var.fallback_subnet)
}
`,
			},
			wantChild: map[string]string{
				"outputs.tf": `# Network module outputs
output "primary_subnet_id" {
  value       = test_subnet.main.id
  description = "The primary subnet" /* inline doc */
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
