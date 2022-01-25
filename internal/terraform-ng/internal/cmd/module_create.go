package cmd

import (
	"github.com/spf13/cobra"
)

const moduleCreateEditionDefault = "TF2021"

var moduleCreateValidEditions = []string{"TF2021"}

var moduleCreateCmd = &cobra.Command{
	Use:   "create DIR",
	Short: "Create boilerplate for a new module in a local directory.",
	Long: `Create the boilerplate for a new module in a local directory.

The given path must either be an empty directory or not exist at all. In both
cases, this command will populate the directory with an initial .tf file and
some other supporting files.`,

	Args: requiredPositionalArgs("target directory"),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveFilterDirs
	},

	Run: stubbedCommand,
}

var moduleCreateArgs = struct {
	Edition string
}{
	Edition: moduleCreateEditionDefault,
}

func init() {
	moduleCreateCmd.Flags().StringVarP(&moduleCreateArgs.Edition, "edition", "", moduleCreateEditionDefault, "language edition to select for the new module")
	moduleCreateCmd.RegisterFlagCompletionFunc("edition", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return moduleCreateValidEditions, cobra.ShellCompDirectiveDefault
	})

	moduleCmd.AddCommand(moduleCreateCmd)
}
