// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package jsonformat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
	"github.com/hashicorp/terraform/internal/command/jsonformat/computed/renderers"
	"github.com/hashicorp/terraform/internal/command/jsonplan"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/command/jsonstate"
	"github.com/hashicorp/terraform/internal/plans"
)

const (
	detectedDrift  string = "drift"
	proposedChange string = "change"
)

type Plan struct {
	PlanFormatVersion  string                            `json:"plan_format_version"`
	OutputChanges      map[string]jsonplan.Change        `json:"output_changes,omitempty"`
	ResourceChanges    []jsonplan.ResourceChange         `json:"resource_changes,omitempty"`
	ResourceDrift      []jsonplan.ResourceChange         `json:"resource_drift,omitempty"`
	RelevantAttributes []jsonplan.ResourceAttr           `json:"relevant_attributes,omitempty"`
	DeferredChanges    []jsonplan.DeferredResourceChange `json:"deferred_changes,omitempty"`

	ProviderFormatVersion string                            `json:"provider_format_version"`
	ProviderSchemas       map[string]*jsonprovider.Provider `json:"provider_schemas,omitempty"`
}

func (plan Plan) getSchema(change jsonplan.ResourceChange) *jsonprovider.Schema {
	switch change.Mode {
	case jsonstate.ManagedResourceMode:
		return plan.ProviderSchemas[change.ProviderName].ResourceSchemas[change.Type]
	case jsonstate.DataResourceMode:
		return plan.ProviderSchemas[change.ProviderName].DataSourceSchemas[change.Type]
	default:
		panic("found unrecognized resource mode: " + change.Mode)
	}
}

