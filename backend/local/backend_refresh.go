package local

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/backend"
)

func (b *Local) opRefresh(
	ctx context.Context,
	op *backend.Operation,
	runningOp *backend.RunningOperation) {
	// Check if our state exists if we're performing a refresh operation. We
	// only do this if we're managing state with this backend.
	if b.Backend == nil {
		if _, err := os.Stat(b.StatePath); err != nil {
			if os.IsNotExist(err) {
				runningOp.Err = fmt.Errorf(
					"The Terraform state file for your infrastructure does not\n"+
						"exist. The 'refresh' command only works and only makes sense\n"+
						"when there is existing state that Terraform is managing. Please\n"+
						"double-check the value given below and try again. If you\n"+
						"haven't created infrastructure with Terraform yet, use the\n"+
						"'terraform apply' command.\n\n"+
						"Path: %s",
					b.StatePath)
				return
			}

			runningOp.Err = fmt.Errorf(
				"There was an error reading the Terraform state that is needed\n"+
					"for refreshing. The path and error are shown below.\n\n"+
					"Path: %s\n\nError: %s",
				b.StatePath, err)
			return
		}
	}

	// Get our context
	tfCtx, state, err := b.context(op)
	if err != nil {
		runningOp.Err = err
		return
	}

	// Set our state
	runningOp.State = state.State()

	// Perform operation and write the resulting state to the running op
	newState, err := tfCtx.Refresh()
	runningOp.State = newState
	if err != nil {
		runningOp.Err = errwrap.Wrapf("Error refreshing state: {{err}}", err)
		return
	}

	// Write and persist the state
	if err := state.WriteState(newState); err != nil {
		runningOp.Err = errwrap.Wrapf("Error writing state: {{err}}", err)
		return
	}
	if err := state.PersistState(); err != nil {
		runningOp.Err = errwrap.Wrapf("Error saving state: {{err}}", err)
		return
	}
}
