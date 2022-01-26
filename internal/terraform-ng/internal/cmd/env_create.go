package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/terraform-ng/internal/localenv"
)

var envCreateCmd = &cobra.Command{
	Use:   "create [--local] LOCATION CONFIG-SOURCE",
	Short: "Create a new environment.",
	Long: `Create a new environment.

Terraform supports both remote and local environments.

Remote environments belong to a Terraform Cloud or Terraform Enterprise
organization, and save all persistent state in the remote system. Actions on
remote environments happen in a remote execution context, so that you can work
without any local setup aside from configuring your credentials for that system.
You may need to first run "terraform-ng login" to obtain an authentication
token for Terraform Cloud or Terraform Enterprise.

Remote environment locations have the following syntax:
  HOSTNAME/ORGANIZATION/STACK-NAME/ENVIRONMENT-NAME

If you are using Terraform Cloud, you can omit the HOSTNAME/ portion and then
Terraform will automatically use the hostname "app.terraform.io".

Local environments are represented just as definition files on local disk, and
don't have any implicit dependencies on any remote systems. However, you can
optionally configure a remote environment to store mutable state in a remote
system specified explicitly in the environment's definition file. When working
with local environments you will need to ensure that all of the necessary
credentials and other context are available on each system where you will run
Terraform.

A local environment location is a path to its definition file in your local
filesystem. Environment definition files must have the filename suffix
".tfenv.hcl".

For both environment types you must specify an configuration source address
which will be used as the current configuration source for the new environment.
You can change the configuration source later using "terraform-ng up".

Both remote and local environments will typically require further configuration
before they can be used. For example, a remote environment may need to be
passed credentials for the providers used by the given configuration, while
a local environment may need a storage location for its mutable state.
`,

	Args: requiredPositionalArgs("new environment location", "initial configuration source"),

	Run: func(cmd *cobra.Command, args []string) {
		// TEMP: We only support local environments for the moment, while
		// we're just stubbing.
		location := args[0]
		initialConfigSource := args[1]
		if !localenv.ValidEnvironmentFilename(location) {
			cmd.PrintErrln("This stub command currently supports only local environment locations, given as a filename with a .tfenv.hcl suffix.")
			os.Exit(1)
		}

		// FIXME: If the given source is a local path then we ought to
		// reinterpret it so it's relative to the directory containing the
		// environment file, rather than the current working directory, in
		// case those are different.
		sourceAddr, err := addrs.ParseModuleSource(initialConfigSource)
		if err != nil {
			cmd.PrintErrf("Invalid initial configuration source location: %s\n\n", err)
			os.Exit(1)
		}

		def, err := localenv.NewDefinitionFile(location, sourceAddr)
		if err != nil {
			cmd.PrintErrf("Failed to create environment definition file: %s\n\n", err)
			os.Exit(1)
		}

		cmd.Printf("New local environment definition created in %q.\n\n", def.Filename())
	},
}

var envCreateOpts = struct {
	Local bool
}{}

func init() {
	envCreateCmd.Flags().BoolVar(&envCreateOpts.Local, "local", false, "create a local-only environment")

	// TODO: Can we offer extra options for pre-configuring needed settings
	// for an environment? That's tricky both due to how many settings are
	// commonly needed and because the set of required settings will vary
	// considerably for remote vs. local environments.

	envCmd.AddCommand(envCreateCmd)
}
