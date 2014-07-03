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

// PlanCommand is a Command implementation that compares a Terraform
// configuration to an actual infrastructure and shows the differences.
type PlanCommand struct {
	ContextOpts *terraform.ContextOpts
	Ui          cli.Ui
}

func (c *PlanCommand) Run(args []string) int {
	var destroy, refresh bool
	var outPath, statePath string

	cmdFlags := flag.NewFlagSet("plan", flag.ContinueOnError)
	cmdFlags.BoolVar(&destroy, "destroy", false, "destroy")
	cmdFlags.BoolVar(&refresh, "refresh", true, "refresh")
	cmdFlags.StringVar(&outPath, "out", "", "path")
	cmdFlags.StringVar(&statePath, "state", "", "path")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) != 1 {
		c.Ui.Error(
			"The plan command expects only one argument with the path\n" +
				"to a Terraform configuration.\n")
		cmdFlags.Usage()
		return 1
	}

	// Load up the state
	var state *terraform.State
	if statePath != "" {
		f, err := os.Open(statePath)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error loading state: %s", err))
			return 1
		}

		state, err = terraform.ReadState(f)
		f.Close()
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error loading state: %s", err))
			return 1
		}
	}

	b, err := config.Load(args[0])
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error loading blueprint: %s", err))
		return 1
	}

	c.ContextOpts.Config = b
	c.ContextOpts.Hooks = append(c.ContextOpts.Hooks, &UiHook{Ui: c.Ui})
	c.ContextOpts.State = state
	ctx := terraform.NewContext(c.ContextOpts)
	if !validateContext(ctx, c.Ui) {
		return 1
	}

	if refresh {
		if _, err := ctx.Refresh(); err != nil {
			c.Ui.Error(fmt.Sprintf("Error refreshing state: %s", err))
			return 1
		}
	}

	plan, err := ctx.Plan(&terraform.PlanOpts{Destroy: destroy})
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error running plan: %s", err))
		return 1
	}

	if plan.Diff.Empty() {
		c.Ui.Output("No changes. Infrastructure is up-to-date.")
		return 0
	}

	c.Ui.Output(strings.TrimSpace(plan.String()))

	if outPath != "" {
		log.Printf("[INFO] Writing plan output to: %s", outPath)
		f, err := os.Create(outPath)
		if err == nil {
			defer f.Close()
			err = terraform.WritePlan(plan, f)
		}
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error writing plan file: %s", err))
			return 1
		}
	}

	return 0
}

func (c *PlanCommand) Help() string {
	helpText := `
Usage: terraform plan [options] [terraform.tf]

  Shows the differences between the Terraform configuration and
  the actual state of an infrastructure.

Options:

  -destroy            If set, a plan will be generated to destroy all resources
                      managed by the given configuration and state.

  -out=path           Write a plan file to the given path. This can be used as
                      input to the "apply" command.

  -refresh=true       Update state prior to checking for differences.

  -state=statefile    Path to a Terraform state file to use to look
                      up Terraform-managed resources.

`
	return strings.TrimSpace(helpText)
}

func (c *PlanCommand) Synopsis() string {
	return "Show changes between Terraform config and infrastructure"
}
