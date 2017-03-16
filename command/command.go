package command

import (
	"fmt"
	"log"
	"os"

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

// DefaultParallelism is the limit Terraform places on total parallel
// operations as it walks the dependency graph.
const DefaultParallelism = 10

// ErrUnsupportedLocalOp is the common error message shown for operations
// that require a backend.Local.
const ErrUnsupportedLocalOp = `The configured backend doesn't support this operation.

The "backend" in Terraform defines how Terraform operates. The default
backend performs all operations locally on your machine. Your configuration
is configured to use a non-local backend. This backend doesn't support this
operation.

If you want to use the state from the backend but force all other data
(configuration, variables, etc.) to come locally, you can force local
behavior with the "-local" flag.
`

// ModulePath returns the path to the root module from the CLI args.
//
// This centralizes the logic for any commands that expect a module path
// on their CLI args. This will verify that only one argument is given
// and that it is a path to configuration.
//
// If your command accepts more than one arg, then change the slice bounds
// to pass validation.
func ModulePath(args []string) (string, error) {
	// TODO: test

	if len(args) > 1 {
		return "", fmt.Errorf("Too many command line arguments. Configuration path expected.")
	}

	if len(args) == 0 {
		path, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("Error getting pwd: %s", err)
		}

		return path, nil
	}

	return args[0], nil
}

func validateContext(ctx *terraform.Context, ui cli.Ui) bool {
	log.Println("[INFO] Validating the context...")
	ws, es := ctx.Validate()
	log.Printf("[INFO] Validation result: %d warnings, %d errors", len(ws), len(es))

	if len(ws) > 0 || len(es) > 0 {
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