func (plan Plan) renderHuman(renderer Renderer, mode plans.Mode, opts ...plans.Quality) {
	checkOpts := func(target plans.Quality) bool {
		for _, opt := range opts {
			if opt == target {
				return true
			}
		}
		return false
	}

	diffs := precomputeDiffs(plan, mode)
	haveRefreshChanges := renderHumanDiffDrift(renderer, diffs, mode)

	willPrintResourceChanges := false
	counts := make(map[plans.Action]int)
	importingCount := 0
	var changes []diff
	for _, diff := range diffs.changes {
		action := jsonplan.UnmarshalActions(diff.change.Change.Actions)
		if action == plans.NoOp && !diff.Moved() && !diff.Importing() {
			// Don't show anything for NoOp changes.
			continue
		}
		if action == plans.Delete && diff.change.Mode != jsonstate.ManagedResourceMode {
			// Don't render anything for deleted data sources.
			continue
		}

		changes = append(changes, diff)

		if diff.Importing() {
			importingCount++
		}

		// Don't count move-only changes
		if action != plans.NoOp {
			willPrintResourceChanges = true
			counts[action]++
		}
	}

	// Precompute the outputs early, so we can make a decision about whether we
	// display the "there are no changes messages".
	outputs := renderHumanDiffOutputs(renderer, diffs.outputs)

	if len(changes) == 0 && len(outputs) == 0 {
		// If we didn't find any changes to report at all then this is a
		// "No changes" plan. How we'll present this depends on whether
		// the plan is "applyable" and, if so, whether it had refresh changes
		// that we already would've presented above.

		if checkOpts(plans.Errored) {
			if haveRefreshChanges {
				renderer.Streams.Print(format.HorizontalRule(renderer.Colorize, renderer.Streams.Stdout.Columns()))
				renderer.Streams.Println()
			}
			renderer.Streams.Print(
				renderer.Colorize.Color("\n[reset][bold][red]Planning failed.[reset][bold] Terraform encountered an error while generating this plan.[reset]\n\n"),
			)
		} else if len(diffs.deferred) > 0 {
			// We had no current changes, but deferred changes
			if haveRefreshChanges {
				renderer.Streams.Print(format.HorizontalRule(renderer.Colorize, renderer.Streams.Stdout.Columns()))
				renderer.Streams.Println("")
			}
			renderer.Streams.Print(
				renderer.Colorize.Color("\n[reset][bold][green]No current changes.[reset][bold] This plan requires another plan to be applied first.[reset]\n\n"),
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

				renderer.Streams.Print(renderer.Colorize.Color("\n[reset][bold][green]No changes.[reset][bold] Your infrastructure still matches the configuration.[reset]\n\n"))
				renderer.Streams.Println(format.WordWrap(
					"Terraform has checked that the real remote objects still match the result of your most recent changes, and found no differences.",
					renderer.Streams.Stdout.Columns()))
			case plans.DestroyMode:
				if haveRefreshChanges {
					renderer.Streams.Print(format.HorizontalRule(renderer.Colorize, renderer.Streams.Stdout.Columns()))
					fmt.Fprintln(renderer.Streams.Stdout.File)
				}
				renderer.Streams.Print(renderer.Colorize.Color("\n[reset][bold][green]No changes.[reset][bold] No objects need to be destroyed.[reset]\n\n"))
				renderer.Streams.Println(format.WordWrap(
					"Either you have not created any objects yet or the existing objects were already deleted outside of Terraform.",
					renderer.Streams.Stdout.Columns()))
			default:
				if haveRefreshChanges {
					renderer.Streams.Print(format.HorizontalRule(renderer.Colorize, renderer.Streams.Stdout.Columns()))
					renderer.Streams.Println("")
				}
				renderer.Streams.Print(
					renderer.Colorize.Color("\n[reset][bold][green]No changes.[reset][bold] Your infrastructure matches the configuration.[reset]\n\n"),
				)

				if haveRefreshChanges {
					if !checkOpts(plans.NoChanges) {
						// In this case, applying this plan will not change any
						// remote objects but _will_ update the state to match what
						// we detected during refresh, so we'll reassure the user
						// about that.
						renderer.Streams.Println(format.WordWrap(
							"Your configuration already matches the changes detected above, so applying this plan will only update the state to include the changes detected above and won't change any real infrastructure.",
							renderer.Streams.Stdout.Columns(),
						))
					} else {
						// In this case we detected changes during refresh but this isn't
						// a planning mode where we consider those to be applyable. The
						// user must re-run in refresh-only mode in order to update the
						// state to match the upstream changes.
						suggestion := "."
						if !renderer.RunningInAutomation {
							// The normal message includes a specific command line to run.
							suggestion = ":\n  terraform apply -refresh-only"
						}
						renderer.Streams.Println(format.WordWrap(
							"Your configuration already matches the changes detected above. If you'd like to update the Terraform state to match, create and apply a refresh-only plan"+suggestion,
							renderer.Streams.Stdout.Columns(),
						))
					}
					return
				}

				// If we get down here then we're just in the simple situation where
				// the plan isn't applyable at all.
				renderer.Streams.Println(format.WordWrap(
					"Terraform has compared your real infrastructure against your configuration and found no differences, so no changes are needed.",
					renderer.Streams.Stdout.Columns(),
				))
			}
		}
	}

	if haveRefreshChanges {
		renderer.Streams.Print(format.HorizontalRule(renderer.Colorize, renderer.Streams.Stdout.Columns()))
		renderer.Streams.Println()
	}

	haveDeferredChanges := renderHumanDeferredChanges(renderer, diffs, mode)
	if haveDeferredChanges {
		renderer.Streams.Print(format.HorizontalRule(renderer.Colorize, renderer.Streams.Stdout.Columns()))
		renderer.Streams.Println()
	}

	if willPrintResourceChanges {
		renderer.Streams.Println(format.WordWrap(
			"\nTerraform used the selected providers to generate the following execution plan. Resource actions are indicated with the following symbols:",
			renderer.Streams.Stdout.Columns()))
		if counts[plans.Create] > 0 {
			renderer.Streams.Println(renderer.Colorize.Color(actionDescription(plans.Create)))
		}
		if counts[plans.Update] > 0 {
			renderer.Streams.Println(renderer.Colorize.Color(actionDescription(plans.Update)))
		}
		if counts[plans.Delete] > 0 {
			renderer.Streams.Println(renderer.Colorize.Color(actionDescription(plans.Delete)))
		}
		if counts[plans.DeleteThenCreate] > 0 {
			renderer.Streams.Println(renderer.Colorize.Color(actionDescription(plans.DeleteThenCreate)))
		}
		if counts[plans.CreateThenDelete] > 0 {
			renderer.Streams.Println(renderer.Colorize.Color(actionDescription(plans.CreateThenDelete)))
		}
		if counts[plans.Read] > 0 {
			renderer.Streams.Println(renderer.Colorize.Color(actionDescription(plans.Read)))
		}
	}

	if len(changes) > 0 {
		if checkOpts(plans.Errored) {
			renderer.Streams.Printf("\nTerraform planned the following actions, but then encountered a problem:\n")
		} else {
			renderer.Streams.Printf("\nTerraform will perform the following actions:\n")
		}

		for _, change := range changes {
			diff, render := renderHumanDiff(renderer, change, proposedChange)
			if render {
				fmt.Fprintln(renderer.Streams.Stdout.File)
				renderer.Streams.Println(diff)
			}
		}

		if importingCount > 0 {
			renderer.Streams.Printf(
				renderer.Colorize.Color("\n[bold]Plan:[reset] %d to import, %d to add, %d to change, %d to destroy.\n"),
				importingCount,
				counts[plans.Create]+counts[plans.DeleteThenCreate]+counts[plans.CreateThenDelete],
				counts[plans.Update],
				counts[plans.Delete]+counts[plans.DeleteThenCreate]+counts[plans.CreateThenDelete])
		} else {
			renderer.Streams.Printf(
				renderer.Colorize.Color("\n[bold]Plan:[reset] %d to add, %d to change, %d to destroy.\n"),
				counts[plans.Create]+counts[plans.DeleteThenCreate]+counts[plans.CreateThenDelete],
				counts[plans.Update],
				counts[plans.Delete]+counts[plans.DeleteThenCreate]+counts[plans.CreateThenDelete])
		}
	}

	if len(outputs) > 0 {
		renderer.Streams.Print("\nChanges to Outputs:\n")
		renderer.Streams.Printf("%s\n", outputs)

		if len(counts) == 0 {
			// If we have output changes but not resource changes then we
			// won't have output any indication about the changes at all yet,
			// so we need some extra context about what it would mean to
			// apply a change that _only_ includes output changes.
			renderer.Streams.Println(format.WordWrap(
				"\nYou can apply this plan to save these new output values to the Terraform state, without changing any real infrastructure.",
				renderer.Streams.Stdout.Columns()))
		}
	}
}

