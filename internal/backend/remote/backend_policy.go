// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package remote

import (
	"github.com/hashicorp/terraform/internal/policy"
)

// PolicyEntitlement implements policy.EntitlementProvider.
//
// It returns the host/token/organization triple that was resolved by
// Configure from the backend block, environment variables, and the cliconfig
// credentials store (in that precedence order). The returned value is the
// single source of truth the policy plugin uses to verify entitlement.
//
// Returns nil if Configure has not produced a complete triple — in that case
// the caller forwards no entitlement on the Setup RPC and lets the plugin
// apply its own fallback behaviour.
func (b *Remote) PolicyEntitlement() *policy.Entitlement {
	if b == nil || b.hostname == "" || b.organization == "" || b.resolvedToken == "" {
		return nil
	}
	return &policy.Entitlement{
		Host:  b.hostname,
		Token: b.resolvedToken,
		Org:   b.organization,
	}
}
