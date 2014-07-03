package command

import (
	"fmt"
	"os"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
)

func PlanArg(
	path string,
	statePath string,
	tf *terraform.Terraform) (*terraform.Plan, error) {
	// First try to just read the plan directly from the path given.
	f, err := os.Open(path)
	if err == nil {
		plan, err := terraform.ReadPlan(f)
		f.Close()
		if err == nil {
			return plan, nil
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

	config, err := config.Load(path)
	if err != nil {
		return nil, fmt.Errorf("Error loading config: %s", err)
	}
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("Error validating config: %s", err)
	}

	plan, err := tf.Plan(&terraform.PlanOpts{
		Config: config,
		State:  state,
	})
	if err != nil {
		return nil, fmt.Errorf("Error running plan: %s", err)
	}

	return plan, nil
}
