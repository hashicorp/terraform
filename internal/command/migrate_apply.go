// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/migrate"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// MigrateApplyCommand is a Command implementation that applies a specific
// migration to the Terraform configuration in the current working directory.
type MigrateApplyCommand struct {
	Meta
}

func (c *MigrateApplyCommand) Run(rawArgs []string) int {
	rawArgs = c.Meta.process(rawArgs)
	common, rawArgs := arguments.ParseView(rawArgs)
	// process() may have already consumed -no-color; propagate to view
	if !c.Color {
		common.NoColor = true
	}
	c.View.Configure(common)

	args, diags := arguments.ParseMigrateApply(rawArgs)
	if diags.HasErrors() {
		c.View.Diagnostics(diags)
		return 1
	}

	view := views.NewMigrateApply(args.ViewType, c.View)
	dir := "."

	// Find migration
	registry := migrate.NewRegistry()
	m, err := registry.Find(args.MigrationID)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Migration not found",
			err.Error(),
		))
		view.Diagnostics(diags)
		return 1
	}

	// Run engine
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
		return c.dryRun(view, args.MigrationID, results)
	case args.Step:
		return c.step(view, dir, m, results)
	default:
		return c.apply(view, dir, args.MigrationID, results)
	}
}

func (c *MigrateApplyCommand) apply(view views.MigrateApply, dir, id string, results []migrate.SubMigrationResult) int {
	view.Applying(id)

	// Write all results
	if err := migrate.WriteResults(dir, results); err != nil {
		var diags tfdiags.Diagnostics
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to write migration results",
			err.Error(),
		))
		view.Diagnostics(diags)
		return 1
	}

	// Show progress for each sub-migration
	totalChanges := 0
	allFiles := map[string]bool{}
	for _, r := range results {
		var filenames []string
		for _, f := range r.Files {
			filenames = append(filenames, f.Filename)
			allFiles[f.Filename] = true
		}
		totalChanges++
		view.Progress(r.SubMigration, filenames)
	}

	view.Summary(totalChanges, len(allFiles))
	return 0
}

func (c *MigrateApplyCommand) dryRun(view views.MigrateApply, id string, results []migrate.SubMigrationResult) int {
	view.DryRunHeader(id)

	totalChanges := 0
	allFiles := map[string]bool{}

	// Build first-seen (before) and last-seen (after) per filename
	firstBefore := map[string][]byte{}
	lastAfter := map[string][]byte{}

	for _, r := range results {
		totalChanges++
		for _, f := range r.Files {
			if _, seen := firstBefore[f.Filename]; !seen {
				firstBefore[f.Filename] = f.Before
			}
			lastAfter[f.Filename] = f.After
			allFiles[f.Filename] = true
		}
	}

	for filename := range allFiles {
		view.Diff(filename, firstBefore[filename], lastAfter[filename])
	}

	view.DryRunSummary(totalChanges, len(allFiles))
	return 0
}

func (c *MigrateApplyCommand) step(view views.MigrateApply, dir string, _ migrate.Migration, results []migrate.SubMigrationResult) int {
	totalChanges := 0
	allFiles := map[string]bool{}

	for i, r := range results {
		view.StepHeader(i+1, len(results), r.SubMigration)

		// Show diff for each file in this sub-migration
		for _, f := range r.Files {
			view.Diff(f.Filename, f.Before, f.After)
		}

		choice := view.StepPrompt(c.Streams)
		switch choice {
		case 'y':
			// Write just this sub-migration's files
			for _, f := range r.Files {
				if err := os.WriteFile(filepath.Join(dir, f.Filename), f.After, 0644); err != nil {
					var diags tfdiags.Diagnostics
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Failed to write file",
						err.Error(),
					))
					view.Diagnostics(diags)
					return 1
				}
				allFiles[f.Filename] = true
			}
			totalChanges++
		case 'n':
			// Skip this sub-migration
			continue
		case 'q':
			// Quit early
			view.Summary(totalChanges, len(allFiles))
			return 0
		}
	}

	view.Summary(totalChanges, len(allFiles))
	return 0
}

func (c *MigrateApplyCommand) Help() string {
	helpText := `
Usage: terraform [global options] migrate <migration-id> [options]

  Applies the specified migration to the Terraform configuration in the
  current working directory. The migration ID is in the format
  namespace/provider/name (e.g. hashicorp/aws/v3-to-v4).

Options:

  -dry-run   Show what changes would be made without modifying any files.

  -step      Apply the migration one sub-migration at a time, prompting
             before each step.

  -json      Output in a machine-readable JSON format.
`
	return strings.TrimSpace(helpText)
}

func (c *MigrateApplyCommand) Synopsis() string {
	return "Apply a specific migration"
}