func renderHumanDiffOutputs(renderer Renderer, outputs map[string]computed.Diff) string {
	var rendered []string

	var keys []string
	escapedKeys := make(map[string]string)
	var escapedKeyMaxLen int
	for key := range outputs {
		escapedKey := renderers.EnsureValidAttributeName(key)
		keys = append(keys, key)
		escapedKeys[key] = escapedKey
		if len(escapedKey) > escapedKeyMaxLen {
			escapedKeyMaxLen = len(escapedKey)
		}
	}
	sort.Strings(keys)

	for _, key := range keys {
		output := outputs[key]
		if output.Action != plans.NoOp {
			rendered = append(rendered, fmt.Sprintf("%s %-*s = %s", renderer.Colorize.Color(format.DiffActionSymbol(output.Action)), escapedKeyMaxLen, escapedKeys[key], output.RenderHuman(0, computed.NewRenderHumanOpts(renderer.Colorize))))
		}
	}
	return strings.Join(rendered, "\n")
}

func renderHumanDiffDrift(renderer Renderer, diffs diffs, mode plans.Mode) bool {
	var drs []diff

	// In refresh-only mode, we show all resources marked as drifted,
	// including those which have moved without other changes. In other plan
	// modes, move-only changes will be rendered in the planned changes, so
	// we skip them here.

	if mode == plans.RefreshOnlyMode {
		drs = diffs.drift
	} else {
		for _, dr := range diffs.drift {
			if dr.diff.Action != plans.NoOp {
				drs = append(drs, dr)
			}
		}
	}

	if len(drs) == 0 {
		return false
	}

	// If the overall plan is empty, and it's not a refresh only plan then we
	// won't show any drift changes.
	if diffs.Empty() && mode != plans.RefreshOnlyMode {
		return false
	}

	renderer.Streams.Print(renderer.Colorize.Color("\n[bold][cyan]Note:[reset][bold] Objects have changed outside of Terraform\n"))
	renderer.Streams.Println()
	renderer.Streams.Print(format.WordWrap(
		"Terraform detected the following changes made outside of Terraform since the last \"terraform apply\" which may have affected this plan:\n",
		renderer.Streams.Stdout.Columns()))

	for _, drift := range drs {
		diff, render := renderHumanDiff(renderer, drift, detectedDrift)
		if render {
			renderer.Streams.Println()
			renderer.Streams.Println(diff)
		}
	}

	switch mode {
	case plans.RefreshOnlyMode:
		renderer.Streams.Println(format.WordWrap(
			"\n\nThis is a refresh-only plan, so Terraform will not take any actions to undo these. If you were expecting these changes then you can apply this plan to record the updated values in the Terraform state without changing any remote objects.",
			renderer.Streams.Stdout.Columns(),
		))
	default:
		renderer.Streams.Println(format.WordWrap(
			"\n\nUnless you have made equivalent changes to your configuration, or ignored the relevant attributes using ignore_changes, the following plan may include actions to undo or respond to these changes.",
			renderer.Streams.Stdout.Columns(),
		))
	}

	return true
}

