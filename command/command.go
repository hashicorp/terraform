package command

import (
	"fmt"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

// Set to true when we're testing
var test bool = false

// DefaultDataDir is the default directory for storing local data.
const DefaultDataDir = ".terraform"

// DefaultStateFilename is the default filename used for the state file.
const DefaultStateFilename = "terraform.tfstate"

// DefaultVarsFilename is the default filename used for vars
const DefaultVarsFilename = "terraform.tfvars"

// DefaultBackupExtension is added to the state file to form the path
const DefaultBackupExtension = ".backup"

// DefaultDataDirectory is the directory where local state is stored
// by default.
const DefaultDataDirectory = ".terraform"

// DefaultParallelism is the limit Terraform places on total parallel
// operations as it walks the dependency graph.
const DefaultParallelism = 10

func validateContext(ctx *terraform.Context, ui cli.Ui) bool {
	if ws, es := ctx.Validate(); len(ws) > 0 || len(es) > 0 {
		ui.Output(
			"There are warnings and/or errors related to your configuration. Please\n" +
				"fix these before continuing.\n")

		if len(ws) > 0 {
			ui.Warn("Warnings:\n")
			for _, w := range ws {
				ui.Warn(fmt.Sprintf("  * %s", w))
			}

			if len(es) > 0 {
				ui.Output("")
			}
		}

		if len(es) > 0 {
			ui.Error("Errors:\n")
			for _, e := range es {
				ui.Error(fmt.Sprintf("  * %s", e))
			}
			return false
		} else {
			ui.Warn(fmt.Sprintf("\n"+
				"No errors found. Continuing with %d warning(s).\n", len(ws)))
			return true
		}
	}

	return true
}
