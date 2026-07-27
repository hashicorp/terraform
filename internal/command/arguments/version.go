// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Version represents the command-line arguments for the version command.
type Version struct {
	// ViewType specifies which output format to use: human, JSON, or "raw".
	ViewType ViewType
}

func ParseVersion(args []string) (*Version, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	var jsonOutput bool
	cmdFlags := defaultFlagSet("version")
	cmdFlags.BoolVar(&jsonOutput, "json", false, "produce JSON output")

	// Enable but ignore the global version flags. In main.go, if any of the
	// arguments are -v, -version, or --version, the version command will be called
	// with the rest of the arguments, so we need to be able to cope with
	// those.
	//
	// If the user runs `terraform init -v` then the `-v` flag is never parsed,
	// however if the user runs `terraform -v` or `terraform version -v` then the flag IS
	// parsed here.
	cmdFlags.Bool("v", true, "version")
	cmdFlags.Bool("version", true, "version")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	// We purposefully ignore any remaining arguments (flags or positional args) instead of checking them and returning an error.
	// This is because in main.go the CLI re-routes all commands run with a `-version` flag to the version command (see above).
	// This enables `terraform -version`, `terraform init -version`, `terraform plan -version` etc. to all work the same as `terraform version`.
	// All arguments (including the original non-"version" command) are passed to the version command, so we ignore them here.
	//
	// For example if a user runs `terraform init -version -upgrade -get=false`, then it's turned into a `terraform version init -version -upgrade -get=false` command.
	// If we used cmdFlags.Args() to check for remaining arguments in that example there would be extra arguments ["init", "-version", "-upgrade", "-get=false"].
	_ = cmdFlags.Args()

	var viewType ViewType
	if jsonOutput {
		viewType = ViewJSON
	} else {
		viewType = ViewHuman
	}

	return &Version{
		ViewType: viewType,
	}, diags
}
