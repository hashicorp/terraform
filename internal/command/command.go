// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"fmt"
	"os"
	"runtime"
)

// Set to true when we're testing
var test bool = false

// DefaultDataDir is the default directory for storing local data.
const DefaultDataDir = ".mnptu"

// PluginPathFile is the name of the file in the data dir which stores the list
// of directories supplied by the user with the `-plugin-dir` flag during init.
const PluginPathFile = "plugin_path"

// pluginMachineName is the directory name used in new plugin paths.
const pluginMachineName = runtime.GOOS + "_" + runtime.GOARCH

// DefaultPluginVendorDir is the location in the config directory to look for
// user-added plugin binaries. mnptu only reads from this path if it
// exists, it is never created by mnptu.
const DefaultPluginVendorDir = "mnptu.d/plugins/" + pluginMachineName

// DefaultStateFilename is the default filename used for the state file.
const DefaultStateFilename = "mnptu.tfstate"

// DefaultVarsFilename is the default filename used for vars
const DefaultVarsFilename = "mnptu.tfvars"

// DefaultBackupExtension is added to the state file to form the path
const DefaultBackupExtension = ".backup"

// DefaultParallelism is the limit mnptu places on total parallel
// operations as it walks the dependency graph.
const DefaultParallelism = 10

// ErrUnsupportedLocalOp is the common error message shown for operations
// that require a backend.Local.
const ErrUnsupportedLocalOp = `The configured backend doesn't support this operation.

The "backend" in mnptu defines how mnptu operates. The default
backend performs all operations locally on your machine. Your configuration
is configured to use a non-local backend. This backend doesn't support this
operation.
`

// ModulePath returns the path to the root module and validates CLI arguments.
//
// This centralizes the logic for any commands that previously accepted
// a module path via CLI arguments. This will error if any extraneous arguments
// are given and suggest using the -chdir flag instead.
//
// If your command accepts more than one arg, then change the slice bounds
// to pass validation.
func ModulePath(args []string) (string, error) {
	// TODO: test

	if len(args) > 0 {
		return "", fmt.Errorf("Too many command line arguments. Did you mean to use -chdir?")
	}

	path, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("Error getting pwd: %s", err)
	}

	return path, nil
}
