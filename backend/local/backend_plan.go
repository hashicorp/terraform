package local

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/command/format"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/plans/planfile"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/states/statemgr"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
)

func (b *Local) opPlan(
	stopCtx context.Context,
	cancelCtx context.Context,
	op *backend.Operation,
	runningOp *backend.RunningOperation) {

	log.Printf("[INFO] backend/local: starting Plan operation")

	var diags tfdiags.Diagnostics

	if op.PlanFile != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Can't re-plan a saved plan",
			"The plan command was given a saved plan file as its input. This command generates "+
				"a new plan, and so it requires a configuration directory as its argument.",
		))
		b.ReportResult(runningOp, diags)
		return
	}

	// Local planning requires a config, unless we're planning to destroy.
	if !op.Destroy && !op.HasConfig() {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"No configuration files",
			"Plan requires configuration to be present. Planning without a configuration would "+
				"mark everything for destruction, which is normally not what is desired. If you "+
				"would like to destroy everything, run plan with the -destroy option. Otherwise, "+
				"create a Terraform configuration file (.tf file) and try again.",
		))
		b.ReportResult(runningOp, diags)
		return
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
	tfCtx, configSnap, opState, ctxDiags := b.context(op)
	diags = diags.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		b.ReportResult(runningOp, diags)
		return
	}

	// Setup the state
	runningOp.State = tfCtx.State()

	// If we're refreshing before plan, perform that
	baseState := runningOp.State
	if op.PlanRefresh {
		log.Printf("[INFO] backend/local: plan calling Refresh")

		if b.CLI != nil {
			b.CLI.Output(b.Colorize().Color(strings.TrimSpace(planRefreshing) + "\n"))
		}

		refreshedState, err := tfCtx.Refresh()
		if err != nil {
			diags = diags.Append(err)
			b.ReportResult(runningOp, diags)
			return
		}
		baseState = refreshedState // plan will be relative to our refreshed state
		if b.CLI != nil {
			b.CLI.Output("\n------------------------------------------------------------------------")
		}
	}

	// Perform the plan in a goroutine so we can be interrupted
	var plan *plans.Plan
	var planDiags tfdiags.Diagnostics
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		log.Printf("[INFO] backend/local: plan calling Plan")
		plan, planDiags = tfCtx.Plan()
	}()

	if b.opWait(doneCh, stopCtx, cancelCtx, tfCtx, opState) {
		// If we get in here then the operation was cancelled, which is always
		// considered to be a failure.
		log.Printf("[INFO] backend/local: plan operation was force-cancelled by interrupt")
		runningOp.Result = backend.OperationFailure
		return
	}
	log.Printf("[INFO] backend/local: plan operation completed")

	diags = diags.Append(planDiags)
	if planDiags.HasErrors() {
		b.ReportResult(runningOp, diags)
		return
	}
	// Record state
	runningOp.PlanEmpty = plan.Changes.Empty()

	// Save the plan to disk
	if path := op.PlanOutPath; path != "" {
		if op.PlanOutBackend == nil {
			// This is always a bug in the operation caller; it's not valid
			// to set PlanOutPath without also setting PlanOutBackend.
			diags = diags.Append(fmt.Errorf(
				"PlanOutPath set without also setting PlanOutBackend (this is a bug in Terraform)"),
			)
			b.ReportResult(runningOp, diags)
			return
		}
		plan.Backend = *op.PlanOutBackend

		// We may have updated the state in the refresh step above, but we
		// will freeze that updated state in the plan file for now and
		// only write it if this plan is subsequently applied.
		plannedStateFile := statemgr.PlannedStateUpdate(opState, baseState)

		log.Printf("[INFO] backend/local: writing plan output to: %s", path)
		err := planfile.Create(path, configSnap, plannedStateFile, plan)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to write plan file",
				fmt.Sprintf("The plan file could not be written: %s.", err),
			))
			b.ReportResult(runningOp, diags)
			return
		}
	}

	// Perform some output tasks if we have a CLI to output to.
	if b.CLI != nil {
		schemas := tfCtx.Schemas()

		if plan.Changes.Empty() {
			b.CLI.Output("\n" + b.Colorize().Color(strings.TrimSpace(planNoChanges)))
			return
		}

		b.renderPlan(plan, baseState, schemas)

		// If we've accumulated any warnings along the way then we'll show them
		// here just before we show the summary and next steps. If we encountered
		// errors then we would've returned early at some other point above.
		b.ShowDiagnostics(diags)

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

func (b *Local) renderPlan(plan *plans.Plan, state *states.State, schemas *terraform.Schemas) {
	counts := map[plans.Action]int{}
	var rChanges []*plans.ResourceInstanceChangeSrc
	for _, change := range plan.Changes.Resources {
		if change.Action == plans.Delete && change.Addr.Resource.Resource.Mode == addrs.DataResourceMode {
			// Avoid rendering data sources on deletion
			continue
		}

		rChanges = append(rChanges, change)
		counts[change.Action]++
	}

	headerBuf := &bytes.Buffer{}
	fmt.Fprintf(headerBuf, "\n%s\n", strings.TrimSpace(planHeaderIntro))
	if counts[plans.Create] > 0 {
		fmt.Fprintf(headerBuf, "%s create\n", format.DiffActionSymbol(plans.Create))
	}
	if counts[plans.Update] > 0 {
		fmt.Fprintf(headerBuf, "%s update in-place\n", format.DiffActionSymbol(plans.Update))
	}
	if counts[plans.Delete] > 0 {
		fmt.Fprintf(headerBuf, "%s destroy\n", format.DiffActionSymbol(plans.Delete))
	}
	if counts[plans.DeleteThenCreate] > 0 {
		fmt.Fprintf(headerBuf, "%s destroy and then create replacement\n", format.DiffActionSymbol(plans.DeleteThenCreate))
	}
	if counts[plans.CreateThenDelete] > 0 {
		fmt.Fprintf(headerBuf, "%s create replacement and then destroy\n", format.DiffActionSymbol(plans.CreateThenDelete))
	}
	if counts[plans.Read] > 0 {
		fmt.Fprintf(headerBuf, "%s read (data resources)\n", format.DiffActionSymbol(plans.Read))
	}

	b.CLI.Output(b.Colorize().Color(headerBuf.String()))

	b.CLI.Output("Terraform will perform the following actions:\n")

	// Note: we're modifying the backing slice of this plan object in-place
	// here. The ordering of resource changes in a plan is not significant,
	// but we can only do this safely here because we can assume that nobody
	// is concurrently modifying our changes while we're trying to print it.
	sort.Slice(rChanges, func(i, j int) bool {
		iA := rChanges[i].Addr
		jA := rChanges[j].Addr
		if iA.String() == jA.String() {
			return rChanges[i].DeposedKey < rChanges[j].DeposedKey
		}
		return iA.Less(jA)
	})

	for _, rcs := range rChanges {
		if rcs.Action == plans.NoOp {
			continue
		}
		providerSchema := schemas.ProviderSchema(rcs.ProviderAddr.ProviderConfig.Type)
		if providerSchema == nil {
			// Should never happen
			b.CLI.Output(fmt.Sprintf("(schema missing for %s)\n", rcs.ProviderAddr))
			continue
		}
		rSchema, _ := providerSchema.SchemaForResourceAddr(rcs.Addr.Resource.Resource)
		if rSchema == nil {
			// Should never happen
			b.CLI.Output(fmt.Sprintf("(schema missing for %s)\n", rcs.Addr))
			continue
		}

		// check if the change is due to a tainted resource
		tainted := false
		if !state.Empty() {
			if is := state.ResourceInstance(rcs.Addr); is != nil {
				if obj := is.GetGeneration(rcs.DeposedKey.Generation()); obj != nil {
					tainted = obj.Status == states.ObjectTainted
				}
			}
		}

		b.CLI.Output(format.ResourceChange(
			rcs,
			tainted,
			rSchema,
			b.CLIColor,
		))
	}

	// stats is similar to counts above, but:
	// - it considers only resource changes
	// - it simplifies "replace" into both a create and a delete
	stats := map[plans.Action]int{}
	for _, change := range rChanges {
		switch change.Action {
		case plans.CreateThenDelete, plans.DeleteThenCreate:
			stats[plans.Create]++
			stats[plans.Delete]++
		default:
			stats[change.Action]++
		}
	}
	b.CLI.Output(b.Colorize().Color(fmt.Sprintf(
		"[reset][bold]Plan:[reset] "+
			"%d to add, %d to change, %d to destroy.",
		stats[plans.Create], stats[plans.Update], stats[plans.Delete],
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
