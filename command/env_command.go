package command

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

// EnvCommand is a Command Implementation that manipulates local state
// environments.
type EnvCommand struct {
	Meta

	newEnv    string
	delEnv    string
	statePath string
	force     bool

	// backend returns by Meta.Backend
	b backend.Backend
	// MultiState Backend
	multi backend.MultiState
}

func (c *EnvCommand) Run(args []string) int {
	args = c.Meta.process(args, true)

	cmdFlags := c.Meta.flagSet("env")
	cmdFlags.StringVar(&c.newEnv, "new", "", "create a new environment")
	cmdFlags.StringVar(&c.delEnv, "delete", "", "delete an existing environment")
	cmdFlags.StringVar(&c.statePath, "state", "", "terraform state file")
	cmdFlags.BoolVar(&c.force, "force", false, "force removal of a non-empty environment")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}
	args = cmdFlags.Args()
	if len(args) > 1 {
		c.Ui.Error("0 or 1 arguments expected.\n")
		return cli.RunResultHelp
	}

	// Load the backend
	b, err := c.Backend(nil)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load backend: %s", err))
		return 1
	}
	c.b = b

	multi, ok := b.(backend.MultiState)
	if !ok {
		c.Ui.Error(envNotSupported)
		return 1
	}
	c.multi = multi

	if c.newEnv != "" {
		return c.createEnv()
	}

	if c.delEnv != "" {
		return c.deleteEnv()
	}

	if len(args) == 1 {
		return c.changeEnv(args[0])
	}

	return c.listEnvs()
}

func (c *EnvCommand) createEnv() int {
	states, _, err := c.multi.States()
	for _, s := range states {
		if c.newEnv == s {
			c.Ui.Error(fmt.Sprintf(envExists, c.newEnv))
			return 1
		}
	}

	err = c.multi.ChangeState(c.newEnv)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	c.Ui.Output(
		c.Colorize().Color(
			fmt.Sprintf(envCreated, c.newEnv),
		),
	)

	if c.statePath == "" {
		// if we're not loading a state, then we're done
		return 0
	}

	// load the new state
	sMgr, err := c.b.State()
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	// load the existing state
	stateFile, err := os.Open(c.statePath)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	s, err := terraform.ReadState(stateFile)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	err = sMgr.WriteState(s)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	return 0
}

func (c *EnvCommand) deleteEnv() int {
	states, current, err := c.multi.States()
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	exists := false
	for _, s := range states {
		if c.delEnv == s {
			exists = true
			break
		}
	}

	if !exists {
		c.Ui.Error(fmt.Sprintf(envDoesNotExist, c.delEnv))
		return 1
	}

	// In order to check if the state being deleted is empty, we need to change
	// to that state and load it.
	if current != c.delEnv {
		if err := c.multi.ChangeState(c.delEnv); err != nil {
			c.Ui.Error(err.Error())
			return 1
		}

		// always try to change back after
		defer func() {
			if err := c.multi.ChangeState(current); err != nil {
				c.Ui.Error(err.Error())
			}
		}()
	}

	// we need the actual state to see if it's empty
	sMgr, err := c.b.State()
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	if err := sMgr.RefreshState(); err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	empty := sMgr.State().Empty()

	if !empty && !c.force {
		c.Ui.Error(fmt.Sprintf(envNotEmpty, c.delEnv))
		return 1
	}

	err = c.multi.DeleteState(c.delEnv)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	c.Ui.Output(
		c.Colorize().Color(
			fmt.Sprintf(envDeleted, c.delEnv),
		),
	)

	if !empty {
		c.Ui.Output(
			c.Colorize().Color(
				fmt.Sprintf(envWarnNotEmpty, c.delEnv),
			),
		)
	}

	return 0
}

func (c *EnvCommand) changeEnv(name string) int {
	states, current, err := c.multi.States()
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	if current == name {
		return 0
	}

	found := false
	for _, s := range states {
		if name == s {
			found = true
			break
		}
	}

	if !found {
		c.Ui.Error(fmt.Sprintf(envDoesNotExist, name))
		return 1
	}

	err = c.multi.ChangeState(name)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	c.Ui.Output(
		c.Colorize().Color(
			fmt.Sprintf(envChanged, name),
		),
	)

	return 0
}

func (c *EnvCommand) listEnvs() int {
	states, current, err := c.multi.States()
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	var out bytes.Buffer
	for _, s := range states {
		if s == current {
			out.WriteString("* ")
		} else {
			out.WriteString("  ")
		}
		out.WriteString(s + "\n")
	}

	c.Ui.Output(out.String())
	return 0
}

func (c *EnvCommand) Help() string {
	helpText := `
Usage: terraform env [options] [NAME]

  Create, change and delete Terraform environments. 

  By default env will list all configured environments. If NAME is provided,
  env will change into to that named environment.


Options:

  -new=name      Create a new environment.
  -delete=name   Delete an existing environment,

  -state=path    Used with -new to copy a state file into the new environment.
  -force         Used with -delete to remove a non-empty environment.
`
	return strings.TrimSpace(helpText)
}

func (c *EnvCommand) Synopsis() string {
	return "Environment management"
}

const (
	envNotSupported = `Backend does not support environments`

	envExists = `Environment %q already exists`

	envDoesNotExist = `Environment %q doesn't exist!
You can create this environment with the "-new" option.`

	envChanged = `[reset][green]Switched to environment %q!`

	envCreated = `[reset][green]Created environment %q!`

	envDeleted = `[reset][green]Deleted environment %q!`

	envNotEmpty = `Environment %[1]q is not empty!
Deleting %[1]q can result in dangling resources: resources that 
exist but are no longer manageable by Terraform. Please destroy
these resources first.  If you want to delete this environment
anyways and risk dangling resources, use the '-force' flag.
`

	envWarnNotEmpty = `[reset][yellow]WARNING: %q was non-empty. 
The resources managed by the deleted environment may still exist,
but are no longer manageable by Terraform since the state has
been deleted.
`
)
