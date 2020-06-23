package command

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/tfdiags"
)

// ProvidersListCommand is a Command implementation that searches local
// provider mirror directories, and displays diagnostic information about the
// layout of providers found.
type ProvidersListCommand struct {
	Meta
}

func (c *ProvidersListCommand) Help() string {
	return providersListCommandHelp
}

func (c *ProvidersListCommand) Synopsis() string {
	return "FIXME"
}

func (c *ProvidersListCommand) Run(args []string) int {
	var pluginDirs FlagStringSlice
	args = c.Meta.process(args)
	cmdFlags := c.Meta.defaultFlagSet("providers list")
	cmdFlags.Var(&pluginDirs, "plugin-dir", "plugin directory")

	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return 1
	}

	if len(pluginDirs) > 0 {
		c.pluginPath = pluginDirs
	}

	if len(pluginDirs) == 0 {
		// By default we use a source that looks for providers in all of the
		// standard locations, possibly customized by the user in CLI config.
		inst := c.providerInstaller()

		pluginDirs = inst.Source().LocalDirs()
	}

	var diags tfdiags.Diagnostics

	for _, pluginDir := range pluginDirs {
		available, searchDiags := getproviders.SearchLocalDirectoryDiags(pluginDir)
		diags = diags.Append(searchDiags)

		if number := len(available); number > 0 {
			noun := "providers"
			if number == 1 {
				noun = "provider"
			}
			c.Ui.Output(fmt.Sprintf("Found %d %s in %s\n", number, noun, pluginDir))

			for _, metaList := range available {
				for _, meta := range metaList {
					c.Ui.Output(fmt.Sprintf("  %s: %s %s", meta.Provider.String(), meta.Version, meta.TargetPlatform))
				}
			}
		} else {
			c.Ui.Output(fmt.Sprintf("No providers found in %s", pluginDir))
		}
	}

	c.showDiagnostics(diags)
	return 0
}

const providersListCommandHelp = `
Usage: terraform providers list

  FIXME
`
