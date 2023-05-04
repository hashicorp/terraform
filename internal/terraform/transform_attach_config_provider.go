// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
)

// GraphNodeAttachProvider is an interface that must be implemented by nodes
// that want provider configurations attached.
type GraphNodeAttachProvider interface {
	// ProviderName with no module prefix. Example: "aws".
	ProviderAddr() addrs.AbsProviderConfig

	// Sets the configuration
	AttachProvider(*configs.Provider)
}
