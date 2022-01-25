package cmd

import (
	"github.com/spf13/cobra"
)

// NOTE: This file is named module_test_cmd.go to avoid using a _test.go
// suffix, which the Go toolchain includes only for "go test".

var moduleTestCmd = &cobra.Command{
	Use:   "test [MODULE-DIR]",
	Short: "Run automated integration tests for a module.",
	Long: `Run automated integration tests for a module.

Searches a subdirectory named "tests" for one or more module testing scenarios,
tries each of those scenarios, and reports the results.

If you don't specify a module directory, this command will use the current
working directory by default.

Tests will be run on your local system by default, which means you'll need to
configure any necessary provider credentials on your local system. If you have
access to a remote execution environment with suitable stored credentials, you
can specify its address using the --remote option.
`,

	Args: cobra.MaximumNArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveFilterDirs
	},

	Run: stubbedCommand,
}

var moduleTestArgs = struct {
	RemoteContext string
}{}

func init() {
	moduleTestCmd.Flags().StringVarP(&moduleTestArgs.RemoteContext, "remote", "", "", "run tests in a specific remote execution context")

	moduleCmd.AddCommand(moduleTestCmd)
}
