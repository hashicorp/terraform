// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package json

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
)

func NewDeferredResourceInstanceChange(deferredChange *plans.DeferredResourceInstanceChangeSrc) *DeferredResourceInstanceChange {
	dc := &DeferredResourceInstanceChange{
		Reason: deferredReason(deferredChange.DeferredReason),
		Change: NewResourceInstanceChange(deferredChange.ChangeSrc),
	}

	return dc
}

type DeferredResourceInstanceChange struct {
	Reason DeferredReason          `json:"reason"`
	Change *ResourceInstanceChange `json:"change"`
}

func (dc *DeferredResourceInstanceChange) String() string {
	return fmt.Sprintf("%s: deferred change, reason: %s", dc.Change.Resource.Addr, dc.Reason)
}

type DeferredReason string

const (
	DeferredReasonInvalid               DeferredReason = "invalid"
	DeferredReasonInstanceCountUnknown  DeferredReason = "instance_count_unknown"
	DeferredReasonResourceConfigUnknown DeferredReason = "resource_config_unknown"
	DeferredReasonProviderConfigUnknown DeferredReason = "provider_config_unknown"
	DeferredReasonAbsentPrereq          DeferredReason = "absent_prereq"
	DeferredReasonDeferredPrereq        DeferredReason = "deferred_prereq"
	DeferredReasonExcluded              DeferredReason = "excluded"
	DeferredReasonExcludedPrereq        DeferredReason = "excluded_prereq"
)

func deferredReason(reason providers.DeferredReason) DeferredReason {
	switch reason {
	case providers.DeferredReasonInvalid:
		return DeferredReasonInvalid
	case providers.DeferredReasonInstanceCountUnknown:
		return DeferredReasonInstanceCountUnknown
	case providers.DeferredReasonResourceConfigUnknown:
		return DeferredReasonResourceConfigUnknown
	case providers.DeferredReasonProviderConfigUnknown:
		return DeferredReasonProviderConfigUnknown
	case providers.DeferredReasonAbsentPrereq:
		return DeferredReasonAbsentPrereq
	case providers.DeferredReasonDeferredPrereq:
		return DeferredReasonDeferredPrereq
	case providers.DeferredReasonExcluded:
		return DeferredReasonExcluded
	case providers.DeferredReasonExcludedPrereq:
		return DeferredReasonExcludedPrereq
	default:
		// This should never happen, but there's no good way to guarantee
		// exhaustive handling of the enum, so a generic fall back is better
		// than a misleading result or a panic
		return DeferredReasonInvalid
	}
}
