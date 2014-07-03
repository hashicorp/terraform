package command

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

// RefreshCommand is a cli.Command implementation that refreshes the state
// file.
type RefreshCommand struct {
	ContextOpts *terraform.ContextOpts
	Ui          cli.Ui
}

func (c *RefreshCommand) Run(args []string) int {
	var outPath string
	statePath := "terraform.tfstate"
	configPath := "."

	cmdFlags := flag.NewFlagSet("refresh", flag.ContinueOnError)
	cmdFlags.StringVar(&outPath, "out", "", "output path")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) != 2 {
		// TODO(mitchellh): this is temporary until we can assume current
		// dir for Terraform config.
		c.Ui.Error("TEMPORARY: The refresh command requires two args.")
		cmdFlags.Usage()
		return 1
	}

	statePath = args[0]
	configPath = args[1]
	if outPath == "" {
		outPath = statePath
	}

	// Load up the state
	f, err := os.Open(statePath)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error loading state: %s", err))
		return 1
	}

	state, err := terraform.ReadState(f)
	f.Close()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error loading state: %s", err))
		return 1
	}

	b, err := config.Load(configPath)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error loading blueprint: %s", err))
		return 1
	}

	c.ContextOpts.Config = b
	c.ContextOpts.Hooks = append(c.ContextOpts.Hooks, &UiHook{Ui: c.Ui})
	ctx := terraform.NewContext(c.ContextOpts)

	state, err = ctx.Refresh()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error refreshing state: %s", err))
		return 1
	}

	log.Printf("[INFO] Writing state output to: %s", outPath)
	f, err = os.Create(outPath)
	if err == nil {
		defer f.Close()
		err = terraform.WriteState(state, f)
	}
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error writing state file: %s", err))
		return 1
	}

	return 0
}

func (c *RefreshCommand) Help() string {
	helpText := `
Usage: terraform refresh [options] [terraform.tfstate] [terraform.tf]

  Refresh and update the state of your infrastructure. This is read-only
  operation that will not modify infrastructure. The read-only property
  is dependent on resource providers being implemented correctly.

Options:

  -out=path     Path to write updated state file. If this is not specified,
                the existing state file will be overridden.

`
	return strings.TrimSpace(helpText)
}

func (c *RefreshCommand) Synopsis() string {
	return "Refresh the state of your infrastructure"
}
