// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/refactoring"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type NodeExecuteMoved struct {
	Config  *configs.Config
	Targets []addrs.Targetable
}

var (
	_ GraphNodeExecutable = (*NodeExecuteMoved)(nil)
)

func (n *NodeExecuteMoved) Execute(context EvalContext, _ walkOperation) tfdiags.Diagnostics {
	moves := context.Moves()

	// To say this is an inefficient function is an understatement.
	// But this is just a proof of concept, in practice we could only apply
	// the changes to the refresh state (since that has the most up-to-date
	// data), and then backport the changes into the other states somehow. Or
	// maybe we'd have to do both prior state and refresh state but we could
	// modify the applyMoves function so it accepts all the various states and
	// applies the changes once if necessary.

	state := context.State().Lock()
	refactoring.ApplyMoves(moves, n.Config, state, func(addr addrs.AbsProviderConfig) providers.Interface {
		return context.Provider(addr)
	})
	context.State().Unlock()

	//prevRunState

	prevRunState := context.PrevRunState().Lock()
	refactoring.ApplyMoves(moves, n.Config, prevRunState, func(addr addrs.AbsProviderConfig) providers.Interface {
		return context.Provider(addr)
	})
	context.PrevRunState().Unlock()

	priorState := context.RefreshState().Lock()
	refactoring.ApplyMoves(moves, n.Config, priorState, func(addr addrs.AbsProviderConfig) providers.Interface {
		return context.Provider(addr)
	})
	context.RefreshState().Unlock()

	diags := prePlanVerifyTargetedMoves(moves, n.Targets)
	return diags
}

func (n *NodeExecuteMoved) String() string {
	return "(moved)"
}

func prePlanVerifyTargetedMoves(moveResults *refactoring.Moves, targets []addrs.Targetable) tfdiags.Diagnostics {
	if len(targets) < 1 {
		return nil // the following only matters when targeting
	}

	var diags tfdiags.Diagnostics

	var excluded []addrs.AbsResourceInstance
	for _, result := range moveResults.Changes.Values() {
		fromMatchesTarget := false
		toMatchesTarget := false
		for _, targetAddr := range targets {
			if targetAddr.TargetContains(result.From) {
				fromMatchesTarget = true
			}
			if targetAddr.TargetContains(result.To) {
				toMatchesTarget = true
			}
		}
		if !fromMatchesTarget {
			excluded = append(excluded, result.From)
		}
		if !toMatchesTarget {
			excluded = append(excluded, result.To)
		}
	}
	if len(excluded) > 0 {
		sort.Slice(excluded, func(i, j int) bool {
			return excluded[i].Less(excluded[j])
		})

		var listBuf strings.Builder
		var prevResourceAddr addrs.AbsResource
		for _, instAddr := range excluded {
			// Targeting generally ends up selecting whole resources rather
			// than individual instances, because we don't factor in
			// individual instances until DynamicExpand, so we're going to
			// always show whole resource addresses here, excluding any
			// instance keys. (This also neatly avoids dealing with the
			// different quoting styles required for string instance keys
			// on different shells, which is handy.)
			//
			// To avoid showing duplicates when we have multiple instances
			// of the same resource, we'll remember the most recent
			// resource we rendered in prevResource, which is sufficient
			// because we sorted the list of instance addresses above, and
			// our sort order always groups together instances of the same
			// resource.
			resourceAddr := instAddr.ContainingResource()
			if resourceAddr.Equal(prevResourceAddr) {
				continue
			}
			fmt.Fprintf(&listBuf, "\n  -target=%q", resourceAddr.String())
			prevResourceAddr = resourceAddr
		}
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Moved resource instances excluded by targeting",
			fmt.Sprintf(
				"Resource instances in your current state have moved to new addresses in the latest configuration. Terraform must include those resource instances while planning in order to ensure a correct result, but your -target=... options do not fully cover all of those resource instances.\n\nTo create a valid plan, either remove your -target=... options altogether or add the following additional target options:%s\n\nNote that adding these options may include further additional resource instances in your plan, in order to respect object dependencies.",
				listBuf.String(),
			),
		))
	}

	return diags
}
