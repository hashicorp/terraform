package views

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/command/arguments"
	"github.com/hashicorp/terraform/command/format"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
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
	haveRefreshChanges := renderChangesDetectedByRefresh(plan.PrevRunState, plan.PriorState, schemas, view)
	if haveRefreshChanges {
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
		view.streams.Print(format.HorizontalRule(view.colorize, view.outputColumns()))
		view.streams.Println("")
	}

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

	view.streams.Println(view.colorize.Color(headerBuf.String()))

	view.streams.Printf("Terraform will perform the following actions:\n\n")

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

	// If there is at least one planned change to the root module outputs
	// then we'll render a summary of those too.
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
	if len(changedRootModuleOutputs) > 0 {
		view.streams.Println(
			view.colorize.Color("[reset]\n[bold]Changes to Outputs:[reset]") +
				format.OutputChanges(changedRootModuleOutputs, view.colorize),
		)
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
func renderChangesDetectedByRefresh(before, after *states.State, schemas *terraform.Schemas, view *View) bool {
	if after.ManagedResourcesEqual(before) {
		return false
	}

	view.streams.Print(
		view.colorize.Color("[reset]\n[bold][cyan]Note:[reset][bold] Objects have changed outside of Terraform[reset]\n\n"),
	)
	view.streams.Print(format.WordWrap(
		"Terraform detected the following changes made outside of Terraform since the last \"terraform apply\":\n\n",
		view.outputColumns(),
	))

	for _, bms := range before.Modules {
		for _, brs := range bms.Resources {
			if brs.Addr.Resource.Mode != addrs.ManagedResourceMode {
				continue // only managed resources can "drift"
			}
			addr := brs.Addr
			prs := after.Resource(brs.Addr)

			provider := brs.ProviderConfig.Provider
			providerSchema := schemas.ProviderSchema(provider)
			if providerSchema == nil {
				// Should never happen
				view.streams.Printf("(schema missing for %s)\n", provider)
				continue
			}
			rSchema, _ := providerSchema.SchemaForResourceAddr(addr.Resource)
			if rSchema == nil {
				// Should never happen
				view.streams.Printf("(schema missing for %s)\n", addr)
				continue
			}

			for key, bis := range brs.Instances {
				var pis *states.ResourceInstance
				if prs != nil {
					pis = prs.Instance(key)
				}

				diff := format.ResourceInstanceDrift(
					addr.Instance(key),
					bis, pis,
					rSchema,
					view.colorize,
				)
				if diff != "" {
					view.streams.Print(diff)
				}
			}
		}
	}

	return true
}

const planHeaderIntro = `
Terraform used the selected providers to generate the following execution plan. Resource actions are indicated with the following symbols:
`
