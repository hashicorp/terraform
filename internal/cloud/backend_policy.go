// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
	"github.com/hashicorp/terraform/internal/policy"
)

// PolicyEntitlement implements policy.EntitlementProvider.
//
// It returns the host/token/organization triple that was resolved by
// Configure from the cloud{} block, environment variables, and the cliconfig
// credentials store (in that precedence order). The returned value is the
// single source of truth the policy plugin uses to verify entitlement.
//
// Returns nil if Configure has not produced a complete triple — in that case
// the caller forwards no entitlement on the Setup RPC and lets the plugin
// apply its own fallback behaviour.
func (b *Cloud) PolicyEntitlement() *policy.Entitlement {
	if b == nil || b.Hostname == "" || b.Organization == "" || b.Token == "" {
		return nil
	}
	return &policy.Entitlement{
		Host:  b.Hostname,
		Token: b.Token,
		Org:   b.Organization,
	}
}
