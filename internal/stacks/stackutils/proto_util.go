// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackutils

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/rpcapi/rawrpc/rawstacks1"
)

func ResourceModeForProto(mode addrs.ResourceMode) rawstacks1.ResourceMode {
	switch mode {
	case addrs.ManagedResourceMode:
		return rawstacks1.ResourceMode_MANAGED
	case addrs.DataResourceMode:
		return rawstacks1.ResourceMode_DATA
	default:
		// Should not get here, because the above should be exhaustive for
		// all addrs.ResourceMode variants.
		return rawstacks1.ResourceMode_UNKNOWN
	}
}
