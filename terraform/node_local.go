package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// nodeExpandLocal represents a named local value in a configuration module,
// which has not yet been expanded.
type nodeExpandLocal struct {
	Addr   addrs.LocalValue
	Module addrs.Module
	Config *configs.Local
}

var (
	_ GraphNodeReferenceable     = (*nodeExpandLocal)(nil)
	_ GraphNodeReferencer        = (*nodeExpandLocal)(nil)
	_ GraphNodeDynamicExpandable = (*nodeExpandLocal)(nil)
	_ graphNodeTemporaryValue    = (*nodeExpandLocal)(nil)
	_ graphNodeExpandsInstances  = (*nodeExpandLocal)(nil)
)

func (n *nodeExpandLocal) expandsInstances() {}

// graphNodeTemporaryValue
func (n *nodeExpandLocal) temporaryValue() bool {
	return true
}

func (n *nodeExpandLocal) Name() string {
	path := n.Module.String()
	addr := n.Addr.String() + " (expand)"

	if path != "" {
		return path + "." + addr
	}
	return addr
}

// GraphNodeModulePath
func (n *nodeExpandLocal) ModulePath() addrs.Module {
	return n.Module
}

// GraphNodeReferenceable
func (n *nodeExpandLocal) ReferenceableAddrs() []addrs.Referenceable {
	return []addrs.Referenceable{n.Addr}
}

// GraphNodeReferencer
func (n *nodeExpandLocal) References() []*addrs.Reference {
	refs, _ := lang.ReferencesInExpr(n.Config.Expr)
	return refs
}

func (n *nodeExpandLocal) DynamicExpand(ctx EvalContext) (*Graph, error) {
	var g Graph
	expander := ctx.InstanceExpander()
	for _, module := range expander.ExpandModule(n.Module) {
		o := &NodeLocal{
			Addr:   n.Addr.Absolute(module),
			Config: n.Config,
		}
		log.Printf("[TRACE] Expanding local: adding %s as %T", o.Addr.String(), o)
		g.Add(o)
	}
	return &g, nil
}

// NodeLocal represents a named local value in a particular module.
//
// Local value nodes only have one operation, common to all walk types:
// evaluate the result and place it in state.
type NodeLocal struct {
	Addr   addrs.AbsLocalValue
	Config *configs.Local
}

var (
	_ GraphNodeModuleInstance = (*NodeLocal)(nil)
	_ GraphNodeReferenceable  = (*NodeLocal)(nil)
	_ GraphNodeReferencer     = (*NodeLocal)(nil)
	_ GraphNodeExecutable     = (*NodeLocal)(nil)
	_ graphNodeTemporaryValue = (*NodeLocal)(nil)
	_ dag.GraphNodeDotter     = (*NodeLocal)(nil)
)

// graphNodeTemporaryValue
func (n *NodeLocal) temporaryValue() bool {
	return true
}

func (n *NodeLocal) Name() string {
	return n.Addr.String()
}

// GraphNodeModuleInstance
func (n *NodeLocal) Path() addrs.ModuleInstance {
	return n.Addr.Module
}

// GraphNodeModulePath
func (n *NodeLocal) ModulePath() addrs.Module {
	return n.Addr.Module.Module()
}

// GraphNodeReferenceable
func (n *NodeLocal) ReferenceableAddrs() []addrs.Referenceable {
	return []addrs.Referenceable{n.Addr.LocalValue}
}

// GraphNodeReferencer
func (n *NodeLocal) References() []*addrs.Reference {
	refs, _ := lang.ReferencesInExpr(n.Config.Expr)
	return refs
}

// GraphNodeExecutable
// NodeLocal.Execute is an Execute implementation that evaluates the
// expression for a local value and writes it into a transient part of
// the state.
func (n *NodeLocal) Execute(ctx EvalContext, op walkOperation) (diags tfdiags.Diagnostics) {
	expr := n.Config.Expr
	addr := n.Addr.LocalValue

	// We ignore diags here because any problems we might find will be found
	// again in EvaluateExpr below.
	refs, _ := lang.ReferencesInExpr(expr)
	for _, ref := range refs {
		if ref.Subject == addr {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Self-referencing local value",
				Detail:   fmt.Sprintf("Local value %s cannot use its own result as part of its expression.", addr),
				Subject:  ref.SourceRange.ToHCL().Ptr(),
				Context:  expr.Range().Ptr(),
			})
		}
	}
	if diags.HasErrors() {
		return diags
	}

	val, moreDiags := ctx.EvaluateExpr(expr, cty.DynamicPseudoType, nil)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return diags
	}

	state := ctx.State()
	if state == nil {
		diags = diags.Append(fmt.Errorf("cannot write local value to nil state"))
		return diags
	}

	state.SetLocalValue(addr.Absolute(ctx.Path()), val)

	return diags
}

// dag.GraphNodeDotter impl.
func (n *NodeLocal) DotNode(name string, opts *dag.DotOpts) *dag.DotNode {
	return &dag.DotNode{
		Name: name,
		Attrs: map[string]string{
			"label": n.Name(),
			"shape": "note",
		},
	}
}
