package command

import (
	"fmt"
	"sort"

	"github.com/hashicorp/terraform/moduledeps"
	"github.com/hashicorp/terraform/terraform"
	"github.com/xlab/treeprint"
)

// ProvidersCommand is a Command implementation that prints out information
// about the providers used in the current configuration/state.
type ProvidersCommand struct {
	Meta
}

func (c *ProvidersCommand) Help() string {
	return providersCommandHelp
}

func (c *ProvidersCommand) Synopsis() string {
	return "Prints a tree of the providers used in the configuration"
}

func (c *ProvidersCommand) Run(args []string) int {
	c.Meta.process(args, false)

	cmdFlags := c.Meta.flagSet("providers")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	configPath, err := ModulePath(cmdFlags.Args())
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	// Load the config
	root, err := c.Module(configPath)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load root config module: %s", err))
		return 1
	}
	if root == nil {
		c.Ui.Error(fmt.Sprintf(
			"No configuration files found in the directory: %s\n\n"+
				"This command requires configuration to run.",
			configPath))
		return 1
	}

	// Validate the config (to ensure the version constraints are valid)
	if diags := root.Validate(); len(diags) != 0 {
		c.showDiagnostics(diags)
		if diags.HasErrors() {
			return 1
		}
	}

	// Load the backend
	b, err := c.Backend(&BackendOpts{
		Config: root.Config(),
	})
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load backend: %s", err))
		return 1
	}

	// Get the state
	env := c.Workspace()
	state, err := b.State(env)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load state: %s", err))
		return 1
	}
	if err := state.RefreshState(); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load state: %s", err))
		return 1
	}

	s := state.State()

	depTree := terraform.ModuleTreeDependencies(root, s)
	depTree.SortDescendents()

	printRoot := treeprint.New()
	providersCommandPopulateTreeNode(printRoot, depTree)

	c.Ui.Output(printRoot.String())

	return 0
}

func providersCommandPopulateTreeNode(node treeprint.Tree, deps *moduledeps.Module) {
	names := make([]string, 0, len(deps.Providers))
	for name := range deps.Providers {
		names = append(names, string(name))
	}
	sort.Strings(names)

	for _, name := range names {
		dep := deps.Providers[moduledeps.ProviderInstance(name)]
		versionsStr := dep.Constraints.String()
		if versionsStr != "" {
			versionsStr = " " + versionsStr
		}
		var reasonStr string
		switch dep.Reason {
		case moduledeps.ProviderDependencyInherited:
			reasonStr = " (inherited)"
		case moduledeps.ProviderDependencyFromState:
			reasonStr = " (from state)"
		}
		node.AddNode(fmt.Sprintf("provider.%s%s%s", name, versionsStr, reasonStr))
	}

	for _, child := range deps.Children {
		childNode := node.AddBranch(fmt.Sprintf("module.%s", child.Name))
		providersCommandPopulateTreeNode(childNode, child)
	}
}

const providersCommandHelp = `
Usage: terraform providers [dir]

  Prints out a tree of modules in the referenced configuration annotated with
  their provider requirements.

  This provides an overview of all of the provider requirements across all
  referenced modules, as an aid to understanding why particular provider
  plugins are needed and why particular versions are selected.

`
