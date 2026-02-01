// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package definitions

import (
	"github.com/hashicorp/hcl/v2"
)

// Connection represents a "connection" block when used within either a
// "resource" or "provisioner" block in a module or file.
type Connection struct {
	Config hcl.Body

	DeclRange hcl.Range
}
