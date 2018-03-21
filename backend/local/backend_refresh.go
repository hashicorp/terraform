package local

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
)

func (b *Local) opRefresh(
	stopCtx context.Context,
	cancelCtx context.Context,
	op *backend.Operation,
	runningOp *backend.RunningOperation) {

	var diags tfdiags.Diagnostics

	// Check if our state exists if we're performing a refresh operation. We
	// only do this if we're managing state with this backend.
	if b.Backend == nil {
		if _, err := os.Stat(b.StatePath); err != nil {
			if os.IsNotExist(err) {
				err = nil
			}

			if err != nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Cannot read state file",
					fmt.Sprintf("Failed to read %s: %s", b.StatePath, err),
				))
				b.ReportResult(runningOp, diags)
				return
			}
		}
	}

	// Get our context
	tfCtx, opState, contextDiags := b.context(op)
	diags = diags.Append(contextDiags)
	if contextDiags.HasErrors() {
		b.ReportResult(runningOp, diags)
		return
	}

	// Set our state
	runningOp.State = opState.State()
	if runningOp.State.Empty() || !runningOp.State.HasResources() {
		if b.CLI != nil {
			b.CLI.Output(b.Colorize().Color(
				strings.TrimSpace(refreshNoState) + "\n"))
		}
	}

	// Perform the refresh in a goroutine so we can be interrupted
	var newState *terraform.State
	var refreshErr error
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		newState, refreshErr = tfCtx.Refresh()
		log.Printf("[INFO] backend/local: refresh calling Refresh")
	}()

	if b.opWait(doneCh, stopCtx, cancelCtx, tfCtx, opState) {
		return
	}

	// write the resulting state to the running op
	runningOp.State = newState
	if refreshErr != nil {
		diags = diags.Append(refreshErr)
		b.ReportResult(runningOp, diags)
		return
	}

	// Write and persist the state
	if err := opState.WriteState(newState); err != nil {
		diags = diags.Append(errwrap.Wrapf("Failed to write state: {{err}}", err))
		b.ReportResult(runningOp, diags)
		return
	}
	if err := opState.PersistState(); err != nil {
		diags = diags.Append(errwrap.Wrapf("Failed to save state: {{err}}", err))
		b.ReportResult(runningOp, diags)
		return
	}
}

const refreshNoState = `
[reset][bold][yellow]Empty or non-existent state file.[reset][yellow]

Refresh will do nothing. Refresh does not error or return an erroneous
exit status because many automation scripts use refresh, plan, then apply
and may not have a state file yet for the first run.
`
