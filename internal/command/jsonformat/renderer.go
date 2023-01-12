package jsonformat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/mitchellh/colorstring"

	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
	"github.com/hashicorp/terraform/internal/command/jsonplan"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/terminal"
)

type RendererOpt int

const (
	detectedDrift  string = "drift"
	proposedChange string = "change"

	Errored RendererOpt = iota
	CanNotApply
)

type Plan struct {
	PlanFormatVersion string                     `json:"plan_format_version"`
	OutputChanges     map[string]jsonplan.Change `json:"output_changes"`
	ResourceChanges   []jsonplan.ResourceChange  `json:"resource_changes"`
	ResourceDrift     []jsonplan.ResourceChange  `json:"resource_drift"`

	ProviderFormatVersion string                            `json:"provider_format_version"`
	ProviderSchemas       map[string]*jsonprovider.Provider `json:"provider_schemas"`
}

type Renderer struct {
	Streams  *terminal.Streams
	Colorize *colorstring.Colorize

	RunningInAutomation bool
}

func (r Renderer) RenderHumanPlan(plan Plan, mode plans.Mode, opts ...RendererOpt) {
	// TODO(liamcervante): Tidy up this detection of version differences, we
	// should only report warnings when the plan is generated using a newer
	// version then we are executing. We could also look into major vs minor
	// version differences. This should work for alpha testing in the meantime.
	if plan.PlanFormatVersion != jsonplan.FormatVersion || plan.ProviderFormatVersion != jsonprovider.FormatVersion {
		r.Streams.Println(format.WordWrap(
			r.Colorize.Color("\n[bold][red]Warning:[reset][bold] This plan was generated using a different version of Terraform, the diff presented here maybe missing representations of recent features."),
			r.Streams.Stdout.Columns()))
	}

	checkOpts := func(target RendererOpt) bool {
		for _, opt := range opts {
			if opt == target {
				return true
			}
		}
		return false
	}

	diffs := precomputeDiffs(plan)
	haveRefreshChanges := r.renderHumanDiffDrift(diffs, mode)

	willPrintResourceChanges := false
	counts := make(map[plans.Action]int)
	var changes []diff
	for _, diff := range diffs.changes {
		action := jsonplan.UnmarshalActions(diff.change.Change.Actions)
		if action == plans.NoOp && !diff.Moved() {
			// Don't show anything for NoOp changes.
			continue
		}
		if action == plans.Delete && diff.change.Mode != "managed" {
			// Don't render anything for deleted data sources.
			continue
		}

		changes = append(changes, diff)
		willPrintResourceChanges = true

		// Don't count move-only changes
		if action != plans.NoOp {
			counts[action]++
		}
	}

	if len(changes) == 0 && len(diffs.outputs) == 0 {
		// If we didn't find any changes to report at all then this is a
		// "No changes" plan. How we'll present this depends on whether
		// the plan is "applyable" and, if so, whether it had refresh changes
		// that we already would've presented above.

		if checkOpts(Errored) {
			if haveRefreshChanges {
				r.Streams.Print(format.HorizontalRule(r.Colorize, r.Streams.Stdout.Columns()))
				r.Streams.Println()
			}
			r.Streams.Print(
				r.Colorize.Color("\n[reset][bold][red]Planning failed.[reset][bold] Terraform encountered an error while generating this plan.[reset]\n\n"),
			)
		} else {
			switch mode {
			case plans.RefreshOnlyMode:
				if haveRefreshChanges {
					// We already generated a sufficient prompt about what will
					// happen if applying this change above, so we don't need to
					// say anything more.
					return
				}

				r.Streams.Print(r.Colorize.Color("\n[reset][bold][green]No changes.[reset][bold] Your infrastructure still matches the configuration.[reset]\n\n"))
				r.Streams.Println(format.WordWrap(
					"Terraform has checked that the real remote objects still match the result of your most recent changes, and found no differences.",
					r.Streams.Stdout.Columns()))
			case plans.DestroyMode:
				if haveRefreshChanges {
					r.Streams.Print(format.HorizontalRule(r.Colorize, r.Streams.Stdout.Columns()))
					fmt.Fprintln(r.Streams.Stdout.File)
				}
				r.Streams.Print(r.Colorize.Color("\n[reset][bold][green]No changes.[reset][bold] No objects need to be destroyed.[reset]\n\n"))
				r.Streams.Println(format.WordWrap(
					"Either you have not created any objects yet or the existing objects were already deleted outside of Terraform.",
					r.Streams.Stdout.Columns()))
			default:
				if haveRefreshChanges {
					r.Streams.Print(format.HorizontalRule(r.Colorize, r.Streams.Stdout.Columns()))
					r.Streams.Println("")
				}
				r.Streams.Print(
					r.Colorize.Color("\n[reset][bold][green]No changes.[reset][bold] Your infrastructure matches the configuration.[reset]\n\n"),
				)

				if haveRefreshChanges {
					if !checkOpts(CanNotApply) {
						// In this case, applying this plan will not change any
						// remote objects but _will_ update the state to match what
						// we detected during refresh, so we'll reassure the user
						// about that.
						r.Streams.Println(format.WordWrap(
							"Your configuration already matches the changes detected above, so applying this plan will only update the state to include the changes detected above and won't change any real infrastructure.",
							r.Streams.Stdout.Columns(),
						))
					} else {
						// In this case we detected changes during refresh but this isn't
						// a planning mode where we consider those to be applyable. The
						// user must re-run in refresh-only mode in order to update the
						// state to match the upstream changes.
						suggestion := "."
						if !r.RunningInAutomation {
							// The normal message includes a specific command line to run.
							suggestion = ":\n  terraform apply -refresh-only"
						}
						r.Streams.Println(format.WordWrap(
							"Your configuration already matches the changes detected above. If you'd like to update the Terraform state to match, create and apply a refresh-only plan"+suggestion,
							r.Streams.Stdout.Columns(),
						))
					}
					return
				}

				// If we get down here then we're just in the simple situation where
				// the plan isn't applyable at all.
				r.Streams.Println(format.WordWrap(
					"Terraform has compared your real infrastructure against your configuration and found no differences, so no changes are needed.",
					r.Streams.Stdout.Columns(),
				))
			}
		}
	}

	if haveRefreshChanges {
		r.Streams.Print(format.HorizontalRule(r.Colorize, r.Streams.Stdout.Columns()))
		r.Streams.Println()
	}

	if willPrintResourceChanges {
		r.Streams.Println("\nTerraform used the selected providers to generate the following execution plan. Resource actions are indicated with the following symbols:")
		if counts[plans.Create] > 0 {
			r.Streams.Println(r.Colorize.Color(actionDescription(plans.Create)))
		}
		if counts[plans.Update] > 0 {
			r.Streams.Println(r.Colorize.Color(actionDescription(plans.Update)))
		}
		if counts[plans.Delete] > 0 {
			r.Streams.Println(r.Colorize.Color(actionDescription(plans.Delete)))
		}
		if counts[plans.DeleteThenCreate] > 0 {
			r.Streams.Println(r.Colorize.Color(actionDescription(plans.DeleteThenCreate)))
		}
		if counts[plans.CreateThenDelete] > 0 {
			r.Streams.Println(r.Colorize.Color(actionDescription(plans.CreateThenDelete)))
		}
		if counts[plans.Read] > 0 {
			r.Streams.Println(r.Colorize.Color(actionDescription(plans.Read)))
		}
	}

	if len(changes) > 0 {
		if checkOpts(Errored) {
			r.Streams.Printf("\nTerraform planned the following actions, but then encountered a problem:\n\n")
		} else {
			r.Streams.Printf("\nTerraform will perform the following actions:\n\n")
		}

		for _, change := range changes {
			diff, render := r.renderHumanDiff(change, proposedChange)
			if render {
				fmt.Fprintln(r.Streams.Stdout.File)
				r.Streams.Println(diff)
			}
		}

		r.Streams.Printf(
			r.Colorize.Color("\n[bold]Plan:[reset] %d to add, %d to change, %d to destroy.\n"),
			counts[plans.Create]+counts[plans.DeleteThenCreate]+counts[plans.CreateThenDelete],
			counts[plans.Update],
			counts[plans.Delete]+counts[plans.DeleteThenCreate]+counts[plans.CreateThenDelete])
	}

	diff := r.renderHumanDiffOutputs(diffs.outputs)
	if len(diff) > 0 {
		r.Streams.Print("\nChanges to Outputs:\n")
		r.Streams.Printf("%s\n", diff)

		if len(counts) == 0 {
			// If we have output changes but not resource changes then we
			// won't have output any indication about the changes at all yet,
			// so we need some extra context about what it would mean to
			// apply a change that _only_ includes output changes.
			r.Streams.Println(format.WordWrap(
				"\nYou can apply this plan to save these new output values to the Terraform state, without changing any real infrastructure.",
				r.Streams.Stdout.Columns()))
		}
	}
}

