package cmd

import (
	"github.com/spf13/cobra"
)

var modulePackagePublishCmd = &cobra.Command{
	Use:   "publish [SOURCE-DIR] [DESTINATION]",
	Short: "Publish a module package to a module registry.",
	Long: `Publish a module package to a module registry.

For module registries that are configured to allow direct publishing, this
command will create a module package containing all of the files and directories
under the specified source directory, excluding any patterns declared in
optional .terraformignore files, and publish it to a remote module registry
as a new version.

DESTINATION must have the following syntax:
  REGISTRY-HOST/NAMESPACE/NAME/TARGET-SYSTEM@VERSION

The part before the @ is a normal registry module package address, including the
hostname of the registry to publish to. The meaning of the other address
components are decided separately by each registry. VERSION is a
semantic-version string, like "1.0.0", specifying the new version to create.

Some registries may disallow direct publication of new versions for some or all
modules. For example, a registry might instead react automatically to
notifications from a source Git repository and consider each new version tag
as a new module package version.

By default, registries should block attempts to overwrite a previously-published
version, because consumers typically expect published module packages to be
immutable. However, some registries may allow privileged users to force
overwriting an existing package using the --overwrite option, described below.
`,

	Args: requiredPositionalArgs("package source directory", "destination address and version"),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveFilterDirs
	},

	Run: stubbedCommand,
}

var modulePackagePublishOpts = struct {
	Overwrite bool
}{}

func init() {
	modulePackagePublishCmd.Flags().BoolVar(&modulePackagePublishOpts.Overwrite, "overwrite", false, "try to overwrite an already-existing version")

	modulePackageCmd.AddCommand(modulePackagePublishCmd)
}
