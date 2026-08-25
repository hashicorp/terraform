// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"flag"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// checkStateMisplacedOptions returns an error diagnostic for each of the given
// positional arguments that names one of the options defined in cmdFlags.
//
// Option parsing stops at the first positional argument, so an option written
// after an address is retained as an additional positional argument instead of
// being applied. Without this check the state subcommands go on to interpret
// that argument as an address, which fails with a confusing diagnostic such as
// "Variable name required" that says nothing about the misplaced option.
//
// Only arguments naming an option that this command actually defines are
// reported, because a positional argument may legitimately begin with a dash.
// An invalid provider address such as "-/-/google" should still be reported as
// an invalid provider address rather than as a misplaced option.
//
// commandName is the subcommand as the user would type it, such as "state rm",
// and argsUsage describes that subcommand's positional arguments, such as
// "ADDRESS...", so that we can show the corrected ordering.
func checkStateMisplacedOptions(cmdFlags *flag.FlagSet, commandName, argsUsage string, args []string) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	for _, arg := range args {
		name, ok := optionName(arg)
		if !ok || cmdFlags.Lookup(name) == nil {
			continue
		}
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Option specified after arguments",
			fmt.Sprintf(
				"Terraform stops interpreting command-line options at the first argument, so %q was treated as an argument to \"terraform %s\" rather than as an option.\n\nSpecify options before the arguments, like this:\n    terraform %s %s %s",
				arg, commandName, commandName, arg, argsUsage,
			),
		))
	}
	return diags
}

// optionName returns the name of the option that the given argument would have
// set had it appeared before any positional arguments, along with whether the
// argument has the form of an option at all.
//
// A bare "-" is not an option, because some commands use it to mean standard
// input.
func optionName(arg string) (string, bool) {
	name, ok := strings.CutPrefix(arg, "-")
	if !ok {
		return "", false
	}
	// Terraform accepts both "-name" and "--name" for the same option.
	name = strings.TrimPrefix(name, "-")

	// "-name=value" sets the option called "name".
	name, _, _ = strings.Cut(name, "=")

	if name == "" {
		return "", false
	}
	return name, true
}
