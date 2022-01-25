package cmd

import (
	"github.com/spf13/cobra"
)

var tryCmd = &cobra.Command{
	Use:   "try ENVIRONMENT SOURCE [options...]",
	Short: "Preview the effect of applying a new configuration to an existing environment.",
	Long: `Preview the effect of applying a new configuration to an existing environment.

This command is similar to "terraform-ng up", but only generates speculative
plans, where Terraform will propose a set of actions but will not offer the
possibility of applying them.

ENVIRONMENT can be either a remote environment, specified by its address, or a
local environment specifed as the path to a local environment file. If you
specify a remote environment then Terraform will run this command in that
environment's remote execution context.

If you specify a local SOURCE for a remote ENVIRONMENT then Terraform will
upload the given configuration file and all of the module directories it
refers to into the remote execution context in order to run the action
remotely. The remote system may therefore store those configuration files,
but it will not update the remote environment to use that configuration for
future updates.
`,

	Args: requiredPositionalArgs("target environment", "new configuration source"),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return []string{"tfenv.hcl"}, cobra.ShellCompDirectiveFilterFileExt
	},

	Run: stubbedCommand,
}

func init() {
	addPlanningOptionsToCommand(tryCmd)

	rootCmd.AddCommand(tryCmd)
}
