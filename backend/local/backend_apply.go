package local

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/command/clistate"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
)

func (b *Local) opApply(
	ctx context.Context,
	op *backend.Operation,
	runningOp *backend.RunningOperation) {
	log.Printf("[INFO] backend/local: starting Apply operation")

	// If we have a nil module at this point, then set it to an empty tree
	// to avoid any potential crashes.
	if op.Plan == nil && op.Module == nil && !op.Destroy {
		runningOp.Err = fmt.Errorf(strings.TrimSpace(applyErrNoConfig))
		return
	}

	// If we have a nil module at this point, then set it to an empty tree
	// to avoid any potential crashes.
	if op.Module == nil {
		op.Module = module.NewEmptyTree()
	}

	// Setup our count hook that keeps track of resource changes
	countHook := new(CountHook)
	stateHook := new(StateHook)
	if b.ContextOpts == nil {
		b.ContextOpts = new(terraform.ContextOpts)
	}
	old := b.ContextOpts.Hooks
	defer func() { b.ContextOpts.Hooks = old }()
	b.ContextOpts.Hooks = append(b.ContextOpts.Hooks, countHook, stateHook)

	// Get our context
	tfCtx, opState, err := b.context(op)
	if err != nil {
		runningOp.Err = err
		return
	}

	if op.LockState {
		lockCtx, cancel := context.WithTimeout(ctx, op.StateLockTimeout)
		defer cancel()

		lockInfo := state.NewLockInfo()
		lockInfo.Operation = op.Type.String()
		lockID, err := clistate.Lock(lockCtx, opState, lockInfo, b.CLI, b.Colorize())
		if err != nil {
			runningOp.Err = errwrap.Wrapf("Error locking state: {{err}}", err)
			return
		}

		defer func() {
			if err := clistate.Unlock(opState, lockID, b.CLI, b.Colorize()); err != nil {
				runningOp.Err = multierror.Append(runningOp.Err, err)
			}
		}()
	}

	// Setup the state
	runningOp.State = tfCtx.State()

	// If we weren't given a plan, then we refresh/plan
	if op.Plan == nil {
		// If we're refreshing before apply, perform that
		if op.PlanRefresh {
			log.Printf("[INFO] backend/local: apply calling Refresh")
			_, err := tfCtx.Refresh()
			if err != nil {
				runningOp.Err = errwrap.Wrapf("Error refreshing state: {{err}}", err)
				return
			}
		}

		// Perform the plan
		log.Printf("[INFO] backend/local: apply calling Plan")
		if _, err := tfCtx.Plan(); err != nil {
			runningOp.Err = errwrap.Wrapf("Error running plan: {{err}}", err)
			return
		}
	}

	// Setup our hook for continuous state updates
	stateHook.State = opState

	// Start the apply in a goroutine so that we can be interrupted.
	var applyState *terraform.State
	var applyErr error
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		applyState, applyErr = tfCtx.Apply()

		/*
			// Record any shadow errors for later
			if err := ctx.ShadowError(); err != nil {
				shadowErr = multierror.Append(shadowErr, multierror.Prefix(
					err, "apply operation:"))
			}
		*/
	}()

	// Wait for the apply to finish or for us to be interrupted so
	// we can handle it properly.
	err = nil
	select {
	case <-ctx.Done():
		if b.CLI != nil {
			b.CLI.Output("Interrupt received. Gracefully shutting down...")
		}

		// Stop execution
		go tfCtx.Stop()

		// Wait for completion still
		<-doneCh
	case <-doneCh:
	}

	// Store the final state
	runningOp.State = applyState

	// Persist the state
	if err := opState.WriteState(applyState); err != nil {
		runningOp.Err = fmt.Errorf("Failed to save state: %s", err)
		return
	}
	if err := opState.PersistState(); err != nil {
		runningOp.Err = fmt.Errorf("Failed to save state: %s", err)
		return
	}

	if applyErr != nil {
		runningOp.Err = fmt.Errorf(
			"Error applying plan:\n\n"+
				"%s\n\n"+
				"Terraform does not automatically rollback in the face of errors.\n"+
				"Instead, your Terraform state file has been partially updated with\n"+
				"any resources that successfully completed. Please address the error\n"+
				"above and apply again to incrementally change your infrastructure.",
			multierror.Flatten(applyErr))
		return
	}

	// If we have a UI, output the results
	if b.CLI != nil {
		if op.Destroy {
			b.CLI.Output(b.Colorize().Color(fmt.Sprintf(
				"[reset][bold][green]\n"+
					"Destroy complete! Resources: %d destroyed.",
				countHook.Removed)))
		} else {
			b.CLI.Output(b.Colorize().Color(fmt.Sprintf(
				"[reset][bold][green]\n"+
					"Apply complete! Resources: %d added, %d changed, %d destroyed.",
				countHook.Added,
				countHook.Changed,
				countHook.Removed)))
		}

		if countHook.Added > 0 || countHook.Changed > 0 {
			b.CLI.Output(b.Colorize().Color(fmt.Sprintf(
				"[reset]\n"+
					"The state of your infrastructure has been saved to the path\n"+
					"below. This state is required to modify and destroy your\n"+
					"infrastructure, so keep it safe. To inspect the complete state\n"+
					"use the `terraform show` command.\n\n"+
					"State path: %s",
				b.StateOutPath)))
		}
	}
}

const applyErrNoConfig = `
No configuration files found!

Apply requires configuration to be present. Applying without a configuration
would mark everything for destruction, which is normally not what is desired.
If you would like to destroy everything, please run 'terraform destroy' instead
which does not require any configuration files.
`
