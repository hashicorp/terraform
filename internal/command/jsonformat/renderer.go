package jsonformat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
	"github.com/hashicorp/terraform/internal/command/jsonformat/differ"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/mitchellh/colorstring"

	"github.com/hashicorp/terraform/internal/command/jsonplan"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/terminal"
)

const (
	detectedDrift  string = "drift"
	proposedChange string = "change"
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
}

type diff struct {
	change jsonplan.ResourceChange
	diff   computed.Diff
}

type diffs struct {
	drift   []diff
	changes []diff
	outputs map[string]computed.Diff
}

func (r Renderer) RenderHumanPlan(plan Plan) {
	// TODO(liamcervante): Tidy up this detection of version differences, we
	// should only report warnings when the plan is generated using a newer
	// version then we are executing. We could also look into major vs minor
	// version differences. This should work for alpha testing in the meantime.
	if plan.PlanFormatVersion != jsonplan.FormatVersion || plan.ProviderFormatVersion != jsonprovider.FormatVersion {
		fmt.Fprintln(r.Streams.Stdout.File, r.Colorize.Color("\n[bold][red]Warning:[reset][bold] This plan was generated using a different version of Terraform, the diff presented here maybe missing representations of recent features."))
	}

	diffs := precomputeDiffs(plan)

	willPrintResourceDrift := false
	for _, drift := range diffs.drift {
		diff, render := r.renderHumanDiff(drift, detectedDrift)
		if render {
			if !willPrintResourceDrift {
				fmt.Fprint(r.Streams.Stdout.File, r.Colorize.Color("\n[bold][cyan]Note:[reset][bold] Objects have changed outside of Terraform\n"))
				fmt.Fprintln(r.Streams.Stdout.File)
				fmt.Fprint(r.Streams.Stdout.File, "Terraform detected the following changes made outside of Terraform since the last \"terraform apply\" which may have affected this plan:\n")
			}
			willPrintResourceDrift = true

			fmt.Fprintln(r.Streams.Stdout.File)
			fmt.Fprintln(r.Streams.Stdout.File, r.Colorize.Color(diff))
		}
	}

	willPrintResourceChanges := false
	counts := make(map[plans.Action]int)
	for _, resource := range plan.ResourceChanges {
		action := jsonplan.UnmarshalActions(resource.Change.Actions)
		if action == plans.NoOp {
			// Don't show anything for NoOp changes.
			continue
		}
		if action == plans.Delete && resource.Mode != "managed" {
			// Don't render anything for deleted data sources.
			continue
		}

		willPrintResourceChanges = true
		counts[action]++
	}
	if willPrintResourceChanges {
		fmt.Fprintln(r.Streams.Stdout.File, "\nTerraform used the selected providers to generate the following execution plan. Resource actions are indicated with the following symbols:")
		if counts[plans.Create] > 0 {
			fmt.Fprintln(r.Streams.Stdout.File, r.Colorize.Color(actionDescription(plans.Create)))
		}
		if counts[plans.Update] > 0 {
			fmt.Fprintln(r.Streams.Stdout.File, r.Colorize.Color(actionDescription(plans.Update)))
		}
		if counts[plans.Delete] > 0 {
			fmt.Fprintln(r.Streams.Stdout.File, r.Colorize.Color(actionDescription(plans.Delete)))
		}
		if counts[plans.DeleteThenCreate] > 0 {
			fmt.Fprintln(r.Streams.Stdout.File, r.Colorize.Color(actionDescription(plans.DeleteThenCreate)))
		}
		if counts[plans.CreateThenDelete] > 0 {
			fmt.Fprintln(r.Streams.Stdout.File, r.Colorize.Color(actionDescription(plans.CreateThenDelete)))
		}
		if counts[plans.Read] > 0 {
			fmt.Fprintln(r.Streams.Stdout.File, r.Colorize.Color(actionDescription(plans.Read)))
		}

		fmt.Fprint(r.Streams.Stdout.File, "\nTerraform will perform the following actions:\n")
	}

	for _, change := range diffs.changes {
		diff, render := r.renderHumanDiff(change, proposedChange)
		if render {
			fmt.Fprintln(r.Streams.Stdout.File)
			fmt.Fprintln(r.Streams.Stdout.File, r.Colorize.Color(diff))
		}
	}

	fmt.Fprintln(
		r.Streams.Stdout.File,
		fmt.Sprintf("\nPlan: %d to add, %d to change, %d to destroy.",
			counts[plans.Create]+counts[plans.DeleteThenCreate]+counts[plans.CreateThenDelete],
			counts[plans.Update],
			counts[plans.Delete]+counts[plans.DeleteThenCreate]+counts[plans.CreateThenDelete]))

	diff := r.renderHumanDiffOutputs(diffs.outputs)
	if len(diff) > 0 {
		fmt.Fprint(r.Streams.Stdout.File, "\nChanges to Outputs:\n")
		fmt.Fprintln(r.Streams.Stdout.File, r.Colorize.Color(diff))
	}
}

func (r Renderer) renderHumanDiffOutputs(outputs map[string]computed.Diff) string {
	var buf bytes.Buffer
	for key, output := range outputs {
		if output.Action != plans.NoOp {
			buf.WriteString(r.Colorize.Color(fmt.Sprintf("%s %s = %s\n", format.DiffActionSymbol(output.Action), key, output.RenderHuman(0, computed.RenderHumanOpts{}))))
		}
	}
	return buf.String()
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
	buf.WriteString(r.Colorize.Color(fmt.Sprintf("%s %s %s", format.DiffActionSymbol(action), resourceChangeHeader(diff.change), diff.diff.RenderHuman(0, computed.RenderHumanOpts{}))))
	return buf.String(), true
}

func (r Renderer) RenderLog(message map[string]interface{}) {
	panic("not implemented")
}

func precomputeDiffs(plan Plan) diffs {
	ctx := diffs{
		outputs: make(map[string]computed.Diff),
	}

	for _, drift := range plan.ResourceDrift {
		schema := plan.ProviderSchemas[drift.ProviderName].ResourceSchemas[drift.Type]
		ctx.drift = append(ctx.drift, diff{
			change: drift,
			diff:   differ.FromJsonChange(drift.Change).ComputeDiffForBlock(schema.Block),
		})
	}

	for _, change := range plan.ResourceChanges {
		schema := plan.ProviderSchemas[change.ProviderName].ResourceSchemas[change.Type]
		ctx.changes = append(ctx.changes, diff{
			change: change,
			diff:   differ.FromJsonChange(change.Change).ComputeDiffForBlock(schema.Block),
		})
	}

	for key, output := range plan.OutputChanges {
		ctx.outputs[key] = differ.FromJsonChange(output).ComputeDiffForOutput()
	}

	return ctx
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
