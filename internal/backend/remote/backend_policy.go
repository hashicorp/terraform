// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package remote

import (
	"github.com/hashicorp/terraform/internal/policy"
)

// PolicyEntitlement returns the host/token/org that Configure resolved, used by
// the policy plugin to verify entitlement. Returns nil if the triple is
// incomplete, in which case the plugin applies its own fallback.
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
