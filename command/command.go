package command

import (
	"fmt"
	"os"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

// DefaultStateFilename is the default filename used for the state file.
const DefaultStateFilename = "terraform.tfstate"

func ContextArg(
	path string,
	statePath string,
	opts *terraform.ContextOpts) (*terraform.Context, error) {
	// First try to just read the plan directly from the path given.
	f, err := os.Open(path)
	if err == nil {
		plan, err := terraform.ReadPlan(f)
		f.Close()
		if err == nil {
			return plan.Context(opts), nil
		}
	}

	if statePath != "" {
		if _, err := os.Stat(statePath); err != nil {
			return nil, fmt.Errorf(
				"There was an error reading the state file. The path\n"+
					"and error are shown below. If you're trying to build a\n"+
					"brand new infrastructure, explicitly pass the '-init'\n"+
					"flag to Terraform to tell it it is okay to build new\n"+
					"state.\n\n"+
					"Path: %s\n"+
					"Error: %s",
				statePath,
				err)
		}
	}

	// Load up the state
	var state *terraform.State
	if statePath != "" {
		f, err := os.Open(statePath)
		if err == nil {
			state, err = terraform.ReadState(f)
			f.Close()
		}

		if err != nil {
			return nil, fmt.Errorf("Error loading state: %s", err)
		}
	}

	config, err := config.LoadDir(path)
	if err != nil {
		return nil, fmt.Errorf("Error loading config: %s", err)
	}
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("Error validating config: %s", err)
	}

	opts.Config = config
	opts.State = state
	ctx := terraform.NewContext(opts)

	if _, err := ctx.Plan(nil); err != nil {
		return nil, fmt.Errorf("Error running plan: %s", err)
	}

	return ctx, nil
}

func validateContext(ctx *terraform.Context, ui cli.Ui) bool {
	if ws, es := ctx.Validate(); len(ws) > 0 || len(es) > 0 {
		ui.Output(
			"There are warnings and/or errors related to your configuration. Please\n" +
				"fix these before continuing.\n")

		if len(ws) > 0 {
			ui.Output("Warnings:\n")
			for _, w := range ws {
				ui.Output(fmt.Sprintf("  * %s", w))
			}

			if len(es) > 0 {
				ui.Output("")
			}
		}

		if len(es) > 0 {
			ui.Output("Errors:\n")
			for _, e := range es {
				ui.Output(fmt.Sprintf("  * %s", e))
			}
		}

		return false
	}

	return true
}
