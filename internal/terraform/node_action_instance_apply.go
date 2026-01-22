package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
)

type NodeApplyableActionInstance struct {
	*NodeAbstractActionInstance
}

// GraphNodeReferencer, overriding the NodeAbstractActionInstance method. The
// action config is embedded in the resource during apply, so we only return
// count/for_each references from the action config itself.
func (n *NodeApplyableActionInstance) References() []*addrs.Reference {
	var result []*addrs.Reference
	c := n.Config
	countRefs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, c.Count)
	result = append(result, countRefs...)
	forEachRefs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, c.ForEach)
	result = append(result, forEachRefs...)

	return result
}
