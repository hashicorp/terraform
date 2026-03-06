# Terraform Migrate Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a prototype `terraform migrate` subcommand that runs source-code migrations published by providers or Terraform Core, with three UX modes (default, -step, -dry-run).

**Architecture:** New command package `internal/command/migrate/` holds types, registry, and engine. Commands in `internal/command/` follow existing Meta/arguments/views patterns. Migrations are hardcoded Go structs with regex-based Apply functions.

**Tech Stack:** Go, standard `regexp` package, `github.com/hashicorp/cli`, existing Terraform command infrastructure (Meta, views, arguments, tfdiags).

---

### Task 1: Migration types and registry

**Files:**
- Create: `internal/command/migrate/migration.go`
- Create: `internal/command/migrate/registry.go`
- Create: `internal/command/migrate/registry_test.go`

**Step 1: Write the types file**

Create `internal/command/migrate/migration.go`:

```go
// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package migrate

// Migration represents a set of sub-migrations published by a provider or
// Terraform Core. Users run the full set as a unit.
type Migration struct {
	Namespace     string         // e.g. "hashicorp"
	Provider      string         // e.g. "aws"
	Name          string         // e.g. "v3-to-v4"
	Description   string         // human-readable summary
	SubMigrations []SubMigration // ordered list of atomic changes
}

// ID returns the fully qualified migration identifier.
func (m Migration) ID() string {
	return m.Namespace + "/" + m.Provider + "/" + m.Name
}

// SubMigration is one atomic transformation within a migration set.
type SubMigration struct {
	Name        string // e.g. "s3-bucket-acl"
	Description string // e.g. "Extract ACL to aws_s3_bucket_acl"

	// Apply transforms source bytes for a single file. It returns the
	// transformed bytes (possibly unchanged) and any error.
	Apply func(filename string, src []byte) ([]byte, error)
}
```

**Step 2: Write the registry with tests**

Create `internal/command/migrate/registry.go`:

```go
// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package migrate

import "fmt"

// Registry holds all known migrations.
type Registry struct {
	migrations []Migration
}

// NewRegistry returns a registry populated with all built-in migrations.
func NewRegistry() *Registry {
	r := &Registry{}
	r.migrations = append(r.migrations, awsMigrations()...)
	r.migrations = append(r.migrations, azurermMigrations()...)
	r.migrations = append(r.migrations, terraformMigrations()...)
	return r
}

// All returns every registered migration.
func (r *Registry) All() []Migration {
	return r.migrations
}

// Find returns the migration matching the given fully-qualified ID,
// or an error if not found.
func (r *Registry) Find(id string) (Migration, error) {
	for _, m := range r.migrations {
		if m.ID() == id {
			return m, nil
		}
	}
	return Migration{}, fmt.Errorf("migration %q not found", id)
}
```

Create `internal/command/migrate/registry_test.go`:

```go
// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package migrate

import "testing"

func TestRegistryAll(t *testing.T) {
	r := NewRegistry()
	all := r.All()
	if len(all) == 0 {
		t.Fatal("expected at least one migration in registry")
	}
}

func TestRegistryFind(t *testing.T) {
	r := NewRegistry()

	t.Run("found", func(t *testing.T) {
		m, err := r.Find("hashicorp/aws/v3-to-v4")
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if m.ID() != "hashicorp/aws/v3-to-v4" {
			t.Fatalf("wrong ID: got %s", m.ID())
		}
		if len(m.SubMigrations) == 0 {
			t.Fatal("expected sub-migrations")
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := r.Find("nonexistent/provider/migration")
		if err == nil {
			t.Fatal("expected error for missing migration")
		}
	})
}
```

**Step 3: Create stub migration files so registry compiles**

Create minimal `internal/command/migrate/migrations_aws.go`, `migrations_azurerm.go`, `migrations_terraform.go` that each return an empty slice:

```go
// Example: migrations_aws.go
// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package migrate

func awsMigrations() []Migration {
	return []Migration{}
}
```

Same pattern for `azurermMigrations()` and `terraformMigrations()`.

**Step 4: Run tests**

Run: `go test ./internal/command/migrate/...`
Expected: registry_test.go tests FAIL (TestRegistryAll fails because no migrations yet, TestRegistryFind "found" fails). This is expected — we'll add real migrations in Task 2.

**Step 5: Commit**

```
feat: add migrate package with types and registry
```

---

### Task 2: Sample migrations (AWS, Azure, Terraform Core)

