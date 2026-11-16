// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// validatePolicyRelationships validates the policy client's relationship
// trees once for the deployment (contract §3.8), against the schemas of
// every provider the deployment can use (contract §3.3). It must be called
// before the walks; with errors, the operation must not walk.
//
// Each component instance's walk then relates and evaluates its own
// instances with the validated trees, in its own policy phase, like a
// Terraform run.
func (m *Main) validatePolicyRelationships(ctx context.Context, phase EvalPhase) tfdiags.Diagnostics {
	client := m.PolicyClient()
	if client == nil {
		return nil
	}
	rels := client.Relationships()
	if len(rels.Trees) == 0 {
		return nil
	}
	schemas := make(map[addrs.Provider]providers.ProviderSchema)
	for addr := range m.providerFactories {
		schema, err := m.ProviderType(addr).Schema(ctx)
		if err != nil {
			log.Printf("[WARN] policy: no schema for %s to validate relationships: %s", addr, err)
			continue
		}
		schemas[addr] = schema
	}
	locks := m.DependencyLocks(phase)
	diags := terraform.ValidatePolicyRelationships(rels, schemas, func(provider addrs.Provider) string {
		if locks == nil {
			return ""
		}
		if lock := locks.Provider(provider); lock != nil {
			return lock.Version().String()
		}
		return ""
	})
	m.mu.Lock()
	defer m.mu.Unlock()
	if diags.HasErrors() {
		m.policyInvalid = true
		return diags
	}
	m.policyRelationships = &rels
	return diags
}

// PolicyInvalid returns true if the policy client's relationship
// definitions are invalid for the deployment.
func (m *Main) PolicyInvalid() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.policyInvalid
}

// PolicyRelationships returns the deployment's validated relationship
// trees, or nil if it has none.
func (m *Main) PolicyRelationships() *policy.Relationships {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.policyRelationships
}
