// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"slices"
	"strings"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

type Stacks struct {
	PluginCacheDirOverride string

	// Flags and positional arguments are kept and passed to the stacks plugin.
	Args []string
}

func ParseStacks(args []string) (*Stacks, tfdiags.Diagnostics) {
	ret := &Stacks{
		Args: []string{}, // Ensure non-nil
	}

	// The only flag relevant in `StacksCommand` is the plugin-cache-dir flag
	cmdFlags := defaultFlagSet("stacks")
	cmdFlags.StringVar(&ret.PluginCacheDirOverride, "plugin-cache-dir", "", "plugin cache directory")

	// We choose to ignore errors from flag parsing.
	// This is because we're processing the plugin-cache-dir flag alongside flags that are unrecognized
	// here but will be passed on to the stacks plugin. It's the plugin's responsibility to handle them.
	// See: https://github.com/hashicorp/terraform/commit/14d378f9ceeb79fd1018da481f1cc25858e0b0de
	cmdFlags.Parse(args)

	// We need to return all flags other than plugin-cache-dir as well as
	// actual positional arguments.
	// To do this we have to use the original input instead of the `flag` package.
	downstreamArgs := slices.DeleteFunc(args, func(arg string) bool {
		return strings.HasPrefix(arg, "-plugin-cache-dir")
	})
	ret.Args = append(ret.Args, downstreamArgs...)

	return ret, nil
}
