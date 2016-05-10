package resource

import (
	"log"

	"github.com/hashicorp/terraform/terraform"
)

// testStepImportState runs an imort state test step
func testStepImportState(
	opts terraform.ContextOpts,
	state *terraform.State,
	step TestStep) (*terraform.State, error) {
	// Determine the ID to import
	importId := step.ImportStateId
	if importId == "" {
		resource, err := testResource(step, state)
		if err != nil {
			return state, err
		}

		importId = resource.Primary.ID
	}

	// Setup the context. We initialize with an empty state. We use the
	// full config for provider configurations.
	mod, err := testModule(opts, step)
	if err != nil {
		return state, err
	}

	opts.Module = mod
	opts.State = terraform.NewState()
	ctx, err := terraform.NewContext(&opts)
	if err != nil {
		return state, err
	}

	// TODO: ImportOpts needs a flag to read a config module so it
	// can load our provider config without env vars.

	// Do the import!
	newState, err := ctx.Import(&terraform.ImportOpts{
		Targets: []*terraform.ImportTarget{
			&terraform.ImportTarget{
				Addr: step.ResourceName,
				ID:   importId,
			},
		},
	})
	if err != nil {
		log.Printf("[ERROR] Test: ImportState failure: %s", err)
		return state, err
	}

	// Go through the new state and verify
	if step.ImportStateCheck != nil {
		var states []*terraform.InstanceState
		for _, r := range newState.RootModule().Resources {
			if r.Primary != nil {
				states = append(states, r.Primary)
			}
		}
		if err := step.ImportStateCheck(states); err != nil {
			return state, err
		}
	}

	// Return the old state (non-imported) so we don't change anything.
	return state, nil
}