**Files:**
- Modify: `internal/command/migrate/migrations_aws.go`
- Modify: `internal/command/migrate/migrations_azurerm.go`
- Modify: `internal/command/migrate/migrations_terraform.go`
- Create: `internal/command/migrate/migrations_aws_test.go`
- Create: `internal/command/migrate/migrations_azurerm_test.go`
- Create: `internal/command/migrate/migrations_terraform_test.go`

**Step 1: Write AWS migration tests**

Create `internal/command/migrate/migrations_aws_test.go` with table-driven tests. Each test case has `input` and `expected` HCL source bytes. Test at minimum:

- `s3-bucket-acl`: Input has `resource "aws_s3_bucket"` with `acl = "private"` inline. Expected output removes the `acl` line and adds a new `resource "aws_s3_bucket_acl"` block.
- `s3-bucket-cors`: Input has `cors_rule { ... }` block inside `aws_s3_bucket`. Expected output extracts it to `aws_s3_bucket_cors_configuration`.
- `s3-bucket-logging`: Input has `logging { ... }` block. Expected output extracts to `aws_s3_bucket_logging`.
- A no-op case: Input has no matching patterns, output is unchanged.

```go
// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package migrate

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestAWSS3BucketACL(t *testing.T) {
	migrations := awsMigrations()
	if len(migrations) == 0 {
		t.Fatal("no AWS migrations found")
	}

	// Find the s3-bucket-acl sub-migration
	var aclMigration SubMigration
	for _, sm := range migrations[0].SubMigrations {
		if sm.Name == "s3-bucket-acl" {
			aclMigration = sm
			break
		}
	}
	if aclMigration.Name == "" {
		t.Fatal("s3-bucket-acl sub-migration not found")
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "extracts acl to separate resource",
			input: `resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
  acl    = "private"
}
`,
			expected: `resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
}

resource "aws_s3_bucket_acl" "example" {
  bucket = aws_s3_bucket.example.id
  acl    = "private"
}
`,
		},
		{
			name: "no change when no acl present",
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

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := aclMigration.Apply("main.tf", []byte(tc.input))
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if diff := cmp.Diff(tc.expected, string(got)); diff != "" {
				t.Fatalf("unexpected output (-want +got):\n%s", diff)
			}
		})
	}
}
```

Write similar test files for Azure and Terraform Core migrations.

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/command/migrate/...`
Expected: FAIL — stub functions return empty slices.

**Step 3: Implement AWS migrations**

Fill in `migrations_aws.go` with regex-based `Apply` functions. Each sub-migration uses `regexp.MustCompile` to find patterns in HCL source and transform them. Keep regexes simple — this is a prototype.

For `s3-bucket-acl`:
- Match `resource "aws_s3_bucket" "(\w+)" {` to capture the resource name
- Find and remove the `acl = "..."` line within the block
- Append a new `resource "aws_s3_bucket_acl" "<name>" { ... }` block

For `s3-bucket-cors`:
- Match and extract `cors_rule { ... }` block from `aws_s3_bucket`
- Create new `aws_s3_bucket_cors_configuration` resource

For `s3-bucket-logging`:
- Match and extract `logging { ... }` block
- Create new `aws_s3_bucket_logging` resource

**Step 4: Implement Azure migrations**

Fill in `migrations_azurerm.go`:

For `subnet-delegation`:
- Extract inline `delegation { ... }` block from `azurerm_subnet`
- Create new `azurerm_subnet_delegation` resource

For `network-security-rule`:
- Extract inline `security_rule { ... }` block from `azurerm_network_security_group`
- Create new `azurerm_network_security_rule` resource

**Step 5: Implement Terraform Core migrations**

Fill in `migrations_terraform.go`:

For `required-providers-map`:
- Convert `required_providers { aws = "~> 3.0" }` to `required_providers { aws = { source = "hashicorp/aws" version = "~> 3.0" } }`

For `backend-to-cloud`:
- Convert `backend "remote" { ... }` to `cloud { ... }` block

**Step 6: Run all tests**

Run: `go test ./internal/command/migrate/...`
Expected: All PASS

**Step 7: Commit**

```
feat: add sample migrations for AWS, Azure, and Terraform Core
```

---

### Task 3: Migration engine

**Files:**
- Create: `internal/command/migrate/engine.go`
- Create: `internal/command/migrate/engine_test.go`

**Step 1: Write engine tests**

The engine takes a directory path and a `Migration`, scans for `.tf` files, applies each sub-migration to each file, and returns a result. Test:

- Engine applies sub-migrations to `.tf` files in a temp directory
- Files without matches are unchanged
- Non-`.tf` files are ignored
- Results track which files changed and which sub-migrations matched

```go
// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package migrate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEngineApply(t *testing.T) {
	dir := t.TempDir()

	// Write a test .tf file
	input := `resource "aws_s3_bucket" "test" {
  bucket = "my-bucket"
  acl    = "private"
}
`
	err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(input), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Write a non-tf file that should be ignored
	err = os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# hello"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	reg := NewRegistry()
	m, _ := reg.Find("hashicorp/aws/v3-to-v4")

	results, err := Apply(dir, m)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}

	// Verify file was not modified (Apply returns results but doesn't write)
	// The caller decides whether to write based on mode
}

