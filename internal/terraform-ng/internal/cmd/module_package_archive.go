package cmd

import (
	"github.com/spf13/cobra"
)

var modulePackageArchiveCmd = &cobra.Command{
	Use:   "archive [SOURCE-DIR]",
	Short: "Create a zip archive to use as a module package.",
	Long: `Create a zip archive to use as a module package.

Terraform supports various different source address types for module packages,
some of which require publishing an archive into a file or blob storage
service.

This command creates a .zip archive containing all of the files and directories
under the specified source directory, excluding any patterns declared in
optional .terraformignore files.

You don't need to use this command if you intend to use commits or tags in a
Git repository as module package versions. Terraform can directly clone a Git
repository and use the tree of a commit as a module package.

By default, Terraform selects a filename by appending .zip to the given
directory name and creating the file alongside that directory. You can override
this using the --output option, as described below.
`,

	Args: requiredPositionalArgs("package source directory"),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveFilterDirs
	},

	Run: stubbedCommand,
}

var modulePackageArchiveArgs = struct {
	OutputPath string
}{}

func init() {
	modulePackageArchiveCmd.Flags().StringVarP(&modulePackageArchiveArgs.OutputPath, "output", "o", "", "filename for the resulting .zip archive file")
	modulePackageArchiveCmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"zip"}, cobra.ShellCompDirectiveFilterFileExt
	})

	modulePackageCmd.AddCommand(modulePackageArchiveCmd)
}
