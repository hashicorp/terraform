package command

import (
	"fmt"
	"log"
	"strings"
)

// TaintCommand is a cli.Command implementation that refreshes the state
// file.
type TaintCommand struct {
	Meta
}

func (c *TaintCommand) Run(args []string) int {
	args = c.Meta.process(args, false)

	cmdFlags := c.Meta.flagSet("taint")
	cmdFlags.StringVar(&c.Meta.statePath, "state", DefaultStateFilename, "path")
	cmdFlags.StringVar(&c.Meta.stateOutPath, "state-out", "", "path")
	cmdFlags.StringVar(&c.Meta.backupPath, "backup", "", "path")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	// Require the one argument for the resource to taint
	args = cmdFlags.Args()
	if len(args) != 1 {
		c.Ui.Error("The taint command expects exactly one argument.")
		cmdFlags.Usage()
		return 1
	}
	name := args[0]

	// Get the state that we'll be modifying
	state, err := c.State()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load state: %s", err))
		return 1
	}

	// Get the actual state structure
	s := state.State()
	if s.Empty() {
		c.Ui.Error(fmt.Sprintf(
			"The state is empty. The most common reason for this is that\n" +
				"an invalid state file path was given or Terraform has never\n " +
				"been run for this infrastructure. Infrastructure must exist\n" +
				"for it to be tainted."))
		return 1
	}

	mod := s.RootModule()

	// If there are no resources in this module, it is an error
	if len(mod.Resources) == 0 {
		c.Ui.Error(fmt.Sprintf(
			"The module %s has no resources. There is nothing to taint.",
			strings.Join(mod.Path, ".")))
		return 1
	}

	// Get the resource we're looking for
	rs, ok := mod.Resources[name]
	if !ok {
		c.Ui.Error(fmt.Sprintf(
			"The resource %s couldn't be found in the module %s.",
			name,
			strings.Join(mod.Path, ".")))
		return 1
	}

	// Taint the resource
	rs.Taint()

	log.Printf("[INFO] Writing state output to: %s", c.Meta.StateOutPath())
	if err := c.Meta.PersistState(s); err != nil {
		c.Ui.Error(fmt.Sprintf("Error writing state file: %s", err))
		return 1
	}

	return 0
}

func (c *TaintCommand) Help() string {
	helpText := `
Usage: terraform taint [options] name

  Manually mark a resource as tainted, forcing a destroy and recreate
  on the next plan/apply.

  This will not modify your infrastructure. This command changes your
  state to mark a resource as tainted so that during the next plan or
  apply, that resource will be destroyed and recreated. This command on
  its own will not modify infrastructure. This command can be undone by
  reverting the state backup file that is created.

Options:

  -backup=path        Path to backup the existing state file before
                      modifying. Defaults to the "-state-out" path with
                      ".backup" extension. Set to "-" to disable backup.

  -no-color           If specified, output won't contain any color.

  -state=path         Path to read and save state (unless state-out
                      is specified). Defaults to "terraform.tfstate".

  -state-out=path     Path to write updated state file. By default, the
                      "-state" path will be used.

`
	return strings.TrimSpace(helpText)
}

func (c *TaintCommand) Synopsis() string {
	return "Manually mark a resource for recreation"
}
