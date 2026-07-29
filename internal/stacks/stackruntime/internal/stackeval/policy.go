// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// validatePolicies validates policies once against the complete provider set
// for the Stack run, before any component-level policy evaluation begins.
func (m *Main) validatePolicies(ctx context.Context) tfdiags.Diagnostics {
	client := m.PolicyClient()
	if client == nil {
		return nil
	}

	providerAddrs := make(map[addrs.Provider]struct{}, len(m.ProviderRefTypes()))
	for addr := range m.ProviderRefTypes() {
		providerAddrs[addr] = struct{}{}
	}
	if m.Planning() {
		state := m.PlanPrevState()
		for componentAddr := range state.AllComponentInstances() {
			for _, configAddr := range state.RequiredProviderInstances(componentAddr) {
				providerAddrs[configAddr.Provider] = struct{}{}
			}
		}
	}
	if m.Applying() {
		for _, component := range m.PlanBeingApplied().AllComponents() {
			for _, configAddr := range component.RequiredProviderInstances() {
				providerAddrs[configAddr.Provider] = struct{}{}
			}
		}
	}

	orderedAddrs := make([]addrs.Provider, 0, len(providerAddrs))
	for addr := range providerAddrs {
		orderedAddrs = append(orderedAddrs, addr)
	}
	sort.Slice(orderedAddrs, func(i, j int) bool {
		return orderedAddrs[i].String() < orderedAddrs[j].String()
	})

	var diags tfdiags.Diagnostics
	schemas := make(map[addrs.Provider]providers.ProviderSchema, len(orderedAddrs))
	for _, addr := range orderedAddrs {
		schema, err := m.ProviderType(addr).Schema(ctx)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Provider initialization error",
				fmt.Sprintf("Failed to fetch the provider schema for %s: %s.", addr, err),
			))
			continue
		}
		schemas[addr] = schema
	}
	if diags.HasErrors() {
		return diags
	}

	return terraform.ValidatePolicies(ctx, client, schemas)
}
