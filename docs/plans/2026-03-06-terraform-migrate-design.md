# Terraform Migrate Subcommand — Prototype Design

## Goal

Build a prototype `terraform migrate` subcommand that migrates user source code
(`.tf` files) to accommodate breaking changes from providers or Terraform Core.
The prototype focuses on user experience for demo to customers and product people.
Transformations are implemented superficially (regex-based) while the command
infrastructure follows real Terraform patterns.

## Commands

### `terraform migrate list`

Lists all applicable migrations for the current codebase, grouped by provider.
Shows the first 3 sub-migrations per set, with a count of remaining.

```
$ terraform migrate list

hashicorp/aws (1 migration available):
  v3-to-v4        Migrate AWS provider from v3 to v4    12 files, 23 changes
    - s3-bucket-acl           Extract ACL to aws_s3_bucket_acl
    - s3-bucket-cors          Extract CORS to aws_s3_bucket_cors_configuration
    - s3-bucket-logging       Extract logging to aws_s3_bucket_logging
    (+3 more, use -detail to list all)

hashicorp/azurerm (1 migration available):
  v3-to-v4        Migrate AzureRM provider from v3 to v4    5 files, 8 changes
    - subnet-delegation       Extract delegation to azurerm_subnet_delegation
    - network-security-rule   Extract rules to azurerm_network_security_rule
    - storage-account-network Extract network rules to separate resource
    (+1 more, use -detail to list all)

terraform (1 migration available):
  v1.x-to-v2.x    Update terraform block syntax             3 files, 3 changes
    - required-providers-map  Convert required_providers to object syntax
    - backend-to-cloud        Migrate backend block to cloud block
    (2 sub-migrations total)
```

With `-detail`, all sub-migrations expand.

### `terraform migrate <namespace/provider/migration>`

Runs a migration set. Three modes:

**Default (no flags):** Apply immediately, show progress.

```
$ terraform migrate hashicorp/aws/v3-to-v4

Applying hashicorp/aws/v3-to-v4...
  ✓ s3-bucket-acl          (main.tf, s3.tf)
  ✓ s3-bucket-cors         (s3.tf)
  ✓ s3-bucket-logging      (s3.tf)

Applied 3 changes across 2 files.
```

**`-step` flag:** Interactive per-sub-migration approval with diff shown.

```
$ terraform migrate hashicorp/aws/v3-to-v4 -step

[1/6] s3-bucket-acl: Extract ACL to aws_s3_bucket_acl

--- main.tf
+++ main.tf
@@ -12,6 +12,10 @@
 resource "aws_s3_bucket" "example" {
-  acl    = "private"
   bucket = "my-bucket"
 }
+resource "aws_s3_bucket_acl" "example" {
+  bucket = aws_s3_bucket.example.id
+  acl    = "private"
+}

Apply this change? [y]es / [n]o / [q]uit: y

[2/6] s3-bucket-cors: Extract CORS to aws_s3_bucket_cors_configuration
...
```

**`-dry-run` flag:** Show combined diff of all changes, then exit. No prompt, no modifications.

```
$ terraform migrate hashicorp/aws/v3-to-v4 -dry-run

Planning hashicorp/aws/v3-to-v4...

--- main.tf
+++ main.tf
@@ -12,20 +12,35 @@
  ... combined diff of all sub-migrations ...

--- s3.tf
+++ s3.tf
  ... combined diff ...

6 changes would be applied across 2 files.
```

## Architecture

### File Layout

```
commands.go                            # Register "migrate" and "migrate list"

internal/command/
  migrate_command.go                   # Parent command (shows help, returns cli.RunResultHelp)
  migrate_list.go                      # "migrate list" subcommand
  migrate_apply.go                     # "migrate <id>" — handles default, -step, -dry-run
  migrate_apply_test.go
  migrate_list_test.go

internal/command/arguments/
  migrate.go                           # ParseMigrateList, ParseMigrateApply

internal/command/views/
  migrate.go                           # MigrateListView, MigrateApplyView (Human + JSON)

internal/command/migrate/
  registry.go                          # Hardcoded migration registry
  migration.go                         # Migration, SubMigration types
  engine.go                            # Transformation engine (applies SubMigrations to files)
  engine_test.go                       # Engine unit tests
  migrations_aws.go                    # AWS S3 v3->v4 sample migrations
  migrations_azurerm.go                # Azure v3->v4 sample migrations
  migrations_terraform.go              # Terraform Core v1.x->v2.x sample migrations
```

### Key Types

```go
// migration.go
type Migration struct {
    Namespace     string           // "hashicorp"
    Provider      string           // "aws"
    Name          string           // "v3-to-v4"
    Description   string
    SubMigrations []SubMigration
}

func (m Migration) ID() string {
    return m.Namespace + "/" + m.Provider + "/" + m.Name
}

type SubMigration struct {
    Name        string
    Description string
    Apply       func(filename string, src []byte) ([]byte, error)
}
```

### Patterns Used

- **Commands** embed `Meta`, call `c.Meta.process(args)`, use `c.Meta.defaultFlagSet()`
- **Arguments** parsed via `arguments.ParseMigrateList()` / `arguments.ParseMigrateApply()`
  returning typed structs, following `arguments/import.go` pattern
- **Views** follow `views/apply.go` pattern: interface with `Human` and `JSON`
  implementations, constructed from `*View` base
- **Diagnostics** via `tfdiags.Diagnostics` throughout
- **Parent command** follows `state_command.go` pattern (returns `cli.RunResultHelp`)
- **Registration** in `commands.go` with `"migrate"` and `"migrate list"` keys

### Sample Migrations

1. **hashicorp/aws/v3-to-v4** — S3 bucket refactoring (real AWS provider v3->v4 change)
   - `s3-bucket-acl`: Extract `acl` argument to `aws_s3_bucket_acl` resource
   - `s3-bucket-cors`: Extract `cors_rule` block to `aws_s3_bucket_cors_configuration`
   - `s3-bucket-logging`: Extract `logging` block to `aws_s3_bucket_logging`

2. **hashicorp/azurerm/v3-to-v4** — Azure resource extractions
   - `subnet-delegation`: Extract inline delegation to `azurerm_subnet_delegation`
   - `network-security-rule`: Extract inline security rules to `azurerm_network_security_rule`

3. **terraform/terraform/v1.x-to-v2.x** — Core syntax updates
   - `required-providers-map`: Convert `required_providers` from map to object syntax
   - `backend-to-cloud`: Migrate `backend` block to `cloud` block

### Transformation Engine

Each `SubMigration.Apply` function takes `(filename string, src []byte)` and returns
`([]byte, error)`. For the prototype, these use regex replacements to transform
HCL source. The engine iterates over `.tf` files in the working directory and
applies each sub-migration.

### Testing Strategy

- **Engine unit tests** (`engine_test.go`): Table-driven tests giving source bytes
  to `SubMigration.Apply`, asserting output bytes match expected.
- **Command integration tests** (`migrate_apply_test.go`, `migrate_list_test.go`):
  Following `fmt_test.go` patterns — create temp dirs with `.tf` files, run the
  command, assert file contents and exit codes.
- **Mode tests**: Verify `-dry-run` does not modify files, `-step` handles y/n/q
  input correctly, default mode applies all changes.