func TestEngineDryRun(t *testing.T) {
	dir := t.TempDir()
	input := `resource "aws_s3_bucket" "test" {
  bucket = "my-bucket"
  acl    = "private"
}
`
	os.WriteFile(filepath.Join(dir, "main.tf"), []byte(input), 0644)

	reg := NewRegistry()
	m, _ := reg.Find("hashicorp/aws/v3-to-v4")

	results, err := Apply(dir, m)
	if err != nil {
		t.Fatal(err)
	}

	// Verify original file is untouched
	got, _ := os.ReadFile(filepath.Join(dir, "main.tf"))
	if string(got) != input {
		t.Fatal("Apply should not modify files; it returns results for the caller to write")
	}

	_ = results // caller uses these to decide what to write
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/command/migrate/...`
Expected: FAIL — `Apply` function not defined.

**Step 3: Implement the engine**

Create `internal/command/migrate/engine.go`:

```go
// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package migrate

import (
	"bytes"
	"os"
	"path/filepath"
)

// FileResult holds the before/after content for a single file after applying
// one sub-migration.
type FileResult struct {
	Filename string
	Before   []byte
	After    []byte
}

// SubMigrationResult holds the outcome of applying one sub-migration across
// all files in the directory.
type SubMigrationResult struct {
	SubMigration SubMigration
	Files        []FileResult // only files that changed
}

// Apply runs all sub-migrations in the given migration against .tf files in
// dir. It does NOT write any files — it returns results that the caller can
// inspect, diff, or write. Each sub-migration sees the output of the previous
// one (they chain).
func Apply(dir string, m Migration) ([]SubMigrationResult, error) {
	// Collect .tf files
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	type fileState struct {
		filename string
		content  []byte
	}

	var files []fileState
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".tf" {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		files = append(files, fileState{filename: e.Name(), content: content})
	}

	var results []SubMigrationResult

	for _, sm := range m.SubMigrations {
		smResult := SubMigrationResult{SubMigration: sm}

		for i, f := range files {
			after, err := sm.Apply(f.filename, f.content)
			if err != nil {
				return nil, err
			}

			if !bytes.Equal(f.content, after) {
				smResult.Files = append(smResult.Files, FileResult{
					Filename: f.filename,
					Before:   f.content,
					After:    after,
				})
				// Update file state for next sub-migration
				files[i].content = after
			}
		}

		if len(smResult.Files) > 0 {
			results = append(results, smResult)
		}
	}

	return results, nil
}

// WriteResults writes all changed files to disk.
func WriteResults(dir string, results []SubMigrationResult) error {
	// Collect the final state of each file across all sub-migrations
	finalState := make(map[string][]byte)
	for _, r := range results {
		for _, f := range r.Files {
			finalState[f.Filename] = f.After
		}
	}

	for filename, content := range finalState {
		if err := os.WriteFile(filepath.Join(dir, filename), content, 0644); err != nil {
			return err
		}
	}
	return nil
}
```

**Step 4: Run tests**

Run: `go test ./internal/command/migrate/...`
Expected: All PASS

**Step 5: Commit**

```
feat: add migration engine that applies sub-migrations to .tf files
```

---

### Task 4: Arguments parsing

**Files:**
- Create: `internal/command/arguments/migrate.go`

**Step 1: Write the arguments file**

Create `internal/command/arguments/migrate.go`:

```go
// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"strings"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// MigrateList represents the command-line arguments for "terraform migrate list".
type MigrateList struct {
	Detail   bool
	ViewType ViewType
}

// ParseMigrateList processes CLI arguments for the migrate list command.
func ParseMigrateList(args []string) (*MigrateList, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var jsonOutput bool

	ml := &MigrateList{}
	cmdFlags := defaultFlagSet("migrate list")
	cmdFlags.BoolVar(&ml.Detail, "detail", false, "detail")
	cmdFlags.BoolVar(&jsonOutput, "json", false, "json")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	args = cmdFlags.Args()
	if len(args) > 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Too many command line arguments",
			"The migrate list command expects no positional arguments.",
		))
	}

	switch {
	case jsonOutput:
		ml.ViewType = ViewJSON
	default:
		ml.ViewType = ViewHuman
	}

	return ml, diags
}

