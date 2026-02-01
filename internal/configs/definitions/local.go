// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package definitions

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
)

// Local represents a single entry from a "locals" block in a module or file.
// The "locals" block itself is not represented, because it serves only to
// provide context for us to interpret its contents.
type Local struct {
	Name string
	Expr hcl.Expression

	DeclRange hcl.Range
}

// Addr returns the address of the local value declared by the receiver,
// relative to its containing module.
func (l *Local) Addr() addrs.LocalValue {
	return addrs.LocalValue{
		Name: l.Name,
	}
}
