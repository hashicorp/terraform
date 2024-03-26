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

	var reason int32
	switch d.Reason {
	case proto.Deferred_UNKNOWN:
		reason = providers.DEFERRED_REASON_UNKNOWN
	case proto.Deferred_RESOURCE_CONFIG_UNKNOWN:
		reason = providers.DEFERRED_REASON_RESOURCE_CONFIG_UNKNOWN
	case proto.Deferred_PROVIDER_CONFIG_UNKNOWN:
		reason = providers.DEFERRED_REASON_PROVIDER_CONFIG_UNKNOWN
	case proto.Deferred_ABSENT_PREREQ:
		reason = providers.DEFERRED_REASON_ABSENT_PREREQ
	default:
		reason = providers.DEFERRED_REASON_UNKNOWN
	}

	return &providers.Deferred{
		Reason: reason,
	}
}
