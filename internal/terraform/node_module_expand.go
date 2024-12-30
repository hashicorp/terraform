// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type ConcreteModuleNodeFunc func(n *nodeExpandModule) dag.Vertex

// nodeExpandModule represents a module call in the configuration that
// might expand into multiple module instances depending on how it is
// configured.
type nodeExpandModule struct {
	Addr       addrs.Module
	Config     *configs.Module
	ModuleCall *configs.ModuleCall
}

var (
	_ GraphNodeExecutable       = (*nodeExpandModule)(nil)
	_ GraphNodeReferenceable    = (*nodeExpandModule)(nil)
	_ GraphNodeReferencer       = (*nodeExpandModule)(nil)
	_ GraphNodeReferenceOutside = (*nodeExpandModule)(nil)
	_ graphNodeExpandsInstances = (*nodeExpandModule)(nil)
)

func (n *nodeExpandModule) expandsInstances() {}

func (n *nodeExpandModule) Name() string {
	return n.Addr.String() + " (expand)"
}

// GraphNodeModulePath implementation
func (n *nodeExpandModule) ModulePath() addrs.Module {
	return n.Addr
}

// GraphNodeReferencer implementation
func (n *nodeExpandModule) References() []*addrs.Reference {
	var refs []*addrs.Reference

	if n.ModuleCall == nil {
		return nil
	}

	refs = append(refs, n.DependsOn()...)

	// Expansion only uses the count and for_each expressions, so this
	// particular graph node only refers to those.
	// Individual variable values in the module call definition might also
	// refer to other objects, but that's handled by
	// NodeApplyableModuleVariable.
	//
	// Because our Path method returns the module instance that contains
	// our call, these references will be correctly interpreted as being
	// in the calling module's namespace, not the namespaces of any of the
	// child module instances we might expand to during our evaluation.

	if n.ModuleCall.Count != nil {
		countRefs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, n.ModuleCall.Count)
		refs = append(refs, countRefs...)
	}
	if n.ModuleCall.ForEach != nil {
		forEachRefs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, n.ModuleCall.ForEach)
		refs = append(refs, forEachRefs...)
	}
	return refs
}

func (n *nodeExpandModule) ReferenceableAddrs() []addrs.Referenceable {
	// Anything referencing this module must do so after the ExpandModule call
	// has been made to the expander, so we return the module call address as
	// the only referenceable address.
	_, call := n.Addr.Call()
	return []addrs.Referenceable{call}
}

func (n *nodeExpandModule) DependsOn() []*addrs.Reference {
	if n.ModuleCall == nil {
		return nil
	}

	var refs []*addrs.Reference
	for _, traversal := range n.ModuleCall.DependsOn {
		ref, diags := addrs.ParseRef(traversal)
		if diags.HasErrors() {
			// We ignore this here, because this isn't a suitable place to return
			// errors. This situation should be caught and rejected during
			// validation.
			log.Printf("[ERROR] Can't parse %#v from depends_on as reference: %s", traversal, diags.Err())
			continue
		}

		refs = append(refs, ref)
	}

	return refs
}

// GraphNodeReferenceOutside
func (n *nodeExpandModule) ReferenceOutside() (selfPath, referencePath addrs.Module) {
	return n.Addr.Parent(), n.Addr.Parent()
}

// GraphNodeExecutable
func (n *nodeExpandModule) Execute(globalCtx EvalContext, op walkOperation) (diags tfdiags.Diagnostics) {
	expander := globalCtx.InstanceExpander()
	_, call := n.Addr.Call()

	// Allowing unknown values in count and for_each is a top-level plan option.
	//
	// If this is false then the codepaths that handle unknown values below
	// become unreachable, because the evaluate functions will reject unknown
	// values as an error.
	allowUnknown := globalCtx.Deferrals().DeferralAllowed()

	// nodeExpandModule itself does not have visibility into how its ancestors
	// were expanded, so we use the expander here to provide all possible paths
	// to our module, and register module instances with each of them.
	for _, module := range expander.ExpandModule(n.Addr.Parent(), false) {
		moduleCtx := evalContextForModuleInstance(globalCtx, module)

		switch {
		case n.ModuleCall.Count != nil:
			count, ctDiags := evaluateCountExpression(n.ModuleCall.Count, moduleCtx, allowUnknown)
			diags = diags.Append(ctDiags)
			if diags.HasErrors() {
				return diags
			}
			if count >= 0 {
				expander.SetModuleCount(module, call, count)
			} else {
				// -1 represents "unknown"
				expander.SetModuleCountUnknown(module, call)
			}

		case n.ModuleCall.ForEach != nil:
			forEach, known, feDiags := evaluateForEachExpression(n.ModuleCall.ForEach, moduleCtx, allowUnknown)
			diags = diags.Append(feDiags)
			if diags.HasErrors() {
				return diags
			}
			if known {
				expander.SetModuleForEach(module, call, forEach)
			} else {
				expander.SetModuleForEachUnknown(module, call)
			}

		default:
			expander.SetModuleSingle(module, call)
		}
	}

	return diags

}

