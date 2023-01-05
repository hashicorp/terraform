package command

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/command/jsonfunction"
	"github.com/hashicorp/terraform/internal/lang"
)

// MetadataFunctionsCommand is a Command implementation that prints out information
// about the available functions in Terraform.
type MetadataFunctionsCommand struct {
	Meta
}

func (c *MetadataFunctionsCommand) Help() string {
	return metadataFunctionsCommandHelp
}

func (c *MetadataFunctionsCommand) Synopsis() string {
	return "Show signatures and descriptions for the available functions"
}

func (c *MetadataFunctionsCommand) Run(args []string) int {
	args = c.Meta.process(args)
	cmdFlags := c.Meta.defaultFlagSet("metadata functions")
	var jsonOutput bool
	cmdFlags.BoolVar(&jsonOutput, "json", false, "produce JSON output")

	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return 1
	}

	if !jsonOutput {
		c.Ui.Error(
			"The `terraform metadata functions` command requires the `-json` flag.\n")
		cmdFlags.Usage()
		return 1
	}

	scope := &lang.Scope{
		ConsoleMode: true,
		BaseDir:     ".", // TODO? might be omitted
	}
	funcs := scope.Functions()
	jsonFunctions, err := jsonfunction.Marshal(funcs)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to marshal function signatures to json: %s", err))
		return 1
	}
	c.Ui.Output(string(jsonFunctions))

	return 0
}

const metadataFunctionsCommandHelp = `
Usage: terraform [global options] metadata functions -json

  Prints out a json representation of the available function signatures.
`
