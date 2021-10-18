package views

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// The Plan view is used for the plan command.
type Plan interface {
	Operation() Operation
	Hooks() []terraform.Hook

	Diagnostics(diags tfdiags.Diagnostics)
	HelpPrompt()
}

// NewPlan returns an initialized Plan implementation for the given ViewType.
func NewPlan(vt arguments.ViewType, view *View) Plan {
	switch vt {
	case arguments.ViewJSON:
		return &PlanJSON{
			view: NewJSONView(view),
		}
	case arguments.ViewHuman:
		return &PlanHuman{
			view:         view,
			inAutomation: view.RunningInAutomation(),
		}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

// The PlanHuman implementation renders human-readable text logs, suitable for
// a scrolling terminal.
type PlanHuman struct {
	view *View

	inAutomation bool
}

var _ Plan = (*PlanHuman)(nil)

func (v *PlanHuman) Operation() Operation {
	return NewOperation(arguments.ViewHuman, v.inAutomation, v.view)
}

func (v *PlanHuman) Hooks() []terraform.Hook {
	return []terraform.Hook{
		NewUiHook(v.view),
	}
}

func (v *PlanHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *PlanHuman) HelpPrompt() {
	v.view.HelpPrompt("plan")
}

// The PlanJSON implementation renders streaming JSON logs, suitable for
// integrating with other software.
type PlanJSON struct {
	view *JSONView
}

var _ Plan = (*PlanJSON)(nil)

func (v *PlanJSON) Operation() Operation {
	return &OperationJSON{view: v.view}
}

func (v *PlanJSON) Hooks() []terraform.Hook {
	return []terraform.Hook{
		newJSONHook(v.view),
	}
}

func (v *PlanJSON) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *PlanJSON) HelpPrompt() {
}

// The plan renderer is used by the Operation view (for plan and apply
// commands) and the Show view (for the show command).
func renderPlan(plan *plans.Plan, schemas *terraform.Schemas, view *View) {
	// In refresh-only mode, we show all resources marked as drifted,
	// including those which have moved without other changes. In other plan
	// modes, move-only changes will be rendered in the planned changes, so
	// we skip them here.
	var driftedResources []*plans.ResourceInstanceChangeSrc
	if plan.UIMode == plans.RefreshOnlyMode {
		driftedResources = plan.DriftedResources
	} else {
		for _, dr := range plan.DriftedResources {
			if dr.Action != plans.NoOp {
				driftedResources = append(driftedResources, dr)
			}
		}
	}

	haveRefreshChanges := len(driftedResources) > 0
	if haveRefreshChanges {
		renderChangesDetectedByRefresh(driftedResources, schemas, view)
		switch plan.UIMode {
		case plans.RefreshOnlyMode:
			view.streams.Println(format.WordWrap(
				"\nThis is a refresh-only plan, so Terraform will not take any actions to undo these. If you were expecting these changes then you can apply this plan to record the updated values in the Terraform state without changing any remote objects.",
				view.outputColumns(),
			))
		default:
			view.streams.Println(format.WordWrap(
				"\nUnless you have made equivalent changes to your configuration, or ignored the relevant attributes using ignore_changes, the following plan may include actions to undo or respond to these changes.",
				view.outputColumns(),
			))
		}
	}

	counts := map[plans.Action]int{}
	var rChanges []*plans.ResourceInstanceChangeSrc
	for _, change := range plan.Changes.Resources {
		if change.Action == plans.NoOp && !change.Moved() {
			continue // We don't show anything for no-op changes
		}
		if change.Action == plans.Delete && change.Addr.Resource.Resource.Mode == addrs.DataResourceMode {
			// Avoid rendering data sources on deletion
			continue
		}

		rChanges = append(rChanges, change)

		// Don't count move-only changes
		if change.Action != plans.NoOp {
			counts[change.Action]++
		}
	}
	var changedRootModuleOutputs []*plans.OutputChangeSrc
	for _, output := range plan.Changes.Outputs {
		if !output.Addr.Module.IsRoot() {
			continue
		}
		if output.ChangeSrc.Action == plans.NoOp {
			continue
		}
		changedRootModuleOutputs = append(changedRootModuleOutputs, output)
	}

	if len(rChanges) == 0 && len(changedRootModuleOutputs) == 0 {
		// If we didn't find any changes to report at all then this is a
		// "No changes" plan. How we'll present this depends on whether
		// the plan is "applyable" and, if so, whether it had refresh changes
		// that we already would've presented above.

		switch plan.UIMode {
		case plans.RefreshOnlyMode:
			if haveRefreshChanges {
				// We already generated a sufficient prompt about what will
				// happen if applying this change above, so we don't need to
				// say anything more.
				return
			}

			view.streams.Print(
				view.colorize.Color("\n[reset][bold][green]No changes.[reset][bold] Your infrastructure still matches the configuration.[reset]\n\n"),
			)
			view.streams.Println(format.WordWrap(
				"Terraform has checked that the real remote objects still match the result of your most recent changes, and found no differences.",
				view.outputColumns(),
			))

		case plans.DestroyMode:
			if haveRefreshChanges {
				view.streams.Print(format.HorizontalRule(view.colorize, view.outputColumns()))
				view.streams.Println("")
			}
			view.streams.Print(
				view.colorize.Color("\n[reset][bold][green]No changes.[reset][bold] No objects need to be destroyed.[reset]\n\n"),
			)
			view.streams.Println(format.WordWrap(
				"Either you have not created any objects yet or the existing objects were already deleted outside of Terraform.",
				view.outputColumns(),
			))

		default:
			if haveRefreshChanges {
				view.streams.Print(format.HorizontalRule(view.colorize, view.outputColumns()))
				view.streams.Println("")
			}
			view.streams.Print(
				view.colorize.Color("\n[reset][bold][green]No changes.[reset][bold] Your infrastructure matches the configuration.[reset]\n\n"),
			)

			if haveRefreshChanges && !plan.CanApply() {
				if plan.CanApply() {
					// In this case, applying this plan will not change any
					// remote objects but _will_ update the state to match what
					// we detected during refresh, so we'll reassure the user
					// about that.
					view.streams.Println(format.WordWrap(
						"Your configuration already matches the changes detected above, so applying this plan will only update the state to include the changes detected above and won't change any real infrastructure.",
						view.outputColumns(),
					))
				} else {
					// In this case we detected changes during refresh but this isn't
					// a planning mode where we consider those to be applyable. The
					// user must re-run in refresh-only mode in order to update the
					// state to match the upstream changes.
					suggestion := "."
					if !view.runningInAutomation {
						// The normal message includes a specific command line to run.
						suggestion = ":\n  terraform apply -refresh-only"
					}
					view.streams.Println(format.WordWrap(
						"Your configuration already matches the changes detected above. If you'd like to update the Terraform state to match, create and apply a refresh-only plan"+suggestion,
						view.outputColumns(),
					))
				}
				return
			}

			// If we get down here then we're just in the simple situation where
			// the plan isn't applyable at all.
			view.streams.Println(format.WordWrap(
				"Terraform has compared your real infrastructure against your configuration and found no differences, so no changes are needed.",
				view.outputColumns(),
			))
		}
		return
	}
	if haveRefreshChanges {
		view.streams.Print(format.HorizontalRule(view.colorize, view.outputColumns()))
		view.streams.Println("")
	}

	if len(counts) > 0 {
		headerBuf := &bytes.Buffer{}
		fmt.Fprintf(headerBuf, "\n%s\n", strings.TrimSpace(format.WordWrap(planHeaderIntro, view.outputColumns())))
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

		view.streams.Print(view.colorize.Color(headerBuf.String()))
	}

	if len(rChanges) > 0 {
		view.streams.Printf("\nTerraform will perform the following actions:\n\n")

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
			if rcs.Action == plans.NoOp && !rcs.Moved() {
				continue
			}

			providerSchema := schemas.ProviderSchema(rcs.ProviderAddr.Provider)
			if providerSchema == nil {
				// Should never happen
				view.streams.Printf("(schema missing for %s)\n\n", rcs.ProviderAddr)
				continue
			}
			rSchema, _ := providerSchema.SchemaForResourceAddr(rcs.Addr.Resource.Resource)
			if rSchema == nil {
				// Should never happen
				view.streams.Printf("(schema missing for %s)\n\n", rcs.Addr)
				continue
			}

			view.streams.Println(format.ResourceChange(
				rcs,
				rSchema,
				view.colorize,
				format.DiffLanguageProposedChange,
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
		view.streams.Printf(
			view.colorize.Color("[reset][bold]Plan:[reset] %d to add, %d to change, %d to destroy.\n"),
			stats[plans.Create], stats[plans.Update], stats[plans.Delete],
		)
	}

	// If there is at least one planned change to the root module outputs
	// then we'll render a summary of those too.
	if len(changedRootModuleOutputs) > 0 {
		view.streams.Println(
			view.colorize.Color("[reset]\n[bold]Changes to Outputs:[reset]") +
				format.OutputChanges(changedRootModuleOutputs, view.colorize),
		)

		if len(counts) == 0 {
			// If we have output changes but not resource changes then we
			// won't have output any indication about the changes at all yet,
			// so we need some extra context about what it would mean to
			// apply a change that _only_ includes output changes.
			view.streams.Println(format.WordWrap(
				"\nYou can apply this plan to save these new output values to the Terraform state, without changing any real infrastructure.",
				view.outputColumns(),
			))
		}
	}
}

// renderChangesDetectedByRefresh is a part of renderPlan that generates
// the note about changes detected by refresh (sometimes considered as "drift").
//
// It will only generate output if there's at least one difference detected.
// Otherwise, it will produce nothing at all. To help the caller recognize
// those two situations incase subsequent output must accommodate it,
// renderChangesDetectedByRefresh returns true if it produced at least one
// line of output, and guarantees to always produce whole lines terminated
// by newline characters.
func renderChangesDetectedByRefresh(drs []*plans.ResourceInstanceChangeSrc, schemas *terraform.Schemas, view *View) {
	view.streams.Print(
		view.colorize.Color("[reset]\n[bold][cyan]Note:[reset][bold] Objects have changed outside of Terraform[reset]\n\n"),
	)
	view.streams.Print(format.WordWrap(
		"Terraform detected the following changes made outside of Terraform since the last \"terraform apply\":\n\n",
		view.outputColumns(),
	))

	// Note: we're modifying the backing slice of this plan object in-place
	// here. The ordering of resource changes in a plan is not significant,
	// but we can only do this safely here because we can assume that nobody
	// is concurrently modifying our changes while we're trying to print it.
	sort.Slice(drs, func(i, j int) bool {
		iA := drs[i].Addr
		jA := drs[j].Addr
		if iA.String() == jA.String() {
			return drs[i].DeposedKey < drs[j].DeposedKey
		}
		return iA.Less(jA)
	})

	for _, rcs := range drs {
		providerSchema := schemas.ProviderSchema(rcs.ProviderAddr.Provider)
		if providerSchema == nil {
			// Should never happen
			view.streams.Printf("(schema missing for %s)\n\n", rcs.ProviderAddr)
			continue
		}
		rSchema, _ := providerSchema.SchemaForResourceAddr(rcs.Addr.Resource.Resource)
		if rSchema == nil {
			// Should never happen
			view.streams.Printf("(schema missing for %s)\n\n", rcs.Addr)
			continue
		}

		view.streams.Println(format.ResourceChange(
			rcs,
			rSchema,
			view.colorize,
			format.DiffLanguageDetectedDrift,
		))
	}
}

const planHeaderIntro = `
Terraform used the selected providers to generate the following execution plan. Resource actions are indicated with the following symbols:
`