// nodeCloseModule represents an expanded module during apply, and is visited
// after all other module instance nodes. This node will depend on all module
// instance resource and outputs, and anything depending on the module should
// wait on this node.
// Besides providing a root node for dependency ordering, nodeCloseModule also
// cleans up state after all the module nodes have been evaluated, removing
// empty resources and modules from the state.
// The root module instance also closes any remaining provisioner plugins which
// do not have a lifecycle controlled by individual graph nodes.
type nodeCloseModule struct {
	Addr addrs.Module
}

var (
	_ GraphNodeReferenceable    = (*nodeCloseModule)(nil)
	_ GraphNodeReferenceOutside = (*nodeCloseModule)(nil)
	_ GraphNodeExecutable       = (*nodeCloseModule)(nil)
)

func (n *nodeCloseModule) ModulePath() addrs.Module {
	return n.Addr
}

func (n *nodeCloseModule) ReferenceOutside() (selfPath, referencePath addrs.Module) {
	return n.Addr.Parent(), n.Addr
}

func (n *nodeCloseModule) ReferenceableAddrs() []addrs.Referenceable {
	_, call := n.Addr.Call()
	return []addrs.Referenceable{
		call,
	}
}

func (n *nodeCloseModule) Name() string {
	if len(n.Addr) == 0 {
		return "root"
	}
	return n.Addr.String() + " (close)"
}

func (n *nodeCloseModule) Execute(ctx EvalContext, op walkOperation) (diags tfdiags.Diagnostics) {
	if !n.Addr.IsRoot() {
		return
	}

	// If this is the root module, we are cleaning up the walk, so close
	// any running plugins
	diags = diags.Append(ctx.ClosePlugins())

	// We also close up the ephemeral resource manager
	diags = diags.Append(ctx.EphemeralResources().Close(ctx.StopCtx()))

	switch op {
	case walkApply, walkDestroy:
		state := ctx.State().Lock()
		defer ctx.State().Unlock()

		for modKey, mod := range state.Modules {
			// clean out any empty resources
			for resKey, res := range mod.Resources {
				if len(res.Instances) == 0 {
					delete(mod.Resources, resKey)
				}
			}

			// we don't ever remove a module that's been overridden - it will
			// have outputs that have been set by the user and wouldn't be
			// removed during normal operations as the module would have created
			// resources. Overrides are only set during tests, and stop the
			// module creating resources but we still care about the outputs.
			overridden := false
			if overrides := ctx.Overrides(); !overrides.Empty() {
				_, overridden = overrides.GetModuleOverride(mod.Addr)
			}

			// empty child modules are always removed
			if len(mod.Resources) == 0 && !mod.Addr.IsRoot() && !overridden {
				delete(state.Modules, modKey)
			}
		}
		return nil
	default:
		return nil
	}
}

// nodeValidateModule wraps a nodeExpand module for validation, ensuring that
// no expansion is attempted during evaluation, when count and for_each
// expressions may not be known.
type nodeValidateModule struct {
	nodeExpandModule
}

var _ GraphNodeExecutable = (*nodeValidateModule)(nil)

// GraphNodeEvalable
func (n *nodeValidateModule) Execute(globalCtx EvalContext, op walkOperation) (diags tfdiags.Diagnostics) {
	_, call := n.Addr.Call()
	expander := globalCtx.InstanceExpander()

	// Modules all evaluate to single instances during validation, only to
	// create a proper context within which to evaluate. All parent modules
	// will be a single instance, but still get our address in the expected
	// manner anyway to ensure they've been registered correctly.
	for _, module := range expander.ExpandModule(n.Addr.Parent(), false) {
		moduleCtx := evalContextForModuleInstance(globalCtx, module)

		// Validate our for_each and count expressions at a basic level
		// We skip validation on known, because there will be unknown values before
		// a full expansion, presuming these errors will be caught in later steps
		switch {
		case n.ModuleCall.Count != nil:
			_, countDiags := evaluateCountExpressionValue(n.ModuleCall.Count, moduleCtx)
			diags = diags.Append(countDiags)

		case n.ModuleCall.ForEach != nil:
			forEachDiags := newForEachEvaluator(n.ModuleCall.ForEach, moduleCtx, false).ValidateResourceValue()
			diags = diags.Append(forEachDiags)
		}

		diags = diags.Append(validateDependsOn(moduleCtx, n.ModuleCall.DependsOn))

		// now set our own mode to single
		expander.SetModuleSingle(module, call)
	}

	return diags
}
