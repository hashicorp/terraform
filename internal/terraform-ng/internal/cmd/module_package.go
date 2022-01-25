package cmd

import (
	"github.com/spf13/cobra"
)

var modulePackageCmd = &cobra.Command{
	Use:   "package",
	Short: "Commands for manipulating module packages.",
	Long: `Commands for manipulating module packages.

A module package is a filesystem subtree which includes one or more Terraform
modules, which are published together as a single artifact.

Modules within a package can refer to each other using local paths, and
intra-package module calls are guaranteed to always select the same module
package version as the caller.

Not all module packages need to be created using these commands. For example,
a Git repository containing one or more modules is automatically a module
package, without the need for any special publishing steps.

However, when using modules for collaboration we recommend publishing explicit
package releases with version numbers into a public or private module registry.`,
}

func init() {
	moduleCmd.AddCommand(modulePackageCmd)
}
