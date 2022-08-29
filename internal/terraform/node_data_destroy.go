package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/tfdiags"
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
	//ctx.State().SetResourceInstanceCurrent(n.Addr, nil, n.ResolvedProvider)
	// FIXME: Not implemented in dynamic provider assignment prototype
	var diags tfdiags.Diagnostics
	diags = diags.Append(tfdiags.Sourceless(
		tfdiags.Error,
		"Unimplemented functionality",
		"The dynamic provider assignment prototype doesn't support destroying data source instances.",
	))
	return diags
}
