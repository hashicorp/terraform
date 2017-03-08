package command

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
	"github.com/ryanuber/columnize"
)

// StateShowCommand is a Command implementation that shows a single resource.
type StateShowCommand struct {
	Meta
	StateMeta
}

func (c *StateShowCommand) Run(args []string) int {
	args, err := c.Meta.process(args, true)
	if err != nil {
		return 1
	}

	cmdFlags := c.Meta.flagSet("state show")
	cmdFlags.StringVar(&c.Meta.statePath, "state", DefaultStateFilename, "path")
	if err := cmdFlags.Parse(args); err != nil {
		return cli.RunResultHelp
	}
	args = cmdFlags.Args()

	// Load the backend
	b, err := c.Backend(nil)
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

	stateReal := state.State()
	if stateReal == nil {
		c.Ui.Error(fmt.Sprintf(errStateNotFound))
		return 1
	}

	filter := &terraform.StateFilter{State: stateReal}
	results, err := filter.Filter(args...)
	if err != nil {
		c.Ui.Error(fmt.Sprintf(errStateFilter, err))
		return 1
	}

	if len(results) == 0 {
		return 0
	}

	instance, err := c.filterInstance(results)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	if instance == nil {
		return 0
	}

	is := instance.Value.(*terraform.InstanceState)

	// Sort the keys
	var keys []string
	for k, _ := range is.Attributes {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build the output
	var output []string
	output = append(output, fmt.Sprintf("id | %s", is.ID))
	for _, k := range keys {
		if k != "id" {
			output = append(output, fmt.Sprintf("%s | %s", k, is.Attributes[k]))
		}
	}

	// Output
	config := columnize.DefaultConfig()
	config.Glue = " = "
	c.Ui.Output(columnize.Format(output, config))
	return 0
}

func (c *StateShowCommand) Help() string {
	helpText := `
Usage: terraform state show [options] ADDRESS

  Shows the attributes of a resource in the Terraform state.

  This command shows the attributes of a single resource in the Terraform
  state. The address argument must be used to specify a single resource.
  You can view the list of available resources with "terraform state list".

Options:

  -state=statefile    Path to a Terraform state file to use to look
                      up Terraform-managed resources. By default it will
                      use the state "terraform.tfstate" if it exists.

`
	return strings.TrimSpace(helpText)
}

func (c *StateShowCommand) Synopsis() string {
	return "Show a resource in the state"
}
