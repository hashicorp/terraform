package command

import (
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform/dag"
	"github.com/mitchellh/cli"
)

// DebugJSON2DotCommand is a Command implementation that translates a json
// graph debug log to Dot format.
type DebugJSON2DotCommand struct {
	Meta
}

func (c *DebugJSON2DotCommand) Run(args []string) int {
	args, err := c.Meta.process(args, true)
	if err != nil {
		return 1
	}
	cmdFlags := c.Meta.flagSet("debug json2dot")

	if err := cmdFlags.Parse(args); err != nil {
		return cli.RunResultHelp
	}

	fileName := cmdFlags.Arg(0)
	if fileName == "" {
		return cli.RunResultHelp
	}

	f, err := os.Open(fileName)
	if err != nil {
		c.Ui.Error(fmt.Sprintf(errInvalidLog, err))
		return cli.RunResultHelp
	}

	dot, err := dag.JSON2Dot(f)
	if err != nil {
		c.Ui.Error(fmt.Sprintf(errInvalidLog, err))
		return cli.RunResultHelp
	}

	c.Ui.Output(string(dot))
	return 0
}

func (c *DebugJSON2DotCommand) Help() string {
	helpText := `
Usage: terraform debug json2dot input.json

  Translate a graph debug file to dot format.

  This command takes a single json graph log file and converts it to a single
  dot graph written to stdout.
`
	return strings.TrimSpace(helpText)
}

func (c *DebugJSON2DotCommand) Synopsis() string {
	return "Convert json graph log to dot"
}

const errInvalidLog = `Error parsing log file: %[1]s`
