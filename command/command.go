package command

import (
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/hashicorp/terraform/terraform"
)

// Set to true when we're testing
var test bool = false

// DefaultDataDir is the default directory for storing local data.
const DefaultDataDir = ".terraform"

// PluginPathFile is the name of the file in the data dir which stores the list
// of directories supplied by the user with the `-plugin-dir` flag during init.
const PluginPathFile = "plugin_path"

// pluginMachineName is the directory name used in new plugin paths.
const pluginMachineName = runtime.GOOS + "_" + runtime.GOARCH

// DefaultPluginVendorDir is the location in the config directory to look for
// user-added plugin binaries. Terraform only reads from this path if it
// exists, it is never created by terraform.
const DefaultPluginVendorDir = "terraform.d/plugins/" + pluginMachineName

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

func (m *Meta) validateContext(ctx *terraform.Context) bool {
	log.Println("[INFO] Validating the context...")
	diags := ctx.Validate()
	log.Printf("[INFO] Validation result: %d diagnostics", len(diags))

	if len(diags) > 0 {
		m.Ui.Output(
			"There are warnings and/or errors related to your configuration. Please\n" +
				"fix these before continuing.\n")

		m.showDiagnostics(diags)
	}

	return !diags.HasErrors()
}
