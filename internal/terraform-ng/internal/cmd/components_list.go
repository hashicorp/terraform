package cmd

import (
	"fmt"
	"os"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/terraform-ng/internal/tfcomponents/componentstree"
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
		loader := componentstree.NewLocalOnlyConfigLoader(".")
		sourceAddr, err := addrs.ParseModuleSource(args[0])
		if err != nil {
			cmd.PrintErrf("Invalid components configuration address %q: %s\n", args[0], err)
			os.Exit(1)
		}

		root, diags := componentstree.LoadComponentsTree(sourceAddr, loader)
		if diags.HasErrors() {
			for _, diag := range diags {
				cmd.PrintErrln(format.DiagnosticPlain(diag, nil, 78))
			}
			os.Exit(1)
		}

		componentsListListComponents("", root)
	},
}

func componentsListListComponents(prefix string, node *componentstree.Node) {
	for _, component := range node.Config.Components {
		if component.ForEach != nil {
			fmt.Printf("%s%s[...] at %s\n", prefix, component.Name, component.DeclRange.StartString())
		} else {
			fmt.Printf("%s%s at %s\n", prefix, component.Name, component.DeclRange.StartString())
		}
	}
	for callAddr, child := range node.Children {
		config := node.ChildCallConfig(callAddr)
		if config.ForEach != nil {
			componentsListListComponents(prefix+callAddr.Name+"[...].", child)
		} else {
			componentsListListComponents(prefix+callAddr.Name+".", child)
		}
	}
}

func init() {
	componentsCmd.AddCommand(componentsListCmd)
}
