package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var upCmd = &cobra.Command{
	Use:   "up ENVIRONMENT [SOURCE] [options...]",
	Short: "Create or update infrastructure in a particular environment.",
	Long: `Create or update infrastructure in a particular environment.

By default, Terraform will compare the environment's current configuration
against its current infrastructure and propose changes to make the
infrastructure match the desired state described in the configuration.

If you specify the SOURCE argument then Terraform will use that source as a
new configuration for the environment. If you choose to apply the proposed
changes, Terraform will then update the stack to refer to that configuration.

ENVIRONMENT can be either a remote environment, specified by its address, or a
local environment specifed as the path to a local environment definition file.
If you specify a remote environment then Terraform will run this command in that
environment's remote execution context. Some remote environments may be subject
to promotion rules whereby a change must be applied in another predecessor
environment first; use --promote with such an environment to automatically
select the only allowed configuration source.

If you specify a local SOURCE for a remote ENVIRONMENT then Terraform will
upload the given configuration file and all of the module directories it
refers to into the remote execution context in order to run the action
remotely. Some remote environments may not allow direct uploading of
configuration to an environment, and may instead require you to first publish
the configuration to a module registry as part of a module package.
`,

	Args: func(cmd *cobra.Command, args []string) error {
		switch {
		case len(args) < 1:
			return fmt.Errorf("must specify the target environment to update")
		case len(args) > 2:
			return fmt.Errorf("unexpected extra argument %q", args[2])
		default:
			return nil
		}
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return []string{"tfenv.hcl"}, cobra.ShellCompDirectiveFilterFileExt
	},

	Run: stubbedCommand,
}

var upOpts = struct {
	LatestVersion bool
	Promote       bool
}{}

func init() {
	addPlanningOptionsToCommand(upCmd)
	upCmd.Flags().BoolVar(&upOpts.LatestVersion, "latest", false, "upgrade to the latest version from the environment's configuration source")
	upCmd.Flags().BoolVar(&upOpts.LatestVersion, "promote", false, "select the same source as the environment's promotion predecessor, if any")

	rootCmd.AddCommand(upCmd)
}
