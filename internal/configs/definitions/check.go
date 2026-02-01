// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package definitions

import (
	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
)

// Check represents a configuration defined check block.
//
// A check block contains 0-1 data blocks, and 0-n assert blocks. The check
// block will load the data block, and execute the assert blocks as check rules
// during the plan and apply Terraform operations.
type Check struct {
	Name string

	DataResource *Resource
	Asserts      []*CheckRule

	DeclRange hcl.Range
}

// Addr returns the address of the check block.
func (c Check) Addr() addrs.Check {
	return addrs.Check{
		Name: c.Name,
	}
}

// Accessible implements the Container interface.
func (c Check) Accessible(addr addrs.Referenceable) bool {
	if check, ok := addr.(addrs.Check); ok {
		return check.Equal(c.Addr())
	}
	return false
}
