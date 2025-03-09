// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/tfdiags"
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
	refs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, n.Config.Expr)
	return refs
}

func (n *nodeExpandLocal) DynamicExpand(ctx EvalContext) (*Graph, tfdiags.Diagnostics) {
	var g Graph
	expander := ctx.InstanceExpander()
	forEachModuleInstance(expander, n.Module, false, func(module addrs.ModuleInstance) {
		o := &NodeLocal{
			Addr:   n.Addr.Absolute(module),
			Config: n.Config,
		}
		log.Printf("[TRACE] Expanding local: adding %s as %T", o.Addr.String(), o)
		g.Add(o)
	}, func(pem addrs.PartialExpandedModule) {
		o := &nodeLocalInPartialModule{
			Addr:   addrs.ObjectInPartialExpandedModule(pem, n.Addr),
			Config: n.Config,
		}
		log.Printf("[TRACE] Expanding local: adding placeholder for all %s as %T", o.Addr.String(), o)
		g.Add(o)
	})
	addRootNodeToGraph(&g)
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
	refs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, n.Config.Expr)
	return refs
}

// GraphNodeExecutable
// NodeLocal.Execute is an Execute implementation that evaluates the
// expression for a local value and writes it into a transient part of
// the state.
func (n *NodeLocal) Execute(ctx EvalContext, op walkOperation) (diags tfdiags.Diagnostics) {
	namedVals := ctx.NamedValues()
	val, diags := evaluateLocalValue(n.Config, n.Addr.LocalValue, n.Addr.String(), ctx)
	namedVals.SetLocalValue(n.Addr, val)
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

// nodeLocalInPartialModule represents an infinite set of possible local value
// instances beneath a partially-expanded module instance prefix.
//
// Its job is to find a suitable placeholder value that approximates the
// values of all of those possible instances. Ideally that's a concrete
// known value if all instances would have the same value, an unknown value
// of a specific type if the definition produces a known type, or a
// totally-unknown value of unknown type in the worst case.
type nodeLocalInPartialModule struct {
	Addr   addrs.InPartialExpandedModule[addrs.LocalValue]
	Config *configs.Local
}

// Path implements [GraphNodePartialExpandedModule], meaning that the
// Execute method receives an [EvalContext] that's set up for partial-expanded
// evaluation instead of full evaluation.
func (n *nodeLocalInPartialModule) Path() addrs.PartialExpandedModule {
	return n.Addr.Module
}

func (n *nodeLocalInPartialModule) Execute(ctx EvalContext, op walkOperation) tfdiags.Diagnostics {
	// Our job here is to make sure that the local value definition is
	// valid for all instances of this local value across all of the possible
	// module instances under our partially-expanded prefix, and to record
	// a placeholder value that captures as precisely as possible what all
	// of those results have in common. In the worst case where they have
	// absolutely nothing in common cty.DynamicVal is the ultimate fallback,
	// but we should try to do better when possible to give operators earlier
	// feedback about any problems they would definitely encounter on a
	// subsequent plan where the local values get evaluated concretely.

	namedVals := ctx.NamedValues()
	val, diags := evaluateLocalValue(n.Config, n.Addr.Local, n.Addr.String(), ctx)
	namedVals.SetLocalValuePlaceholder(n.Addr, val)
	return diags
}

// evaluateLocalValue is the common evaluation logic shared between
// [NodeLocal] and [nodeLocalInPartialModule].
//
// The overall validation and evaluation process is the same for each, with
// the differences encapsulated inside the given [EvalContext], which is
// configured in a different way when doing partial-expanded evaluation.
//
// the addrStr argument should be the canonical string representation of the
// anbsolute address of the object being evaluated, which should either be an
// [addrs.AbsLocalValue] or an [addrs.InPartialEvaluatedModule[addrs.LocalValue]]
// depending on which of the two callers are calling this function.
//
// localAddr should match the local portion of the address that was stringified
// for addrStr, describing the local value relative to the module it's declared
// inside.
func evaluateLocalValue(config *configs.Local, localAddr addrs.LocalValue, addrStr string, ctx EvalContext) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	expr := config.Expr

	// We ignore diags here because any problems we might find will be found
	// again in EvaluateExpr below.
	refs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, expr)
	for _, ref := range refs {
		if ref.Subject == localAddr {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Self-referencing local value",
				Detail:   fmt.Sprintf("Local value %s cannot use its own result as part of its expression.", addrStr),
				Subject:  ref.SourceRange.ToHCL().Ptr(),
				Context:  expr.Range().Ptr(),
			})
		}
	}
	if diags.HasErrors() {
		return cty.DynamicVal, diags
	}

	val, moreDiags := ctx.EvaluateExpr(expr, cty.DynamicPseudoType, nil)
	diags = diags.Append(moreDiags)
	if val == cty.NilVal {
		val = cty.DynamicVal
	}
	return val, diags
}
