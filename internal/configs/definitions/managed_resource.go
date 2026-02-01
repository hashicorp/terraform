// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package definitions

import (
	"github.com/hashicorp/hcl/v2"
)

// ManagedResource represents a "resource" block in a module or file.
type ManagedResource struct {
	Connection     *Connection
	Provisioners   []*Provisioner
	ActionTriggers []*ActionTrigger

	CreateBeforeDestroy bool
	PreventDestroy      bool
	IgnoreChanges       []hcl.Traversal
	IgnoreAllChanges    bool

	CreateBeforeDestroySet bool
	PreventDestroySet      bool
}
