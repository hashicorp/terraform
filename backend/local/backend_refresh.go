package local

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/states/statemgr"
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
	tfCtx, _, opState, contextDiags := b.context(op)
	diags = diags.Append(contextDiags)
	if contextDiags.HasErrors() {
		b.ReportResult(runningOp, diags)
		return
	}

	// Set our state
	runningOp.State = opState.State()
	if !runningOp.State.HasResources() {
		if b.CLI != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Warning,
				"Empty or non-existent state",
				"There are currently no resources tracked in the state, so there is nothing to refresh.",
			))
			b.CLI.Output(b.Colorize().Color(strings.TrimSpace(refreshNoState) + "\n"))
		}
	}

	// Perform the refresh in a goroutine so we can be interrupted
	var newState *states.State
	var refreshDiags tfdiags.Diagnostics
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		newState, refreshDiags = tfCtx.Refresh()
		log.Printf("[INFO] backend/local: refresh calling Refresh")
	}()

	if b.opWait(doneCh, stopCtx, cancelCtx, tfCtx, opState) {
		return
	}

	// write the resulting state to the running op
	runningOp.State = newState
	diags = diags.Append(refreshDiags)
	if refreshDiags.HasErrors() {
		b.ReportResult(runningOp, diags)
		return
	}

	err := statemgr.WriteAndPersist(opState, newState)
	if err != nil {
		diags = diags.Append(errwrap.Wrapf("Failed to write state: {{err}}", err))
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
