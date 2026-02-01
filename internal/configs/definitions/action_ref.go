// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package definitions

import (
	"github.com/hashicorp/hcl/v2"
)

// ActionRef represents a reference to a configured Action
type ActionRef struct {
	Expr  hcl.Expression
	Range hcl.Range
}
