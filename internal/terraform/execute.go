// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import "github.com/hashicorp/terraform/internal/tfdiags"

// GraphNodeExecutable is the interface that graph nodes must implement to
// enable execution.
type GraphNodeExecutable interface {
	Execute(EvalContext, walkOperation) tfdiags.Diagnostics
}

type GraphNodeExcludeable interface {
	SetExcluded(bool)
	IsExcluded() bool
}

type Excluded struct {
	excluded bool
}

func (n *Excluded) SetExcluded(excluded bool) {
	n.excluded = excluded
}

func (n *Excluded) IsExcluded() bool {
	return n.excluded
}
