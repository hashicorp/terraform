// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package definitions

import (
	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
)

// Removed describes the contents of a "removed" block in configuration.
type Removed struct {
	// From is the address of the configuration object being removed.
	From *addrs.RemoveTarget

	// Destroy indicates that the resource should be destroyed, not just removed
	// from state. Defaults to true.
	Destroy bool

	// Managed captures a number of metadata fields that are applicable only
	// for managed resources, and not for other resource modes.
	//
	// "removed" blocks support only a subset of the fields in [ManagedResource].
	Managed *ManagedResource

	DeclRange hcl.Range
}
