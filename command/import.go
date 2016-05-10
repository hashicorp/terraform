package command

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/terraform"
)

// ImportCommand is a cli.Command implementation that adds an existing
// resource to the state file
type ImportCommand struct {
	Meta
}

func (c *ImportCommand) Run(args []string) int {
	args = c.Meta.process(args, false)

	var module string
	cmdFlags := c.Meta.flagSet("import")
	cmdFlags.StringVar(&module, "module", "", "module")
	cmdFlags.StringVar(&c.Meta.statePath, "state", DefaultStateFilename, "path")
	cmdFlags.StringVar(&c.Meta.stateOutPath, "state-out", "", "path")
	cmdFlags.StringVar(&c.Meta.backupPath, "backup", "", "path")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	// Require the two arguments for the resource to import
	args = cmdFlags.Args()
	if len(args) != 2 {
		c.Ui.Error("The import command expects exactly two arguments.")
		cmdFlags.Usage()
		return 1
	}

	name := args[0]
	resourceID := args[1]
	if module == "" {
		module = "root"
	} else {
		module = "root." + module
	}

	// Get the state that we'll be modifying
	state, err := c.State()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load state: %s", err))
		return 1
	}

	// Get the actual state structure
	s := state.State()

	// Get the proper module containing the resource to import.
	modPath := strings.Split(module, ".")
	mod := s.ModuleByPath(modPath)
	if mod == nil {
		c.Ui.Error(fmt.Sprintf(
			"The module %s could not be found. Module must first exist in order to import.",
			module))
		return 1
	}

	// Get the resource we're looking for
	_, ok := mod.Resources[name]
	if ok {
		c.Ui.Error(fmt.Sprintf(
			"Cannot import resource %s that already exists in the module %s.",
			name,
			module))
		return 1
	}

	// Import the resource
	rs := &terraform.ResourceState{
		Type: strings.Split(name, ".")[0],
		Primary: &terraform.InstanceState{
			ID: resourceID,
		},
	}
	mod.Resources[name] = rs

	log.Printf("[INFO] Writing state output to: %s", c.Meta.StateOutPath())
	if err := c.Meta.PersistState(s); err != nil {
		c.Ui.Error(fmt.Sprintf("Error writing state file: %s", err))
		return 1
	}

	c.Ui.Output(fmt.Sprintf(
		"The resource %s in the module %s has been imported. Please refresh.!",
		name, module))
	return 0
}

func (c *ImportCommand) Help() string {
	helpText := `
Usage: terraform import [options] name id

  Manually import a resource using the provided id.

  This will not modify your infrastructure. This command changes your
  state to set the specified resource's id to the value provided.

Options:

  -backup=path        Path to backup the existing state file before
                      modifying. Defaults to the "-state-out" path with
                      ".backup" extension. Set to "-" to disable backup.

  -module=path        The module path where the resource lives. By
                      default this will be root. Child modules can be specified
                      by names. Ex. "consul" or "consul.vpc" (nested modules).

  -no-color           If specified, output won't contain any color.

  -state=path         Path to read and save state (unless state-out
                      is specified). Defaults to "terraform.tfstate".

  -state-out=path     Path to write updated state file. By default, the
                      "-state" path will be used.

`
	return strings.TrimSpace(helpText)
}

func (c *ImportCommand) Synopsis() string {
	return "Import an existing resource to manage"
}
