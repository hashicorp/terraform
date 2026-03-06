// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"strings"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/migrate"
	"github.com/hashicorp/terraform/internal/command/views"
)

// MigrateListCommand is a Command implementation that lists available
// migrations for the current working directory.
type MigrateListCommand struct {
	Meta
}

func (c *MigrateListCommand) Run(rawArgs []string) int {
	// Process global flags
	rawArgs = c.Meta.process(rawArgs)
	common, rawArgs := arguments.ParseView(rawArgs)
	if !c.Color {
		common.NoColor = true
	}
	c.View.Configure(common)

	// Parse command flags
	args, diags := arguments.ParseMigrateList(rawArgs)
	if diags.HasErrors() {
		c.View.Diagnostics(diags)
		c.View.HelpPrompt("migrate list")
		return 1
	}

	// Create view
	view := views.NewMigrateList(args.ViewType, c.View)

	// Working directory
	dir := "."

	// Get all migrations from registry
	registry := migrate.NewRegistry()
	all := registry.All()

	// Run engine to find matches (dry-run style)
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

	// Render
	return view.List(all, resultsByMigration, args.Detail)
}

func (c *MigrateListCommand) Help() string {
	helpText := `
Usage: terraform [global options] migrate list [options]

  Lists available migrations for the Terraform configuration in the
  current working directory.

Options:

  -detail    Show all sub-migrations, not just a summary.

  -json      Output the migration list in a machine-readable JSON format.
`
	return strings.TrimSpace(helpText)
}

func (c *MigrateListCommand) Synopsis() string {
	return "List available migrations"
}
