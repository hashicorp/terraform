// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import "github.com/hashicorp/terraform/internal/tfdiags"

type Stacks struct {
	PluginCacheDirOverride string

	// Positional arguments after flags are kept and passed to the stacks plugin.
	Args []string
}

func ParseStacks(args []string) (*Stacks, tfdiags.Diagnostics) {
	ret := &Stacks{
		Args: []string{}, // Ensure non-nil value for downstream code in `stacks`.
	}

	cmdFlags := defaultFlagSet("stacks")
	cmdFlags.StringVar(&ret.PluginCacheDirOverride, "plugin-cache-dir", "", "plugin cache directory")

	// We choose to ignore errors from flag parsing.
	// See: https://github.com/hashicorp/terraform/commit/14d378f9ceeb79fd1018da481f1cc25858e0b0de
	cmdFlags.Parse(args)

	// All positional args are recorded and passed to downstream code.
	ret.Args = append(ret.Args, cmdFlags.Args()...)

	return ret, nil
}
