// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package definitions

import (
	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
)

// Output represents an "output" block in a module or file.
type Output struct {
	Name        string
	Description string
	Expr        hcl.Expression
	DependsOn   []hcl.Traversal
	Sensitive   bool
	Ephemeral   bool

	Preconditions []*CheckRule

	DescriptionSet bool
	SensitiveSet   bool
	EphemeralSet   bool

	DeclRange hcl.Range
}

// Addr returns the address of the output.
func (o *Output) Addr() addrs.OutputValue {
	return addrs.OutputValue{Name: o.Name}
}
