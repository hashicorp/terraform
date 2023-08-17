// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mnptu

import (
	"log"

	"github.com/hashicorp/mnptu/internal/tfdiags"
)

// NodeDestroyableDataResourceInstance represents a resource that is "destroyable":
// it is ready to be destroyed.
type NodeDestroyableDataResourceInstance struct {
	*NodeAbstractResourceInstance
}

var (
	_ GraphNodeExecutable = (*NodeDestroyableDataResourceInstance)(nil)
)

// GraphNodeExecutable
func (n *NodeDestroyableDataResourceInstance) Execute(ctx EvalContext, op walkOperation) tfdiags.Diagnostics {
	log.Printf("[TRACE] NodeDestroyableDataResourceInstance: removing state object for %s", n.Addr)
	ctx.State().SetResourceInstanceCurrent(n.Addr, nil, n.ResolvedProvider)
	return nil
}