func (r Renderer) renderHumanDiffOutputs(outputs map[string]computed.Diff) string {
	var rendered []string

	var keys []string
	for key := range outputs {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		output := outputs[key]
		if output.Action != plans.NoOp {
			rendered = append(rendered, fmt.Sprintf("%s %s = %s", r.Colorize.Color(format.DiffActionSymbol(output.Action)), key, output.RenderHuman(0, computed.NewRenderHumanOpts(r.Colorize))))
		}
	}
	return strings.Join(rendered, "\n")
}

func (r Renderer) renderHumanDiffDrift(diffs diffs, mode plans.Mode) bool {
	var drs []diff
	if mode == plans.RefreshOnlyMode {
		drs = diffs.drift
	} else {
		for _, dr := range diffs.drift {
			// TODO(liamcervante): Look into if we have to keep filtering resource changes.
			// For now we still want to remove the moved resources from here as
			// they will show up in the regular changes.
			if dr.diff.Action != plans.NoOp {
				drs = append(drs, dr)
			}
		}
	}

	if len(drs) == 0 {
		return false
	}

	if diffs.Empty() && mode != plans.RefreshOnlyMode {
		return false
	}

	r.Streams.Print(r.Colorize.Color("\n[bold][cyan]Note:[reset][bold] Objects have changed outside of Terraform\n"))
	r.Streams.Println()
	r.Streams.Print(format.WordWrap(
		"Terraform detected the following changes made outside of Terraform since the last \"terraform apply\" which may have affected this plan:\n",
		r.Streams.Stdout.Columns()))

	for _, drift := range drs {
		diff, render := r.renderHumanDiff(drift, detectedDrift)
		if render {
			r.Streams.Println()
			r.Streams.Println(diff)
		}
	}

	return true
}

