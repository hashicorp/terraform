package local

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/command/clistate"
	"github.com/hashicorp/terraform/command/format"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
)

func (b *Local) opPlan(
	stopCtx context.Context,
	cancelCtx context.Context,
	op *backend.Operation,
	runningOp *backend.RunningOperation) {
	log.Printf("[INFO] backend/local: starting Plan operation")

	if b.CLI != nil && op.Plan != nil {
		b.CLI.Output(b.Colorize().Color(
			"[reset][bold][yellow]" +
				"The plan command received a saved plan file as input. This command\n" +
				"will output the saved plan. This will not modify the already-existing\n" +
				"plan. If you wish to generate a new plan, please pass in a configuration\n" +
				"directory as an argument.\n\n"))
	}

	// A local plan requires either a plan or a module
	if op.Plan == nil && op.Module == nil && !op.Destroy {
		runningOp.Err = fmt.Errorf(strings.TrimSpace(planErrNoConfig))
		return
	}

	// If we have a nil module at this point, then set it to an empty tree
	// to avoid any potential crashes.
	if op.Module == nil {
		op.Module = module.NewEmptyTree()
	}

	// Setup our count hook that keeps track of resource changes
	countHook := new(CountHook)
	if b.ContextOpts == nil {
		b.ContextOpts = new(terraform.ContextOpts)
	}
	old := b.ContextOpts.Hooks
	defer func() { b.ContextOpts.Hooks = old }()
	b.ContextOpts.Hooks = append(b.ContextOpts.Hooks, countHook)

	// Get our context
	tfCtx, opState, err := b.context(op)
	if err != nil {
		runningOp.Err = err
		return
	}

	if op.LockState {
		lockCtx, cancel := context.WithTimeout(stopCtx, op.StateLockTimeout)
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

	// If we're refreshing before plan, perform that
	if op.PlanRefresh {
		log.Printf("[INFO] backend/local: plan calling Refresh")

		if b.CLI != nil {
			b.CLI.Output(b.Colorize().Color(strings.TrimSpace(planRefreshing) + "\n"))
		}

		_, err := tfCtx.Refresh()
		if err != nil {
			runningOp.Err = errwrap.Wrapf("Error refreshing state: {{err}}", err)
			return
		}
		if b.CLI != nil {
			b.CLI.Output("\n------------------------------------------------------------------------")
		}
	}

	// Perform the plan in a goroutine so we can be interrupted
	var plan *terraform.Plan
	var planErr error
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		log.Printf("[INFO] backend/local: plan calling Plan")
		plan, planErr = tfCtx.Plan()
	}()

	if b.opWait(doneCh, stopCtx, cancelCtx, tfCtx, opState) {
		return
	}

	if planErr != nil {
		runningOp.Err = errwrap.Wrapf("Error running plan: {{err}}", planErr)
		return
	}
	// Record state
	runningOp.PlanEmpty = plan.Diff.Empty()

	// Save the plan to disk
	if path := op.PlanOutPath; path != "" {
		// Write the backend if we have one
		plan.Backend = op.PlanOutBackend

		// This works around a bug (#12871) which is no longer possible to
		// trigger but will exist for already corrupted upgrades.
		if plan.Backend != nil && plan.State != nil {
			plan.State.Remote = nil
		}

		log.Printf("[INFO] backend/local: writing plan output to: %s", path)
		f, err := os.Create(path)
		if err == nil {
			err = terraform.WritePlan(plan, f)
		}
		f.Close()
		if err != nil {
			runningOp.Err = fmt.Errorf("Error writing plan file: %s", err)
			return
		}
	}

	// Perform some output tasks if we have a CLI to output to.
	if b.CLI != nil {
		dispPlan := format.NewPlan(plan)
		if dispPlan.Empty() {
			b.CLI.Output("\n" + b.Colorize().Color(strings.TrimSpace(planNoChanges)))
			return
		}

		b.renderPlan(dispPlan)

		// Give the user some next-steps, unless we're running in an automation
		// tool which is presumed to provide its own UI for further actions.
		if !b.RunningInAutomation {

			b.CLI.Output("\n------------------------------------------------------------------------")

			if path := op.PlanOutPath; path == "" {
				b.CLI.Output(fmt.Sprintf(
					"\n" + strings.TrimSpace(planHeaderNoOutput) + "\n",
				))
			} else {
				b.CLI.Output(fmt.Sprintf(
					"\n"+strings.TrimSpace(planHeaderYesOutput)+"\n",
					path, path,
				))
			}

		}
	}
}

func (b *Local) renderPlan(dispPlan *format.Plan) {

	headerBuf := &bytes.Buffer{}
	fmt.Fprintf(headerBuf, "\n%s\n", strings.TrimSpace(planHeaderIntro))
	counts := dispPlan.ActionCounts()
	if counts[terraform.DiffCreate] > 0 {
		fmt.Fprintf(headerBuf, "%s create\n", format.DiffActionSymbol(terraform.DiffCreate))
	}
	if counts[terraform.DiffUpdate] > 0 {
		fmt.Fprintf(headerBuf, "%s update in-place\n", format.DiffActionSymbol(terraform.DiffUpdate))
	}
	if counts[terraform.DiffDestroy] > 0 {
		fmt.Fprintf(headerBuf, "%s destroy\n", format.DiffActionSymbol(terraform.DiffDestroy))
	}
	if counts[terraform.DiffDestroyCreate] > 0 {
		fmt.Fprintf(headerBuf, "%s destroy and then create replacement\n", format.DiffActionSymbol(terraform.DiffDestroyCreate))
	}
	if counts[terraform.DiffRefresh] > 0 {
		fmt.Fprintf(headerBuf, "%s read (data resources)\n", format.DiffActionSymbol(terraform.DiffRefresh))
	}

	b.CLI.Output(b.Colorize().Color(headerBuf.String()))

	b.CLI.Output("Terraform will perform the following actions:\n")

	b.CLI.Output(dispPlan.Format(b.Colorize()))

	stats := dispPlan.Stats()
	b.CLI.Output(b.Colorize().Color(fmt.Sprintf(
		"[reset][bold]Plan:[reset] "+
			"%d to add, %d to change, %d to destroy.",
		stats.ToAdd, stats.ToChange, stats.ToDestroy,
	)))
}

const planErrNoConfig = `
No configuration files found!

Plan requires configuration to be present. Planning without a configuration
would mark everything for destruction, which is normally not what is desired.
If you would like to destroy everything, please run plan with the "-destroy"
flag or create a single empty configuration file. Otherwise, please create
a Terraform configuration file in the path being executed and try again.
`

const planHeaderIntro = `
An execution plan has been generated and is shown below.
Resource actions are indicated with the following symbols:
`

const planHeaderNoOutput = `
Note: You didn't specify an "-out" parameter to save this plan, so Terraform
can't guarantee that exactly these actions will be performed if
"terraform apply" is subsequently run.
`

const planHeaderYesOutput = `
This plan was saved to: %s

To perform exactly these actions, run the following command to apply:
    terraform apply %q
`

const planNoChanges = `
[reset][bold][green]No changes. Infrastructure is up-to-date.[reset][green]

This means that Terraform did not detect any differences between your
configuration and real physical resources that exist. As a result, no
actions need to be performed.
`

const planRefreshing = `
[reset][bold]Refreshing Terraform state in-memory prior to plan...[reset]
The refreshed state will be used to calculate this plan, but will not be
persisted to local or remote state storage.
`
