package local

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/mitchellh/colorstring"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/command/format"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/plans/objchange"
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
	// the state was locked during succesfull context creation; unlock the state
	// when the operation completes
	defer func() {
		err := op.StateLocker.Unlock(nil)
		if err != nil {
			b.ShowDiagnostics(err)
			runningOp.Result = backend.OperationFailure
		}
	}()

	// Before we do anything else we'll take a snapshot of the prior state
	// so we can use it for some fixups to our detection of whether the plan
	// includes externally-visible side-effects that need to be applied.
	// (We should be able to remove this once we complete the planned work
	// described in the comment for func planHasSideEffects below.)
	// We go directly to the state manager here because the state inside
	// tfCtx was already implicitly changed by a validation walk inside
	// the b.context method.
	priorState := opState.State().DeepCopy()

	runningOp.State = tfCtx.State()

	// If we're refreshing before plan, perform that
	baseState := runningOp.State
	if op.PlanRefresh {
		log.Printf("[INFO] backend/local: plan calling Refresh")

		if b.CLI != nil {
			b.CLI.Output(b.Colorize().Color(strings.TrimSpace(planRefreshing) + "\n"))
		}

		refreshedState, refreshDiags := tfCtx.Refresh()
		diags = diags.Append(refreshDiags)
		if diags.HasErrors() {
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
	// Record whether this plan includes any side-effects that could be applied.
	runningOp.PlanEmpty = !planHasSideEffects(priorState, plan.Changes)

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

		if runningOp.PlanEmpty {
			b.CLI.Output("\n" + b.Colorize().Color(strings.TrimSpace(planNoChanges)))
			// Even if there are no changes, there still could be some warnings
			b.ShowDiagnostics(diags)
			return
		}

		b.renderPlan(plan, baseState, priorState, schemas)

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

func (b *Local) renderPlan(plan *plans.Plan, baseState *states.State, priorState *states.State, schemas *terraform.Schemas) {
	RenderPlan(plan, baseState, priorState, schemas, b.CLI, b.Colorize())
}

// RenderPlan renders the given plan to the given UI.
//
// This is exported only so that the "terraform show" command can re-use it.
// Ideally it would be somewhere outside of this backend code so that both
// can call into it, but we're leaving it here for now in order to avoid
// disruptive refactoring.
//
// If you find yourself wanting to call this function from a third callsite,
// please consider whether it's time to do the more disruptive refactoring
// so that something other than the local backend package is offering this
// functionality.
//
// The difference between baseState and priorState is that baseState is the
// result of implicitly running refresh (unless that was disabled) while
// priorState is a snapshot of the state as it was before we took any actions
// at all. priorState can optionally be nil if the caller has only a saved
// plan and not the prior state it was built from. In that case, changes to
// output values will not currently be rendered because their prior values
// are currently stored only in the prior state. (see the docstring for
// func planHasSideEffects for why this is and when that might change)
func RenderPlan(plan *plans.Plan, baseState *states.State, priorState *states.State, schemas *terraform.Schemas, ui cli.Ui, colorize *colorstring.Colorize) {
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

	ui.Output(colorize.Color(headerBuf.String()))

	ui.Output("Terraform will perform the following actions:\n")

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

		providerSchema := schemas.ProviderSchema(rcs.ProviderAddr.Provider)
		if providerSchema == nil {
			// Should never happen
			ui.Output(fmt.Sprintf("(schema missing for %s)\n", rcs.ProviderAddr))
			continue
		}
		rSchema, _ := providerSchema.SchemaForResourceAddr(rcs.Addr.Resource.Resource)
		if rSchema == nil {
			// Should never happen
			ui.Output(fmt.Sprintf("(schema missing for %s)\n", rcs.Addr))
			continue
		}

		// check if the change is due to a tainted resource
		tainted := false
		if !baseState.Empty() {
			if is := baseState.ResourceInstance(rcs.Addr); is != nil {
				if obj := is.GetGeneration(rcs.DeposedKey.Generation()); obj != nil {
					tainted = obj.Status == states.ObjectTainted
				}
			}
		}

		ui.Output(format.ResourceChange(
			rcs,
			tainted,
			rSchema,
			colorize,
		))
	}

	// If there is at least one planned change to the root module outputs
	// then we'll render a summary of those too. This is easier said than done
	// because currently output changes are not accurately recorded in
	// plan.Changes.Outputs (see the func planHasSideEffects docstring for why)
	// and so we must use priorState to produce an actually-accurate changeset
	// to display.
	//
	// Some callers (i.e. "terraform show") only have the plan and therefore
	// can't provide the prior state. In that case we'll skip showing the
	// outputs for now, until we can make plan.Changes.Outputs itself be
	// accurate and self-contained.
	outputChangeCount := 0
	if priorState != nil {
		var synthOutputChanges []*plans.OutputChangeSrc
		for _, addr := range allRootModuleOutputs(priorState, plan.Changes) {
			before := cty.NullVal(cty.DynamicPseudoType)
			after := cty.NullVal(cty.DynamicPseudoType)
			sensitive := false
			if changeSrc := plan.Changes.OutputValue(addr); changeSrc != nil {
				sensitive = sensitive || changeSrc.Sensitive
				change, err := changeSrc.Decode()
				if err != nil {
					// It would be very strange to get here because changeSrc was
					// presumably just created by Terraform Core and so should never
					// be invalid.
					panic(fmt.Sprintf("failed to decode change for %s: %s", addr, err))
				}
				after = change.After
			}
			if priorOutputState := priorState.OutputValue(addr); priorOutputState != nil {
				sensitive = sensitive || priorOutputState.Sensitive
				before = priorOutputState.Value
			}

			// We'll now construct ourselves a new, accurate change.
			change := &plans.OutputChange{
				Addr:      addr,
				Sensitive: sensitive,
				Change: plans.Change{
					Action: objchange.ActionForChange(before, after),
					Before: before,
					After:  after,
				},
			}
			if change.Action == plans.NoOp {
				continue // ignore non-changes
			}
			outputChangeCount++
			newChangeSrc, err := change.Encode()
			if err != nil {
				// Again, it would be very strange to see an error here because
				// we've literally just created this value in memory above.
				panic(fmt.Sprintf("failed to encode change for %s: %s", addr, err))
			}
			synthOutputChanges = append(synthOutputChanges, newChangeSrc)
		}
		if outputChangeCount > 0 {
			ui.Output(colorize.Color("[reset][bold]Changes to Outputs:[reset]" + format.OutputChanges(synthOutputChanges, colorize) + "\n"))
		}
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
	ui.Output(colorize.Color(fmt.Sprintf(
		"[reset][bold]Plan:[reset] "+
			"%d to add, %d to change, %d to destroy, %d changes to outputs.",
		stats[plans.Create], stats[plans.Update], stats[plans.Delete],
		outputChangeCount,
	)))
}

// planHasSideEffects determines whether the given planned changeset has
// externally-visible side-effects that warrant giving the user an opportunity
// to apply the plan. If planHasSideEffects returns false, the caller should
// return a "No changes" message and not offer to apply the plan.
//
// This is currently implemented here, rather than in the "terraform" package,
// because with the current separation of the refresh vs. plan walks there is
// never any single point in the "terraform" package where both the prior and
// planned new values for outputs are available at once. We have this out here
// as a temporary workaround for that design problem, with the intent of moving
// this down into the "terraform" package once we've completed some work to
// combine the refresh and plan walks together into a single walk and thus
// that walk will be able to see both the prior and new values for outputs.
func planHasSideEffects(priorState *states.State, changes *plans.Changes) bool {
	if !changes.Empty() {
		// At the time of writing, changes.Empty considers only resource
		// changes because the planned changes for outputs are inaccurate.
		// If we have at least one resource change then we know we have
		// side-effects though, regardless of outputs.
		return true
	}

	// If we get here then there are definitely no resource changes in the plan
	// but we may have some changes to outputs that "changes" hasn't properly
	// captured, because it treats all outputs as being either created or
	// deleted regardless of their prior values. To work around that for now,
	// we'll use priorState to see if those planned changes really are changes.
	for _, addr := range allRootModuleOutputs(priorState, changes) {
		before := cty.NullVal(cty.DynamicPseudoType)
		after := cty.NullVal(cty.DynamicPseudoType)
		if changeSrc := changes.OutputValue(addr); changeSrc != nil {
			change, err := changeSrc.Decode()
			if err != nil {
				// It would be very strange to get here because changeSrc was
				// presumably just created by Terraform Core and so should never
				// be invalid. In this unlikely event, we'll just conservatively
				// assume there is a change.
				return true
			}
			after = change.After
		}
		if priorState != nil {
			if priorOutputState := priorState.OutputValue(addr); priorOutputState != nil {
				before = priorOutputState.Value
			}
		}
		if objchange.ActionForChange(before, after) != plans.NoOp {
			return true
		}
	}

	// If we fall out here then we didn't find any effective changes in the
	// outputs, and we already showed that there were no resource changes, so
	// this plan has no side-effects.
	return false
}

// allRootModuleOutputs is a helper function to produce the union of all
// root module output values across both the given prior state and the given
// changeset. This is to compensate for the fact that the outputs portion of
// a plans.Changes is currently incomplete and inaccurate due to limitations of
// Terraform Core's design; we need to use information from the prior state
// to compensate for those limitations when making decisions based on the
// effective output changes.
func allRootModuleOutputs(priorState *states.State, changes *plans.Changes) []addrs.AbsOutputValue {
	m := make(map[string]addrs.AbsOutputValue)
	if priorState != nil {
		for _, os := range priorState.RootModule().OutputValues {
			m[os.Addr.String()] = os.Addr
		}
	}
	if changes != nil {
		for _, oc := range changes.Outputs {
			if !oc.Addr.Module.IsRoot() {
				continue
			}
			m[oc.Addr.String()] = oc.Addr
		}
	}
	if len(m) == 0 {
		return nil
	}
	ret := make([]addrs.AbsOutputValue, 0, len(m))
	for _, addr := range m {
		ret = append(ret, addr)
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].OutputValue.Name < ret[j].OutputValue.Name
	})
	return ret
}

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
