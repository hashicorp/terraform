// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
	"github.com/hashicorp/terraform/internal/policy"
)

// PolicyEntitlement returns the host/token/org that Configure resolved, used by
// the policy plugin to verify entitlement. Returns nil if the triple is
// incomplete, in which case the plugin applies its own fallback.
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
