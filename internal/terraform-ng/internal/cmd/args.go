package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func requiredPositionalArgs(names ...string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		switch {
		case len(args) == len(names):
			return nil
		case len(args) > len(names):
			return fmt.Errorf("unexpected extra argument %q", args[len(names)])
		default:
			// We'll complain about the first argument that's omitted, and
			// assume the user will refer to the accompanying usage output
			// to see what else they need for a multi-argument command.
			missing := names[len(args)]
			return fmt.Errorf("no argument given for %s", missing)
		}
	}
}

func requiredPositionalArgsPrefix(names ...string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		switch {
		case len(args) >= len(names):
			return nil
		default:
			// We'll complain about the first argument that's omitted, and
			// assume the user will refer to the accompanying usage output
			// to see what else they need for a multi-argument command.
			missing := names[len(args)]
			return fmt.Errorf("no argument given for %s argument", missing)
		}
	}
}