func renderHumanDeferredChanges(renderer Renderer, diffs diffs, mode plans.Mode) bool {
	if len(diffs.deferred) == 0 {
		return false
	}

	renderer.Streams.Print(renderer.Colorize.Color("\n[bold][cyan]Note:[reset][bold] This is a partial plan, parts can only be known in the next plan / apply cycle.\n"))
	renderer.Streams.Println()

	for _, deferred := range diffs.deferred {
		diff, render := renderHumanDeferredDiff(renderer, deferred)
		if render {
			renderer.Streams.Println()
			renderer.Streams.Println(diff)
		}
	}
	return true
}

func renderHumanDiff(renderer Renderer, diff diff, cause string) (string, bool) {

	// Internally, our computed diffs can't tell the difference between a
	// replace action (eg. CreateThenDestroy, DestroyThenCreate) and a simple
	// update action. So, at the top most level we rely on the action provided
	// by the plan itself instead of what we compute. Nested attributes and
	// blocks however don't have the replace type of actions, so we can trust
	// the computed actions of these.

	action := jsonplan.UnmarshalActions(diff.change.Change.Actions)
	if action == plans.NoOp && !diff.Moved() && !diff.Importing() {
		// Skip resource changes that have nothing interesting to say.
		return "", false
	}

	var buf bytes.Buffer
	buf.WriteString(renderer.Colorize.Color(resourceChangeComment(diff.change, action, cause)))

	opts := computed.NewRenderHumanOpts(renderer.Colorize)
	opts.ShowUnchangedChildren = diff.Importing()

	buf.WriteString(fmt.Sprintf("%s %s %s", renderer.Colorize.Color(format.DiffActionSymbol(action)), resourceChangeHeader(diff.change), diff.diff.RenderHuman(0, opts)))
	return buf.String(), true
}

func renderHumanDeferredDiff(renderer Renderer, deferred deferredDiff) (string, bool) {

	// Internally, our computed diffs can't tell the difference between a
	// replace action (eg. CreateThenDestroy, DestroyThenCreate) and a simple
	// update action. So, at the top most level we rely on the action provided
	// by the plan itself instead of what we compute. Nested attributes and
	// blocks however don't have the replace type of actions, so we can trust
	// the computed actions of these.
	action := jsonplan.UnmarshalActions(deferred.diff.change.Change.Actions)
	if action == plans.NoOp && !deferred.diff.Moved() && !deferred.diff.Importing() {
		// Skip resource changes that have nothing interesting to say.
		return "", false
	}

	var buf bytes.Buffer
	var explanation string
	switch deferred.reason {
	// TODO: Add other cases
	case jsonplan.DeferredReasonInstanceCountUnknown:
		explanation = "because the number of resource instances is unknown"
	case jsonplan.DeferredReasonResourceConfigUnknown:
		explanation = "because the resource configuration is unknown"
	case jsonplan.DeferredReasonProviderConfigUnknown:
		explanation = "because the provider configuration is unknown"
	case jsonplan.DeferredReasonDeferredPrereq:
		explanation = "because a prerequisite for this resource is deferred"
	case jsonplan.DeferredReasonAbsentPrereq:
		explanation = "because a prerequisite for this resource has not yet been created"
	default:
		explanation = "for an unknown reason"
	}

	buf.WriteString(renderer.Colorize.Color(fmt.Sprintf("[bold]  # %s[reset] was deferred\n", deferred.diff.change.Address)))
	buf.WriteString(renderer.Colorize.Color(fmt.Sprintf("  #[reset] (%s)\n", explanation)))

	opts := computed.NewRenderHumanOpts(renderer.Colorize)
	opts.ShowUnchangedChildren = deferred.diff.Importing()

	buf.WriteString(fmt.Sprintf("%s %s %s", renderer.Colorize.Color(format.DiffActionSymbol(action)), resourceChangeHeader(deferred.diff.change), deferred.diff.diff.RenderHuman(0, opts)))
	return buf.String(), true
}