// MigrateApply represents the command-line arguments for "terraform migrate <id>".
type MigrateApply struct {
	// MigrationID is the fully qualified migration identifier
	// (namespace/provider/name).
	MigrationID string

	// DryRun shows what would change without modifying files.
	DryRun bool

	// Step enables interactive per-sub-migration approval.
	Step bool

	ViewType ViewType
}

// ParseMigrateApply processes CLI arguments for the migrate apply command.
func ParseMigrateApply(args []string) (*MigrateApply, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var jsonOutput bool

	ma := &MigrateApply{}
	cmdFlags := defaultFlagSet("migrate")
	cmdFlags.BoolVar(&ma.DryRun, "dry-run", false, "dry-run")
	cmdFlags.BoolVar(&ma.Step, "step", false, "step")
	cmdFlags.BoolVar(&jsonOutput, "json", false, "json")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	if ma.DryRun && ma.Step {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid flag combination",
			"The -dry-run and -step flags are mutually exclusive.",
		))
	}

	args = cmdFlags.Args()
	if len(args) != 1 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Expected migration ID",
			"Usage: terraform migrate <namespace/provider/name> [-dry-run] [-step]",
		))
	} else {
		ma.MigrationID = args[0]
		// Validate format: must have exactly 2 slashes
		parts := strings.SplitN(ma.MigrationID, "/", 4)
		if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid migration ID format",
				"Migration ID must be in the format namespace/provider/name (e.g. hashicorp/aws/v3-to-v4).",
			))
		}
	}

	switch {
	case jsonOutput:
		ma.ViewType = ViewJSON
	default:
		ma.ViewType = ViewHuman
	}

	return ma, diags
}
```

**Step 2: Run build to verify compilation**

Run: `go build ./internal/command/arguments/...`
Expected: PASS

**Step 3: Commit**

```
feat: add argument parsing for migrate list and migrate apply commands
```

---

### Task 5: Views

**Files:**
- Create: `internal/command/views/migrate.go`

**Step 1: Write the views file**

Create `internal/command/views/migrate.go`. This implements the `MigrateListView` and `MigrateApplyView` interfaces with Human and JSON variants.

The Human views handle:
- `MigrateListView.List()`: Grouped output with first 3 sub-migrations shown, "+N more" for the rest
- `MigrateApplyView.Progress()`: `✓ sub-migration-name (file1.tf, file2.tf)` lines
- `MigrateApplyView.Summary()`: `Applied N changes across M files.`
- `MigrateApplyView.DryRunSummary()`: `N changes would be applied across M files.`
- `MigrateApplyView.Diff()`: unified diff output
- `MigrateApplyView.StepPrompt()`: `[1/N] name: description` + diff + prompt
- `MigrateApplyView.Diagnostics()`: delegates to base View

```go
// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/migrate"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// MigrateList is the view interface for "terraform migrate list".
type MigrateList interface {
	// List renders the grouped migration list.
	List(migrations []migrate.Migration, results map[string][]migrate.SubMigrationResult, detail bool) int
	Diagnostics(diags tfdiags.Diagnostics)
}