func (r Renderer) renderHumanDiff(diff diff, cause string) (string, bool) {

	// Internally, our computed diffs can't tell the difference between a
	// replace action (eg. CreateThenDestroy, DestroyThenCreate) and a simple
	// update action. So, at the top most level we rely on the action provided
	// by the plan itself instead of what we compute. Nested attributes and
	// blocks however don't have the replace type of actions, so we can trust
	// the computed actions of these.

	action := jsonplan.UnmarshalActions(diff.change.Change.Actions)
	if action == plans.NoOp && (len(diff.change.PreviousAddress) == 0 || diff.change.PreviousAddress == diff.change.Address) {
		// Skip resource changes that have nothing interesting to say.
		return "", false
	}

	var buf bytes.Buffer
	buf.WriteString(r.Colorize.Color(resourceChangeComment(diff.change, action, cause)))
	buf.WriteString(fmt.Sprintf("%s %s %s", r.Colorize.Color(format.DiffActionSymbol(action)), resourceChangeHeader(diff.change), diff.diff.RenderHuman(0, computed.NewRenderHumanOpts(r.Colorize))))
	return buf.String(), true
}

func (r Renderer) RenderLog(message map[string]interface{}) {
	panic("not implemented")
}

func resourceChangeComment(resource jsonplan.ResourceChange, action plans.Action, changeCause string) string {
	var buf bytes.Buffer

	dispAddr := resource.Address
	if len(resource.Deposed) != 0 {
		dispAddr = fmt.Sprintf("%s (deposed object %s)", dispAddr, resource.Deposed)
	}

	switch action {
	case plans.Create:
		buf.WriteString(fmt.Sprintf("[bold]  # %s[reset] will be created", dispAddr))
	case plans.Read:
		buf.WriteString(fmt.Sprintf("[bold]  # %s[reset] will be read during apply", dispAddr))
		switch resource.ActionReason {
		case jsonplan.ResourceInstanceReadBecauseConfigUnknown:
			buf.WriteString("\n  # (config refers to values not yet known)")
		case jsonplan.ResourceInstanceReadBecauseDependencyPending:
			buf.WriteString("\n  # (depends on a resource or a module with changes pending)")
		}
	case plans.Update:
		switch changeCause {
		case proposedChange:
			buf.WriteString(fmt.Sprintf("[bold]  # %s[reset] will be updated in-place", dispAddr))
		case detectedDrift:
			buf.WriteString(fmt.Sprintf("[bold]  # %s[reset] has changed", dispAddr))
		default:
			buf.WriteString(fmt.Sprintf("[bold]  # %s[reset] update (unknown reason %s)", dispAddr, changeCause))
		}
	case plans.CreateThenDelete, plans.DeleteThenCreate:
		switch resource.ActionReason {
		case jsonplan.ResourceInstanceReplaceBecauseTainted:
			buf.WriteString(fmt.Sprintf("[bold]  # %s[reset] is tainted, so must be [bold][red]replaced[reset]", dispAddr))
		case jsonplan.ResourceInstanceReplaceByRequest:
			buf.WriteString(fmt.Sprintf("[bold]  # %s[reset] will be [bold][red]replaced[reset], as requested", dispAddr))
		case jsonplan.ResourceInstanceReplaceByTriggers:
			buf.WriteString(fmt.Sprintf("[bold]  # %s[reset] will be [bold][red]replaced[reset] due to changes in replace_triggered_by", dispAddr))
		default:
			buf.WriteString(fmt.Sprintf("[bold]  # %s[reset] must be [bold][red]replaced[reset]", dispAddr))
		}
	case plans.Delete:
		switch changeCause {
		case proposedChange:
			buf.WriteString(fmt.Sprintf("[bold]  # %s[reset] will be [bold][red]destroyed[reset]", dispAddr))
		case detectedDrift:
			buf.WriteString(fmt.Sprintf("[bold]  # %s[reset] has been deleted", dispAddr))
		default:
			buf.WriteString(fmt.Sprintf("[bold]  # %s[reset] delete (unknown reason %s)", dispAddr, changeCause))
		}
		// We can sometimes give some additional detail about why we're
		// proposing to delete. We show this as additional notes, rather than
		// as additional wording in the main action statement, in an attempt
		// to make the "will be destroyed" message prominent and consistent
		// in all cases, for easier scanning of this often-risky action.
		switch resource.ActionReason {
		case jsonplan.ResourceInstanceDeleteBecauseNoResourceConfig:
			buf.WriteString(fmt.Sprintf("\n  # (because %s.%s is not in configuration)", resource.Type, resource.Name))
		case jsonplan.ResourceInstanceDeleteBecauseNoMoveTarget:
			buf.WriteString(fmt.Sprintf("\n  # (because %s was moved to %s, which is not in configuration)", resource.PreviousAddress, resource.Address))
		case jsonplan.ResourceInstanceDeleteBecauseNoModule:
			// FIXME: Ideally we'd truncate addr.Module to reflect the earliest
			// step that doesn't exist, so it's clearer which call this refers
			// to, but we don't have enough information out here in the UI layer
			// to decide that; only the "expander" in Terraform Core knows
			// which module instance keys are actually declared.
			buf.WriteString(fmt.Sprintf("\n  # (because %s is not in configuration)", resource.ModuleAddress))
		case jsonplan.ResourceInstanceDeleteBecauseWrongRepetition:
			var index interface{}
			if resource.Index != nil {
				if err := json.Unmarshal(resource.Index, &index); err != nil {
					panic(err)
				}
			}

			// We have some different variations of this one
			switch index.(type) {
			case nil:
				buf.WriteString("\n  # (because resource uses count or for_each)")
			case float64:
				buf.WriteString("\n  # (because resource does not use count)")
			case string:
				buf.WriteString("\n  # (because resource does not use for_each)")
			}
		case jsonplan.ResourceInstanceDeleteBecauseCountIndex:
			buf.WriteString(fmt.Sprintf("\n  # (because index [%s] is out of range for count)", resource.Index))
		case jsonplan.ResourceInstanceDeleteBecauseEachKey:
			buf.WriteString(fmt.Sprintf("\n  # (because key [%s] is not in for_each map)", resource.Index))
		}
		if len(resource.Deposed) != 0 {
			// Some extra context about this unusual situation.
			buf.WriteString("\n  # (left over from a partially-failed replacement of this instance)")
		}
	case plans.NoOp:
		if len(resource.PreviousAddress) > 0 && resource.PreviousAddress != resource.Address {
			buf.WriteString(fmt.Sprintf("[bold]  # %s[reset] has moved to [bold]%s[reset]", resource.PreviousAddress, dispAddr))
			break
		}
		fallthrough
	default:
		// should never happen, since the above is exhaustive
		buf.WriteString(fmt.Sprintf("%s has an action the plan renderer doesn't support (this is a bug)", dispAddr))
	}
	buf.WriteString("\n")

	if len(resource.PreviousAddress) > 0 && resource.PreviousAddress != resource.Address && action != plans.NoOp {
		buf.WriteString(fmt.Sprintf("  # [reset](moved from %s)\n", resource.PreviousAddress))
	}

	return buf.String()
}

func resourceChangeHeader(change jsonplan.ResourceChange) string {
	mode := "resource"
	if change.Mode != "managed" {
		mode = "data"
	}
	return fmt.Sprintf("%s \"%s\" \"%s\"", mode, change.Type, change.Name)
}

func actionDescription(action plans.Action) string {
	switch action {
	case plans.Create:
		return "  [green]+[reset] create"
	case plans.Delete:
		return "  [red]-[reset] destroy"
	case plans.Update:
		return "  [yellow]~[reset] update in-place"
	case plans.CreateThenDelete:
		return "[green]+[reset]/[red]-[reset] create replacement and then destroy"
	case plans.DeleteThenCreate:
		return "[red]-[reset]/[green]+[reset] destroy and then create replacement"
	case plans.Read:
		return " [cyan]<=[reset] read (data resources)"
	default:
		panic(fmt.Sprintf("unrecognized change type: %s", action.String()))
	}
}
