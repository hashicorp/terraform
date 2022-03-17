package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var componentsListCmd = &cobra.Command{
	Use:   "list COMPONENTS-SOURCE",
	Short: "List the components defined in a .tfcomponents.hcl file and its descendent groups.",

	Args: cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return []string{"tfcomponents.hcl"}, cobra.ShellCompDirectiveFilterFileExt
	},

	Run: func(cmd *cobra.Command, args []string) {
		cmd.PrintErrln("Not yet implemented.")
		os.Exit(1)
	},
}

func init() {
	componentsCmd.AddCommand(componentsListCmd)
}