// NewMigrateList creates a MigrateList view for the given view type.
func NewMigrateList(vt arguments.ViewType, view *View) MigrateList {
	switch vt {
	case arguments.ViewJSON:
		return &MigrateListJSON{view: view}
	case arguments.ViewHuman:
		return &MigrateListHuman{view: view}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

// MigrateApply is the view interface for "terraform migrate <id>".
type MigrateApply interface {
	// Applying prints the "Applying <id>..." header.
	Applying(id string)
	// Progress prints a single sub-migration result line.
	Progress(sm migrate.SubMigration, filenames []string)
	// Summary prints the final summary line.
	Summary(changes int, files int)
	// DryRunHeader prints the "Planning <id>..." header.
	DryRunHeader(id string)
	// Diff prints a unified diff for one file.
	Diff(filename string, before, after []byte)
	// DryRunSummary prints "N changes would be applied...".
	DryRunSummary(changes int, files int)
	// StepHeader prints the "[1/N] name: description" header.
	StepHeader(index, total int, sm migrate.SubMigration)
	// StepPrompt prints the prompt and reads the user's choice.
	// Returns 'y', 'n', or 'q'.
	StepPrompt() byte
	Diagnostics(diags tfdiags.Diagnostics)
}

// NewMigrateApply creates a MigrateApply view for the given view type.
func NewMigrateApply(vt arguments.ViewType, view *View) MigrateApply {
	switch vt {
	case arguments.ViewJSON:
		return &MigrateApplyJSON{view: view}
	case arguments.ViewHuman:
		return &MigrateApplyHuman{view: view}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}
```

Then implement `MigrateListHuman`, `MigrateListJSON`, `MigrateApplyHuman`, `MigrateApplyJSON` structs. The Human implementations use `v.view.streams` and `v.view.colorize` for colored output. The JSON implementations output structured JSON.

Key implementation details for `MigrateListHuman.List()`:
- Group migrations by `Namespace + "/" + Provider`
- For each group, show `<namespace>/<provider> (N migrations available):`
- For each migration: `  <name>    <description>    <files> files, <changes> changes`
- Show first 3 sub-migrations: `    - <name>    <description>`
- If more than 3: `    (+N more, use -detail to list all)`
- If 3 or fewer, show count: `    (N sub-migrations total)`

Key implementation details for `MigrateApplyHuman.StepPrompt()`:
- Print `Apply this change? [y]es / [n]o / [q]uit: `
- Read a single byte from `v.view.streams.Stdin.File`
- Return 'y', 'n', or 'q'

For diff output, use a simple line-by-line diff function or the `internal/command/format` package if available. For the prototype, a basic unified diff using `--- filename` / `+++ filename` headers with `-` and `+` prefixed lines is sufficient.

**Step 2: Run build**

Run: `go build ./internal/command/views/...`
Expected: PASS

**Step 3: Commit**

```
feat: add views for migrate list and migrate apply commands
```

---

### Task 6: Parent command (terraform migrate)

**Files:**
- Create: `internal/command/migrate_command.go`

**Step 1: Write the parent command**

Create `internal/command/migrate_command.go` following the `state_command.go` pattern:

```go
// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"strings"

	"github.com/hashicorp/cli"
)

// MigrateCommand is a Command implementation that just shows help for
// the subcommands nested below it.
type MigrateCommand struct {
	Meta
}

func (c *MigrateCommand) Run(args []string) int {
	return cli.RunResultHelp
}

func (c *MigrateCommand) Help() string {
	helpText := `
Usage: terraform [global options] migrate <subcommand> [options] [args]

  This command has subcommands for running source code migrations.

  Migrations update your Terraform configuration files to accommodate
  breaking changes in provider versions or Terraform Core updates.
  Migrations are published by providers and Terraform Core.

Subcommands:

  list    List available migrations for the current configuration
`
	return strings.TrimSpace(helpText)
}

func (c *MigrateCommand) Synopsis() string {
	return "Run source code migrations"
}
```

**Step 2: Run build**

Run: `go build ./internal/command/...`
Expected: PASS

**Step 3: Commit**

```
feat: add parent migrate command with help text
```

---

### Task 7: Migrate list command

**Files:**
- Create: `internal/command/migrate_list.go`
- Create: `internal/command/migrate_list_test.go`

**Step 1: Write the test**

Create `internal/command/migrate_list_test.go`:

```go
// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"strings"
	"testing"
)

func TestMigrateList(t *testing.T) {
	// Create a temp directory with .tf files that match migrations
	dir := t.TempDir()
	// Write test fixtures...
	testCopyDir(t, testFixturePath("migrate-list"), dir)
	defer testChdir(t, dir)()

	view, done := testView(t)
	c := &MigrateListCommand{
		Meta: Meta{
			View: view,
		},
	}

	code := c.Run(nil)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := done(t)
	got := output.Stdout()

	// Verify key output elements
	if !strings.Contains(got, "hashicorp/aws") {
		t.Errorf("expected output to contain 'hashicorp/aws', got:\n%s", got)
	}
	if !strings.Contains(got, "v3-to-v4") {
		t.Errorf("expected output to contain 'v3-to-v4', got:\n%s", got)
	}
}
```

Also create `testdata/migrate-list/` fixture directory with sample `.tf` files that contain patterns matching the sample migrations.

**Step 2: Run test to verify it fails**

Run: `go test -run TestMigrateList ./internal/command/...`
Expected: FAIL — `MigrateListCommand` not defined.

**Step 3: Implement the command**

Create `internal/command/migrate_list.go`:

```go
// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/migrate"
	"github.com/hashicorp/terraform/internal/command/views"
)

