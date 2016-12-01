package command

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform/helper/wrappedstreams"
	"github.com/hashicorp/terraform/repl"

	"github.com/mitchellh/cli"
)

// ConsoleCommand is a Command implementation that applies a Terraform
// configuration and actually builds or changes infrastructure.
type ConsoleCommand struct {
	Meta

	// When this channel is closed, the apply will be cancelled.
	ShutdownCh <-chan struct{}
}

func (c *ConsoleCommand) Run(args []string) int {
	args = c.Meta.process(args, true)
	cmdFlags := c.Meta.flagSet("console")
	cmdFlags.StringVar(&c.Meta.statePath, "state", DefaultStateFilename, "path")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	pwd, err := os.Getwd()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error getting pwd: %s", err))
		return 1
	}

	var configPath string
	args = cmdFlags.Args()
	if len(args) > 1 {
		c.Ui.Error("The console command expects at most one argument.")
		cmdFlags.Usage()
		return 1
	} else if len(args) == 1 {
		configPath = args[0]
	} else {
		configPath = pwd
	}

	// Build the context based on the arguments given
	ctx, _, err := c.Context(contextOpts{
		Path:        configPath,
		PathEmptyOk: true,
		StatePath:   c.Meta.statePath,
	})
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	// Setup the UI so we can output directly to stdout
	ui := &cli.BasicUi{
		Writer:      wrappedstreams.Stdout(),
		ErrorWriter: wrappedstreams.Stderr(),
	}

	// IO Loop
	session := &repl.Session{
		Interpolater: ctx.Interpolater(),
	}

	// Determine if stdin is a pipe. If so, we evaluate directly.
	if c.StdinPiped() {
		return c.modePiped(session, ui)
	}

	return c.modeInteractive(session, ui)
}

func (c *ConsoleCommand) modePiped(session *repl.Session, ui cli.Ui) int {
	var lastResult string
	scanner := bufio.NewScanner(wrappedstreams.Stdin())
	for scanner.Scan() {
		// Handle it. If there is an error exit immediately
		result, err := session.Handle(strings.TrimSpace(scanner.Text()))
		if err != nil {
			ui.Error(err.Error())
			return 1
		}

		// Store the last result
		lastResult = result
	}

	// Output the final result
	ui.Output(lastResult)

	return 0
}

func (c *ConsoleCommand) Help() string {
	helpText := `
Usage: terraform console [options] [DIR]

  Starts an interactive console for experimenting with Terraform
  interpolations.

  This will open an interactive console that you can use to type
  interpolations into and inspect their values. This command loads the
  current state. This lets you explore and test interpolations before
  using them in future configurations.

  This command will never modify your state.

  DIR can be set to a directory with a Terraform state to load. By
  default, this will default to the current working directory.

Options:

  -state=path            Path to read state. Defaults to "terraform.tfstate"

  -var 'foo=bar'         Set a variable in the Terraform configuration. This
                         flag can be set multiple times.

  -var-file=foo          Set variables in the Terraform configuration from
                         a file. If "terraform.tfvars" is present, it will be
                         automatically loaded if this flag is not specified.


`
	return strings.TrimSpace(helpText)
}

func (c *ConsoleCommand) Synopsis() string {
	return "Interactive console for Terraform interpolations"
}
