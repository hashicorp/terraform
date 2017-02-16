package command

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

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
	newEnv := strings.TrimSpace(c.newEnv)
	if newEnv == "" {
		c.Ui.Error(fmt.Sprintf("invalid environment: %q", c.newEnv))
		return 1
	}

	envs, err := ListEnvs()
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	for _, env := range envs {
		if newEnv == env {
			c.Ui.Error(fmt.Sprintf(envExists, newEnv))
			return 1
		}
	}

	err = os.MkdirAll(filepath.Join(DefaultEnvDir, newEnv), 0755)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("error creating environment directory: %s", err))
		return 1
	}

	if c.statePath != "" {
		stateData, err := ioutil.ReadFile(c.statePath)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("error reading state file: %s", err))
			return 1
		}

		newState := filepath.Join(DefaultEnvDir, newEnv, DefaultStateFilename)
		err = ioutil.WriteFile(newState, stateData, 0644)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("error writing state file: %s", err))
			return 1
		}
	}

	c.Ui.Output(
		c.Colorize().Color(
			fmt.Sprintf(envCreated, newEnv),
		),
	)

	return c.changeEnv(newEnv)
}

func (c *EnvCommand) deleteEnv() int {
	envs, err := ListEnvs()
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	delEnv := ""
	for _, env := range envs {
		if c.delEnv == env {
			delEnv = env
			break
		}
	}

	if delEnv == "" {
		c.Ui.Error(fmt.Sprintf(envDoesNotExist, c.delEnv))
		return 1
	}

	warnNotEmpty := false

	statePath := filepath.Join(DefaultEnvDir, delEnv, DefaultStateFilename)
	stateFile, err := os.Open(statePath)
	if err != nil {
		if !os.IsNotExist(err) {
			c.Ui.Error(err.Error())
			return 1
		}
	} else {
		defer stateFile.Close()
		state, err := terraform.ReadState(stateFile)
		// no need to check the error, as invalid state might as well be no
		// state.
		if err == nil && !state.Empty() {
			if !c.force {
				c.Ui.Error(fmt.Sprintf(envNotEmpty, delEnv))
				return 1
			}

			warnNotEmpty = true
		}
	}

	err = os.RemoveAll(filepath.Join(DefaultEnvDir, delEnv))
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	c.Ui.Output(
		c.Colorize().Color(
			fmt.Sprintf(envDeleted, delEnv),
		),
	)

	if warnNotEmpty {
		c.Ui.Output(
			c.Colorize().Color(
				fmt.Sprintf(envWarnNotEmpty, delEnv),
			),
		)
	}

	return 0
}

func (c *EnvCommand) changeEnv(newEnv string) int {
	current, err := CurrentEnv()
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	if newEnv == current {
		return 0
	}

	envs, err := ListEnvs()
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	exists := false
	for _, env := range envs {
		if env == newEnv {
			exists = true
			break
		}
	}

	if !exists {
		c.Ui.Error(fmt.Sprintf(envDoesNotExist, newEnv))
		return 1
	}

	err = os.MkdirAll(DefaultDataDir, 0755)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	err = ioutil.WriteFile(
		filepath.Join(DefaultDataDir, DefaultEnvFile),
		[]byte(newEnv),
		0644,
	)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	c.Ui.Output(
		c.Colorize().Color(
			fmt.Sprintf(envChanged, newEnv),
		),
	)

	return 0
}

func (c *EnvCommand) listEnvs() int {
	envs, err := ListEnvs()
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	current, err := CurrentEnv()
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	var out bytes.Buffer
	for _, env := range envs {
		if env == current {
			fmt.Fprintf(&out, "* %s\n", env)
		} else {
			fmt.Fprintf(&out, "  %s\n", env)
		}
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
