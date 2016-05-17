package resource

import (
	"fmt"
	"log"
	"reflect"

	"github.com/davecgh/go-spew/spew"
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

	// Do the import!
	newState, err := ctx.Import(&terraform.ImportOpts{
		// Set the module so that any provider config is loaded
		Module: mod,

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

	// Verify that all the states match
	if step.ImportStateVerify {
		new := newState.RootModule().Resources
		old := state.RootModule().Resources
		for _, r := range new {
			// Find the existing resource
			var oldR *terraform.ResourceState
			for _, r2 := range old {
				if r2.Primary != nil && r2.Primary.ID == r.Primary.ID {
					oldR = r2
					break
				}
			}
			if oldR == nil {
				return state, fmt.Errorf(
					"Failed state verification, resource with ID %s not found",
					r.Primary.ID)
			}

			// Compare their attributes
			actual := r.Primary.Attributes
			expected := oldR.Primary.Attributes
			if !reflect.DeepEqual(actual, expected) {
				// Determine only the different attributes
				for k, v := range expected {
					if av, ok := actual[k]; ok && v == av {
						delete(expected, k)
						delete(actual, k)
					}
				}

				spewConf := spew.NewDefaultConfig()
				spewConf.SortKeys = true
				return state, fmt.Errorf(
					"Attributes not equivalent. Difference is shown below. Top is actual, bottom is expected."+
						"\n\n%s\n\n%s",
					spewConf.Sdump(actual), spewConf.Sdump(expected))
			}
		}
	}

	// Return the old state (non-imported) so we don't change anything.
	return state, nil
}
