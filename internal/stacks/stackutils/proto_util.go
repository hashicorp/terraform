// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackutils

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1/stacks"
)

func ResourceModeForProto(mode addrs.ResourceMode) stacks.ResourceMode {
	switch mode {
	case addrs.ManagedResourceMode:
		return stacks.ResourceMode_MANAGED
	case addrs.DataResourceMode:
		return stacks.ResourceMode_DATA
	default:
		// Should not get here, because the above should be exhaustive for
		// all addrs.ResourceMode variants.
		return stacks.ResourceMode_UNKNOWN
	}
}
