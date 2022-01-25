package cmd

import (
	"github.com/spf13/cobra"
)

var moduleValidateCmd = &cobra.Command{
	Use:   "validate [MODULE-DIR]",
	Short: "Check validity of an individual module.",
	Long: `Check validity of an individual module.

This command performs a subset of the Terraform language validation checks that
are relevant for testing a module in isolation, outside the context of any
particular stack.

If you don't specify a module directory, this command will use the current
working directory by default.`,

	Args: cobra.MaximumNArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveFilterDirs
	},

	Run: stubbedCommand,
}

func init() {
	moduleCmd.AddCommand(moduleValidateCmd)
}
