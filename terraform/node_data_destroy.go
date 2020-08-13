package terraform

import (
	"log"

	"github.com/hashicorp/terraform/tfdiags"
)

// NodeDestroyableDataResourceInstance represents a resource that is "destroyable":
// it is ready to be destroyed.
type NodeDestroyableDataResourceInstance struct {
	*NodeAbstractResourceInstance
}

func (n *NodeDestroyableDataResourceInstance) Execute(ctx EvalContext) tfdiags.Diagnostics {
	log.Printf("[TRACE] NodeDestroyableDataResourceInstance: removing state object for %s", n.Addr)
	ctx.State().SetResourceInstanceCurrent(n.Addr, nil, n.ResolvedProvider)
	return tfdiags.Diagnostics{}
}
