// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package convert

import (
	"github.com/hashicorp/terraform/internal/providers"
	proto "github.com/hashicorp/terraform/internal/tfplugin5"
)

// ProtoToDeferred translates a proto.Deferred to a providers.Deferred.
func ProtoToDeferred(d *proto.Deferred) *providers.Deferred {
	if d == nil {
		return nil
	}

	var reason providers.DeferredReason
	switch d.Reason {
	case proto.Deferred_UNKNOWN:
		reason = providers.DeferredReasonInvalid
	case proto.Deferred_RESOURCE_CONFIG_UNKNOWN:
		reason = providers.DeferredReasonResourceConfigUnknown
	case proto.Deferred_PROVIDER_CONFIG_UNKNOWN:
		reason = providers.DeferredReasonProviderConfigUnknown
	case proto.Deferred_ABSENT_PREREQ:
		reason = providers.DeferredReasonAbsentPrereq
	default:
		reason = providers.DeferredReasonInvalid
	}

	return &providers.Deferred{
		Reason: reason,
	}
}