// MigrateListCommand lists available migrations for the current configuration.
type MigrateListCommand struct {
	Meta
}

func (c *MigrateListCommand) Run(rawArgs []string) int {
	rawArgs = c.Meta.process(rawArgs)
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	args, diags := arguments.ParseMigrateList(rawArgs)
	if diags.HasErrors() {
		c.View.Diagnostics(diags)
		c.View.HelpPrompt("migrate list")
		return 1
	}

	view := views.NewMigrateList(args.ViewType, c.View)

	// Get the working directory
	dir, err := c.WorkingDir.Dir()
	if err != nil {
		// fall back to "."
		dir = "."
	}

	// Scan for applicable migrations
	registry := migrate.NewRegistry()
	all := registry.All()

	// Run engine in dry-run mode to find which migrations have matches
	resultsByMigration := make(map[string][]migrate.SubMigrationResult)
	for _, m := range all {
		results, err := migrate.Apply(dir, m)
		if err != nil {
			diags = diags.Append(err)
			view.Diagnostics(diags)
			return 1
		}
		if len(results) > 0 {
			resultsByMigration[m.ID()] = results
		}
	}

	return view.List(all, resultsByMigration, args.Detail)
}

func (c *MigrateListCommand) Help() string {
	return migrateListHelp
}

func (c *MigrateListCommand) Synopsis() string {
	return "List available migrations"
}

