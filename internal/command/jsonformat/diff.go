// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package jsonformat

import (
	"fmt"
	"slices"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
	"github.com/hashicorp/terraform/internal/command/jsonformat/differ"
	"github.com/hashicorp/terraform/internal/command/jsonformat/structured"
	"github.com/hashicorp/terraform/internal/command/jsonformat/structured/attribute_path"
	"github.com/hashicorp/terraform/internal/command/jsonplan"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/plans"
)

func precomputeDiffs(plan Plan, mode plans.Mode) diffs {
	diffs := diffs{
		outputs: make(map[string]computed.Diff),
	}

	for _, drift := range plan.ResourceDrift {

		var relevantAttrs attribute_path.Matcher
		if mode == plans.RefreshOnlyMode {
			// For a refresh only plan, we show all the drift.
			relevantAttrs = attribute_path.AlwaysMatcher()
		} else {
			matcher := attribute_path.Empty(true)

			// Otherwise we only want to show the drift changes that are
			// relevant.
			for _, attr := range plan.RelevantAttributes {
				if len(attr.Resource) == 0 || attr.Resource == drift.Address {
					matcher = attribute_path.AppendSingle(matcher, attr.Attr)
				}
			}

			if len(matcher.Paths) > 0 {
				relevantAttrs = matcher
			}
		}

		if relevantAttrs == nil {
			// If we couldn't build a relevant attribute matcher, then we are
			// not going to show anything for this drift.
			continue
		}

		schema := plan.getSchema(drift)
		change := structured.FromJsonChange(drift.Change, relevantAttrs)
		diffs.drift = append(diffs.drift, diff{
			change: drift,
			diff:   differ.ComputeDiffForBlock(change, schema.Block),
		})
	}

	for _, change := range plan.ResourceChanges {
		schema := plan.getSchema(change)
		structuredChange := structured.FromJsonChange(change.Change, attribute_path.AlwaysMatcher())

		before := []jsonplan.ActionInvocation{}
		after := []jsonplan.ActionInvocation{}

		for _, action := range plan.ActionInvocations {
			if action.LifecycleActionTrigger == nil || action.LifecycleActionTrigger.TriggeringResourceAddress != change.Address {
				continue
			}

			switch action.LifecycleActionTrigger.ActionTriggerEvent {
			case configs.BeforeCreate.String(), configs.BeforeUpdate.String(), configs.BeforeDestroy.String():
				before = append(before, action)
			case configs.AfterCreate.String(), configs.AfterUpdate.String(), configs.AfterDestroy.String():
				after = append(after, action)
			default:
				// The switch should be exhaustive.
				panic(fmt.Sprintf("Unexpected triggering event when rendering action %s", action.LifecycleActionTrigger.ActionTriggerEvent))
			}
		}

		slices.SortFunc(before, jsonplan.ActionInvocationCompare)
		slices.SortFunc(after, jsonplan.ActionInvocationCompare)

		var beforeActionsTriggered []actionInvocation
		var afterActionsTriggered []actionInvocation
		for _, action := range before {
			schema := plan.getActionSchema(action)
			beforeActionsTriggered = append(beforeActionsTriggered, actionInvocation{
				invocation: action,
				schema:     schema,
			})
		}
		for _, action := range after {
			schema := plan.getActionSchema(action)
			afterActionsTriggered = append(afterActionsTriggered, actionInvocation{
				invocation: action,
				schema:     schema,
			})
		}

		diffs.changes = append(diffs.changes, diff{
			change:                 change,
			diff:                   differ.ComputeDiffForBlock(structuredChange, schema.Block),
			beforeActionsTriggered: beforeActionsTriggered,
			afterActionsTriggered:  afterActionsTriggered,
		})
	}

	for _, action := range plan.ActionInvocations {
		if action.InvokeActionTrigger == nil {
			// lifecycle actions are handled within the resource
			continue
		}
		diffs.actions = append(diffs.actions, actionInvocation{
			invocation: action,
			schema:     plan.getActionSchema(action),
		})
	}

	for _, change := range plan.DeferredChanges {
		schema := plan.getSchema(change.ResourceChange)
		structuredChange := structured.FromJsonChange(change.ResourceChange.Change, attribute_path.AlwaysMatcher())
		diffs.deferred = append(diffs.deferred, deferredDiff{
			reason: change.Reason,
			diff: diff{
				change: change.ResourceChange,
				diff:   differ.ComputeDiffForBlock(structuredChange, schema.Block),
			},
		})
	}

	for key, output := range plan.OutputChanges {
		change := structured.FromJsonChange(output, attribute_path.AlwaysMatcher())
		diffs.outputs[key] = differ.ComputeDiffForOutput(change)
	}

	return diffs
}

type diffs struct {
	drift    []diff
	changes  []diff
	deferred []deferredDiff
	actions  []actionInvocation
	outputs  map[string]computed.Diff
}

func (d diffs) Empty() bool {
	for _, change := range d.changes {
		if change.diff.Action != plans.NoOp || change.Moved() {
			return false
		}
	}

	for _, output := range d.outputs {
		if output.Action != plans.NoOp {
			return false
		}
	}

	return true
}

type diff struct {
	change                 jsonplan.ResourceChange
	diff                   computed.Diff
	beforeActionsTriggered []actionInvocation
	afterActionsTriggered  []actionInvocation
}

type actionInvocation struct {
	invocation jsonplan.ActionInvocation
	schema     *jsonprovider.ActionSchema
}

func (d diff) Moved() bool {
	return len(d.change.PreviousAddress) > 0 && d.change.PreviousAddress != d.change.Address
}

func (d diff) Importing() bool {
	return d.change.Change.Importing != nil
}

type deferredDiff struct {
	diff   diff
	reason string
}
