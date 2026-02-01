// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package definitions

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
)

// Moved represents a "moved" block in a module or file.
type Moved struct {
	From *addrs.MoveEndpoint
	To   *addrs.MoveEndpoint

	DeclRange hcl.Range
}
