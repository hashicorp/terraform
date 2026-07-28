// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package local

import (
	"github.com/hashicorp/terraform/internal/policy"
)

// PolicyEntitlement delegates to the wrapped backend, so plan and apply can read
// the host/token/org from the underlying cloud or remote backend. Returns nil if
// the wrapped backend can't supply one.
func (b *Local) PolicyEntitlement() *policy.Entitlement {
	if b == nil || b.Backend == nil {
		return nil
	}
	provider, ok := b.Backend.(policy.EntitlementProvider)
	if !ok {
		return nil
	}
	return provider.PolicyEntitlement()
}
