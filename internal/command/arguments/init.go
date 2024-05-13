// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"time"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Init represents the command-line arguments for the init command.
type Init struct {
	// FromModule identifies the module to copy into the target directory before init.
	FromModule string

	// Lockfile specifies a dependency lockfile mode.
	Lockfile string

	// TestDirectory is the directory containing any test files that should be
	// validated alongside the main configuration. Should be relative to the
	// Path.
	TestsDirectory string

	// ViewType specifies which init format to use: human or JSON.
	ViewType ViewType

	// Backend specifies whether to disable backend or HCP Terraform initialization.
	Backend bool

	// Cloud specifies whether to disable backend or HCP Terraform initialization.
	Cloud bool

	// Get specifies whether to disable downloading modules for this configuration
	Get bool

	// ForceInitCopy specifies whether to suppress prompts about copying state data.
	ForceInitCopy bool

	// StateLock specifies whether hold a state lock during backend migration.
	StateLock bool

	// StateLockTimeout specifies the duration to wait for a state lock.
	StateLockTimeout time.Duration

	// Reconfigure specifies whether to disregard any existing configuration, preventing migration of any existing state
	Reconfigure bool

	// MigrateState specifies whether to attempt to copy existing state to the new backend
	MigrateState bool

	// Upgrade specifies whether to upgrade modules and plugins as part of their respective installation steps
	Upgrade bool

	// Json specifies whether to output in JSON format
	Json bool

	// IgnoreRemoteVersion specifies whether to ignore remote and local Terraform versions compatibility
	IgnoreRemoteVersion bool

	BackendConfig FlagNameValueSlice

	Vars *Vars

	// InputEnabled is used to disable interactive input for unspecified
	// variable and backend config values. Default is true.
	InputEnabled bool

	TargetFlags []string

	CompactWarnings bool

	PluginPath FlagStringSlice

	Args []string
}

// ParseInit processes CLI arguments, returning an Init value and errors.
// If errors are encountered, an Init value is still returned representing
// the best effort interpretation of the arguments.
func ParseInit(args []string) (*Init, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	init := &Init{
		Vars: &Vars{},
	}
	init.BackendConfig = NewFlagNameValueSlice("-backend-config")

	cmdFlags := extendedFlagSet("init", nil, nil, init.Vars)

	cmdFlags.Var((*FlagStringSlice)(&init.TargetFlags), "target", "resource to target")
	cmdFlags.BoolVar(&init.InputEnabled, "input", true, "input")
	cmdFlags.BoolVar(&init.CompactWarnings, "compact-warnings", false, "use compact warnings")
	cmdFlags.BoolVar(&init.Backend, "backend", true, "")
	cmdFlags.BoolVar(&init.Cloud, "cloud", true, "")
	cmdFlags.StringVar(&init.FromModule, "from-module", "", "copy the source of the given module into the directory before init")
	cmdFlags.BoolVar(&init.Get, "get", true, "")
	cmdFlags.BoolVar(&init.ForceInitCopy, "force-copy", false, "suppress prompts about copying state data")
	cmdFlags.BoolVar(&init.StateLock, "lock", true, "lock state")
	cmdFlags.DurationVar(&init.StateLockTimeout, "lock-timeout", 0, "lock timeout")
	cmdFlags.BoolVar(&init.Reconfigure, "reconfigure", false, "reconfigure")
	cmdFlags.BoolVar(&init.MigrateState, "migrate-state", false, "migrate state")
	cmdFlags.BoolVar(&init.Upgrade, "upgrade", false, "")
	cmdFlags.StringVar(&init.Lockfile, "lockfile", "", "Set a dependency lockfile mode")
	cmdFlags.BoolVar(&init.IgnoreRemoteVersion, "ignore-remote-version", false, "continue even if remote and local Terraform versions are incompatible")
	cmdFlags.StringVar(&init.TestsDirectory, "test-directory", "tests", "test-directory")
	cmdFlags.BoolVar(&init.Json, "json", false, "json")
	cmdFlags.Var(&init.BackendConfig, "backend-config", "")
	cmdFlags.Var(&init.PluginPath, "plugin-dir", "plugin directory")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	if init.MigrateState && init.Json {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"The -migrate-state and -json options are mutually-exclusive",
			"Terraform cannot ask for interactive approval when -json is set. To use the -migrate-state option, disable the -json option.",
		))
	}

	if init.MigrateState && init.Reconfigure {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid init options",
			"The -migrate-state and -reconfigure options are mutually-exclusive.",
		))
	}

	init.Args = cmdFlags.Args()

	backendFlagSet := FlagIsSet(cmdFlags, "backend")
	cloudFlagSet := FlagIsSet(cmdFlags, "cloud")

	if backendFlagSet && cloudFlagSet {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid init options",
			"The -backend and -cloud options are aliases of one another and mutually-exclusive in their use",
		))
	} else if backendFlagSet {
		init.Cloud = init.Backend
	} else if cloudFlagSet {
		init.Backend = init.Cloud
	}

	switch {
	case init.Json:
		init.ViewType = ViewJSON
	default:
		init.ViewType = ViewHuman
	}

	return init, diags
}
