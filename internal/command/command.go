// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"runtime"
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

// DefaultStatePersistInterval is the default interval a backend should persist
// Terraform state, if applicable. Backends can set their own custom defaults.
const DefaultStatePersistInterval = 20

// DefaultBackupExtension is added to the state file to form the path
const DefaultBackupExtension = ".backup"

// DefaultParallelism is the limit Terraform places on total parallel
// operations as it walks the dependency graph.
const DefaultParallelism = 10

// ErrUnsupportedLocalOp is the common error message shown for operations
// that require a backendrun.Local.
const ErrUnsupportedLocalOp = `The configured backend doesn't support this operation.

The "backend" in Terraform defines how Terraform operates. The default
backend performs all operations locally on your machine. Your configuration
is configured to use a non-local backend. This backend doesn't support this
operation.
`
