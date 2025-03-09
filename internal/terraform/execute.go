// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import "github.com/hashicorp/terraform/internal/tfdiags"

// GraphNodeExecutable is the interface that graph nodes must implement to
// enable execution.
type GraphNodeExecutable interface {
	Execute(EvalContext, walkOperation) tfdiags.Diagnostics
}

// GraphNodeValidatable is the interface that graph nodes must implement to
// enable validation. Most executable nodes will also be validatable.
type GraphNodeValidatable interface {
	Validate(EvalContext, walkOperation) tfdiags.Diagnostics
}