const migrateListHelp = `
Usage: terraform [global options] migrate list [options]

  Lists available migrations for the current Terraform configuration.
  Shows migrations grouped by provider, with a preview of sub-migrations.

Options:

  -detail    Show all sub-migrations (default: show first 3 per migration)

  -json      Output in machine-readable JSON format.
`
```

Note: The working directory resolution may need adjustment based on how `Meta.WorkingDir` works. For the prototype, falling back to `"."` is acceptable.

**Step 4: Run tests**

Run: `go test -run TestMigrateList ./internal/command/...`
Expected: PASS

**Step 5: Commit**

```
feat: add migrate list command
```

---

### Task 8: Migrate apply command (default + dry-run modes)

**Files:**
- Create: `internal/command/migrate_apply.go`
- Create: `internal/command/migrate_apply_test.go`
- Create: `testdata/migrate-apply/` fixture directory

**Step 1: Write tests**

Create `internal/command/migrate_apply_test.go` with tests for:

- **Default mode**: Run migrate, verify files are modified, verify progress output
- **Dry-run mode**: Run with `-dry-run`, verify files are NOT modified, verify diff output and summary

```go
// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrateApply_default(t *testing.T) {
	dir := t.TempDir()
	testCopyDir(t, testFixturePath("migrate-apply"), dir)
	defer testChdir(t, dir)()

	view, done := testView(t)
	c := &MigrateApplyCommand{
		Meta: Meta{
			View: view,
		},
	}

	code := c.Run([]string{"hashicorp/aws/v3-to-v4"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := done(t)
	got := output.Stdout()

	if !strings.Contains(got, "Applying hashicorp/aws/v3-to-v4") {
		t.Errorf("expected 'Applying' header, got:\n%s", got)
	}
	if !strings.Contains(got, "✓") {
		t.Errorf("expected checkmark in output, got:\n%s", got)
	}

	// Verify file was actually modified
	content, err := os.ReadFile(filepath.Join(dir, "main.tf"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "acl") {
		t.Error("expected acl to be extracted from aws_s3_bucket")
	}
	if !strings.Contains(string(content), "aws_s3_bucket_acl") {
		t.Error("expected aws_s3_bucket_acl resource to be created")
	}
}

func TestMigrateApply_dryRun(t *testing.T) {
	dir := t.TempDir()
	testCopyDir(t, testFixturePath("migrate-apply"), dir)
	defer testChdir(t, dir)()

	// Read original content
	original, _ := os.ReadFile(filepath.Join(dir, "main.tf"))

	view, done := testView(t)
	c := &MigrateApplyCommand{
		Meta: Meta{
			View: view,
		},
	}

	code := c.Run([]string{"-dry-run", "hashicorp/aws/v3-to-v4"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := done(t)
	got := output.Stdout()

	if !strings.Contains(got, "Planning") {
		t.Errorf("expected 'Planning' header, got:\n%s", got)
	}
	if !strings.Contains(got, "would be applied") {
		t.Errorf("expected dry-run summary, got:\n%s", got)
	}

	// Verify file was NOT modified
	content, _ := os.ReadFile(filepath.Join(dir, "main.tf"))
	if string(content) != string(original) {
		t.Error("dry-run should not modify files")
	}
}
```

**Step 2: Create test fixtures**

Create `testdata/migrate-apply/main.tf`:

```hcl
resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
  acl    = "private"
}
```

**Step 3: Run tests to verify they fail**

Run: `go test -run TestMigrateApply ./internal/command/...`
Expected: FAIL — `MigrateApplyCommand` not defined.

**Step 4: Implement the command**

Create `internal/command/migrate_apply.go`:

```go
// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/migrate"
	"github.com/hashicorp/terraform/internal/command/views"
)

// MigrateApplyCommand runs a migration set against the current configuration.
type MigrateApplyCommand struct {
	Meta
}

func (c *MigrateApplyCommand) Run(rawArgs []string) int {
	rawArgs = c.Meta.process(rawArgs)
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	args, diags := arguments.ParseMigrateApply(rawArgs)
	if diags.HasErrors() {
		c.View.Diagnostics(diags)
		return 1
	}

	view := views.NewMigrateApply(args.ViewType, c.View)

	dir, err := c.WorkingDir.Dir()
	if err != nil {
		dir = "."
	}

	registry := migrate.NewRegistry()
	m, err := registry.Find(args.MigrationID)
	if err != nil {
		diags = diags.Append(err)
		view.Diagnostics(diags)
		return 1
	}

	results, err := migrate.Apply(dir, m)
	if err != nil {
		diags = diags.Append(err)
		view.Diagnostics(diags)
		return 1
	}

	if len(results) == 0 {
		view.Summary(0, 0)
		return 0
	}

	switch {
	case args.DryRun:
		return c.dryRun(view, results)
	case args.Step:
		return c.step(view, dir, m, results)
	default:
		return c.apply(view, dir, results)
	}
}

func (c *MigrateApplyCommand) apply(view views.MigrateApply, dir string, results []migrate.SubMigrationResult) int {
	// ... show progress, write files, show summary
}

func (c *MigrateApplyCommand) dryRun(view views.MigrateApply, results []migrate.SubMigrationResult) int {
	// ... show diffs, show dry-run summary, don't write
}

func (c *MigrateApplyCommand) step(view views.MigrateApply, dir string, m migrate.Migration, results []migrate.SubMigrationResult) int {
	// ... for each result, show diff, prompt, apply or skip
}
```

Fill in the `apply`, `dryRun`, and `step` methods. The `step` method needs special handling — it must apply sub-migrations one at a time and re-run subsequent ones if a step is skipped (since later steps may depend on earlier ones' output).

**Step 5: Run tests**

Run: `go test -run TestMigrateApply ./internal/command/...`
Expected: PASS

**Step 6: Commit**

```
feat: add migrate apply command with default and dry-run modes
```

---

### Task 9: Step mode (interactive approval)

**Files:**
- Modify: `internal/command/migrate_apply.go` (fill in `step` method)
- Add to: `internal/command/migrate_apply_test.go`

**Step 1: Write step mode test**

Add to `migrate_apply_test.go`. Since step mode reads stdin, use `terminal.StreamsForTesting` with piped input:

```go
func TestMigrateApply_step(t *testing.T) {
	dir := t.TempDir()
	testCopyDir(t, testFixturePath("migrate-apply"), dir)
	defer testChdir(t, dir)()

	// Simulate user typing "y\n" for each prompt
	streams, done := terminal.StreamsForTesting(t)
	streams.Stdin = // pipe with "y\n" input

	view := views.NewView(streams)
	c := &MigrateApplyCommand{
		Meta: Meta{
			View:    view,
			Streams: streams,
		},
	}

	code := c.Run([]string{"-step", "hashicorp/aws/v3-to-v4"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := done(t)
	got := output.Stdout()

	if !strings.Contains(got, "[1/") {
		t.Errorf("expected step header, got:\n%s", got)
	}
	if !strings.Contains(got, "Apply this change?") {
		t.Errorf("expected prompt, got:\n%s", got)
	}
}

func TestMigrateApply_stepQuit(t *testing.T) {
	// Same setup but pipe "q\n" — should exit early
	// Verify that no files were modified
}
```

Note: The exact mechanism for piping stdin in tests needs to match how `terminal.StreamsForTesting` works. Look at existing tests that use stdin input for the pattern.

**Step 2: Run test to verify it fails**

Run: `go test -run TestMigrateApply_step ./internal/command/...`
Expected: FAIL

**Step 3: Implement step method**

Fill in the `step` method on `MigrateApplyCommand`. For each sub-migration result:
1. Call `view.StepHeader(index, total, sm)`
2. For each changed file, call `view.Diff(filename, before, after)`
3. Call `view.StepPrompt()` and handle the response
4. If 'y': write the changed files for this step
5. If 'n': skip (but need to re-compute subsequent results without this step)
6. If 'q': exit immediately without further changes

**Step 4: Run tests**

Run: `go test -run TestMigrateApply_step ./internal/command/...`
Expected: PASS

**Step 5: Commit**

```
feat: add interactive step mode to migrate apply command
```

---

### Task 10: Command registration and wiring

**Files:**
- Modify: `commands.go` (add migrate commands)

**Step 1: Register commands**

Add to the `Commands` map in `commands.go` around line 388 (near "state"):

```go
"migrate": func() (cli.Command, error) {
    return &command.MigrateCommand{
        Meta: meta,
    }, nil
},

"migrate list": func() (cli.Command, error) {
    return &command.MigrateListCommand{
        Meta: meta,
    }, nil
},
```

The `terraform migrate hashicorp/aws/v3-to-v4` case is trickier — the CLI framework uses the first word after `migrate` as a potential subcommand. Since `hashicorp/aws/v3-to-v4` is not a registered subcommand, we need the parent `MigrateCommand` to detect positional args and delegate to `MigrateApplyCommand`.

Alternative approach: register `MigrateCommand.Run()` to check if the first arg looks like a migration ID (contains `/`) and if so, create and run a `MigrateApplyCommand` inline. This is simpler than trying to make the CLI framework handle this.

Update `MigrateCommand.Run()`:

```go
func (c *MigrateCommand) Run(args []string) int {
	// If the first arg looks like a migration ID (contains /),
	// delegate to MigrateApplyCommand
	if len(args) > 0 && strings.Contains(args[0], "/") {
		apply := &MigrateApplyCommand{Meta: c.Meta}
		return apply.Run(args)
	}
	return cli.RunResultHelp
}
```

**Step 2: Build and verify**

Run: `go build ./...`
Expected: PASS

**Step 3: Manual smoke test**

Run: Create a temp directory with test `.tf` files, then:
- `go run . migrate list`
- `go run . migrate hashicorp/aws/v3-to-v4 -dry-run`
- `go run . migrate hashicorp/aws/v3-to-v4`

**Step 4: Commit**

```
feat: register migrate commands in CLI
```

---

### Task 11: Test fixtures and integration tests

**Files:**
- Create: `testdata/migrate-list/main.tf`
- Create: `testdata/migrate-list/s3.tf`
- Create: `testdata/migrate-apply/main.tf`
- Finalize: `internal/command/migrate_list_test.go`
- Finalize: `internal/command/migrate_apply_test.go`

**Step 1: Create comprehensive test fixtures**

`testdata/migrate-list/main.tf` — should have resources matching multiple migration sets:

```hcl
terraform {
  required_providers {
    aws = "~> 3.0"
  }
}

resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
  acl    = "private"

  cors_rule {
    allowed_methods = ["GET"]
    allowed_origins = ["*"]
  }

  logging {
    target_bucket = "log-bucket"
  }
}
```

`testdata/migrate-apply/main.tf` — simpler fixture for apply tests.

**Step 2: Run full test suite**

Run: `go test ./internal/command/migrate/... ./internal/command/...`
Expected: All PASS

**Step 3: Commit**

```
feat: add test fixtures and finalize integration tests
```

---

### Task 12: Polish and diff output

**Files:**
- Create or modify: `internal/command/migrate/diff.go` (if needed)
- Polish: `internal/command/views/migrate.go`

**Step 1: Implement unified diff**

For the prototype, implement a simple unified diff function that compares before/after line by line. If there's an existing diff utility in the codebase (check `internal/command/format/`), use that. Otherwise, write a minimal one.

**Step 2: Polish colored output**

Ensure the Human views use `v.view.colorize` for:
- Green `✓` checkmarks
- Red `-` lines in diffs
- Green `+` lines in diffs
- Bold migration names and headers

Use colorstring syntax: `[green]✓[reset]`, `[red]-line[reset]`, etc.

**Step 3: Run full test suite**

Run: `go test ./internal/command/migrate/... ./internal/command/...`
Expected: All PASS

**Step 4: Commit**

```
feat: add unified diff output and colored formatting
```
