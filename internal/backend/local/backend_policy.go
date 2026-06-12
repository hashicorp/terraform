// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package local

import (
	"github.com/hashicorp/terraform/internal/policy"
)

// PolicyEntitlement implements policy.EntitlementProvider by delegating to the
// wrapped backend, if it can supply an entitlement. This lets plan and apply
// pull the host/token/org triple from the underlying cloud or remote backend
// even when state operations are handled by a wrapping Local.
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