func resourceChangeComment(resource jsonplan.ResourceChange, action plans.Action, changeCause string) string {
	var buf bytes.Buffer

	dispAddr := resource.Address
	if len(resource.Deposed) != 0 {
		dispAddr = fmt.Sprintf("%s (deposed object %s)", dispAddr, resource.Deposed)
	}

	var printedMoved bool
	var printedImported bool

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
		case jsonplan.ResourceInstanceReadBecauseCheckNested:
			buf.WriteString("\n  # (config will be reloaded to verify a check block)")
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
	case plans.CreateThenForget:
		buf.WriteString(fmt.Sprintf("[bold] # %s[reset] must be replaced, but the existing object will not be destroyed", dispAddr))
		buf.WriteString("\n # (destroy = false is set in the configuration)")
	case plans.Forget:
		if len(resource.Deposed) > 0 {
			buf.WriteString(fmt.Sprintf("[bold] # %s[reset] will be removed from Terraform state, but [bold][red]will not be destroyed[reset]", dispAddr))
			buf.WriteString("\n[bold] # (left over from a partially-failed replacement of this instance)")
		} else {
			buf.WriteString(fmt.Sprintf("[bold] # %s[reset] will no longer be managed by Terraform, but [bold][red]will not be destroyed[reset]", dispAddr))
		}
		buf.WriteString("\n # (destroy = false is set in the configuration)")
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
			printedMoved = true
			break
		}
		if resource.Change.Importing != nil {
			buf.WriteString(fmt.Sprintf("[bold]  # %s[reset] will be imported", dispAddr))
			if len(resource.Change.GeneratedConfig) > 0 {
				buf.WriteString("\n  #[reset] (config will be generated)")
			}
			printedImported = true
			break
		}
		fallthrough
	default:
		// should never happen, since the above is exhaustive
		buf.WriteString(fmt.Sprintf("%s has an action the plan renderer doesn't support (this is a bug)", dispAddr))
	}
	buf.WriteString("\n")

	if len(resource.PreviousAddress) > 0 && resource.PreviousAddress != resource.Address && !printedMoved {
		buf.WriteString(fmt.Sprintf("  # [reset](moved from %s)\n", resource.PreviousAddress))
	}
	if resource.Change.Importing != nil && !printedImported {
		// We want to make this as forward compatible as possible, and we know
		// the ID may be removed from the Importing metadata in favour of
		// something else.
		// As Importing metadata is loaded from a JSON struct, the effect of it
		// being removed in the future will mean this renderer will receive it
		// as an empty string
		if len(resource.Change.Importing.ID) > 0 {
			buf.WriteString(fmt.Sprintf("  # [reset](imported from \"%s\")\n", resource.Change.Importing.ID))
		} else {
			// This means we're trying to render a plan from a future version
			// and we didn't get given the ID. So we'll do our best.
			buf.WriteString("  # [reset](will be imported first)\n")
		}
	}
	if resource.Change.Importing != nil && (action == plans.CreateThenDelete || action == plans.DeleteThenCreate) {
		buf.WriteString("  # [reset][yellow]Warning: this will destroy the imported resource[reset]\n")
	}

	return buf.String()
}

func resourceChangeHeader(change jsonplan.ResourceChange) string {
	mode := "resource"
	if change.Mode != jsonstate.ManagedResourceMode {
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
