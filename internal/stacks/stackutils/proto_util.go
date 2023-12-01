package stackutils

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1"
)

func ResourceModeForProto(mode addrs.ResourceMode) terraform1.ResourceMode {
	switch mode {
	case addrs.ManagedResourceMode:
		return terraform1.ResourceMode_MANAGED
	case addrs.DataResourceMode:
		return terraform1.ResourceMode_DATA
	default:
		// Should not get here, because the above should be exhaustive for
		// all addrs.ResourceMode variants.
		return terraform1.ResourceMode_UNKNOWN
	}
}
