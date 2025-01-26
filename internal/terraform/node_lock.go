// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// nodeExpandLock represents a named lock in a configuration module,
// which has not yet been expanded.
type nodeExpandLock struct {
	Addr   addrs.Lock
	Module addrs.Module
	Config *configs.Lock
}

var (
	_ GraphNodeReferenceable     = (*nodeExpandLock)(nil)
	_ GraphNodeReferencer        = (*nodeExpandLock)(nil)
	_ GraphNodeDynamicExpandable = (*nodeExpandLock)(nil)
	_ graphNodeTemporaryValue    = (*nodeExpandLock)(nil)
	_ graphNodeExpandsInstances  = (*nodeExpandLock)(nil)
)

func (n *nodeExpandLock) expandsInstances() {}

// graphNodeTemporaryValue
func (n *nodeExpandLock) temporaryValue() bool {
	return true
}

func (n *nodeExpandLock) Name() string {
	path := n.Module.String()
	addr := n.Addr.String() + " (expand)"

	if path != "" {
		return path + "." + addr
	}
	return addr
}

// GraphNodeModulePath
func (n *nodeExpandLock) ModulePath() addrs.Module {
	return n.Module
}

// GraphNodeReferenceable
func (n *nodeExpandLock) ReferenceableAddrs() []addrs.Referenceable {
	return []addrs.Referenceable{n.Addr}
}

// GraphNodeReferencer
func (n *nodeExpandLock) References() []*addrs.Reference {
	refs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, n.Config.Concurrency)
	return refs
}

func (n *nodeExpandLock) DynamicExpand(ctx EvalContext) (*Graph, tfdiags.Diagnostics) {
	var g Graph
	expander := ctx.InstanceExpander()
	for _, instAddr := range expander.ExpandModule(n.Module, false) {
		o := &NodeLock{
			Addr:   n.Addr.Absolute(instAddr),
			Config: n.Config,
		}
		log.Printf("[TRACE] Expanding local: adding %s as %T", o.Addr.String(), o)
		g.Add(o)
	}
	addRootNodeToGraph(&g)
	return &g, nil
}

// NodeLock represents a named lock value in a particular module.
//
// Lock value nodes only have one operation, common to all walk types:
// evaluate the result and place it in state.
type NodeLock struct {
	Addr   addrs.AbsLock
	Config *configs.Lock
}

var (
	_ GraphNodeModuleInstance = (*NodeLock)(nil)
	_ GraphNodeReferenceable  = (*NodeLock)(nil)
	_ GraphNodeReferencer     = (*NodeLock)(nil)
	_ GraphNodeExecutable     = (*NodeLock)(nil)
	_ graphNodeTemporaryValue = (*NodeLock)(nil)
	_ dag.GraphNodeDotter     = (*NodeLock)(nil)
)

// graphNodeTemporaryValue
func (n *NodeLock) temporaryValue() bool {
	return true
}

func (n *NodeLock) Name() string {
	return n.Addr.String()
}

// GraphNodeModuleInstance
func (n *NodeLock) Path() addrs.ModuleInstance {
	return n.Addr.Module
}

// GraphNodeModulePath
func (n *NodeLock) ModulePath() addrs.Module {
	return n.Addr.Module.Module()
}

// GraphNodeReferenceable
func (n *NodeLock) ReferenceableAddrs() []addrs.Referenceable {
	return []addrs.Referenceable{n.Addr.Lock}
}

// GraphNodeReferencer
func (n *NodeLock) References() []*addrs.Reference {
	refs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, n.Config.Concurrency)
	return refs
}

// GraphNodeExecutable
// NodeLock.Execute is an Execute implementation that evaluates the
// expression for a lock value and writes it into the state.
func (n *NodeLock) Execute(ctx EvalContext, op walkOperation) (diags tfdiags.Diagnostics) {
	// TODO: Add to interface
	builtin, ok := ctx.(*BuiltinEvalContext)
	if !ok {
		return nil
	}
	val, diags := evaluateLock(n.Config, n.Addr.Lock, n.Addr.String(), ctx)
	if diags.HasErrors() {
		return diags
	}
	intVal, _ := val.AsBigFloat().Int64() // we already know it's a number
	if intVal < 1 {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid lock value",
			Detail:   "Lock value must be a positive integer.",
			Subject:  n.Config.Concurrency.Range().Ptr(),
		})
	}
	builtin.SetSemaphore(n.Addr, NewSemaphore(int(intVal)))
	return diags
}

// dag.GraphNodeDotter impl.
func (n *NodeLock) DotNode(name string, opts *dag.DotOpts) *dag.DotNode {
	return &dag.DotNode{
		Name: name,
		Attrs: map[string]string{
			"label": n.Name(),
			"shape": "note",
		},
	}
}

func evaluateLock(config *configs.Lock, lockAddr addrs.Lock, addrStr string, ctx EvalContext) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	conc := config.Concurrency

	// We ignore diags here because any problems we might find will be found
	// again in EvaluateExpr below.
	refs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, conc)
	for _, ref := range refs {
		if ref.Subject == lockAddr {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Self-referencing lock value",
				Detail:   fmt.Sprintf("Lock %q cannot use its own result as part of its expression.", addrStr),
				Subject:  ref.SourceRange.ToHCL().Ptr(),
				Context:  conc.Range().Ptr(),
			})
		}
	}
	if diags.HasErrors() {
		return cty.DynamicVal, diags
	}

	val, moreDiags := ctx.EvaluateExpr(conc, cty.Number, nil)
	diags = diags.Append(moreDiags)
	if val == cty.NilVal {
		val = cty.DynamicVal
	}

	return val, diags
}
