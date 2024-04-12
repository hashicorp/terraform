// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"slices"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// nodeVariableValidation checks the author-specified validation rules against
// the final value of all expanded instances of a given input variable.
//
// A node of this type should always depend on another node that's responsible
// for deciding the final values for the nominated variable and registering
// them in the current "named values" state. [variableValidationTransformer]
// is the one responsible for inserting nodes of this type and ensuring that
// they each depend on the node that will register the final variable value.
type nodeVariableValidation struct {
	configAddr addrs.ConfigInputVariable
	rules      []*configs.CheckRule

	// defnRange is whatever source range we consider to best represent
	// the definition of the variable, which should ideally cover the
	// source code of the expression that was assigned to the variable.
	// When that's not possible -- for example, if the variable was
	// set from a non-configuration location like an environment variable --
	// it's acceptable to use the declaration location instead.
	defnRange hcl.Range

	// allowGeneralReference is set for nodes that are associated with input
	// variables that belong to modules participating in the
	// "variable_validation_crossref" language experiment, which allows
	// validation rules to refer to other objects declared in the same
	// module as the variable.
	allowGeneralReferences bool
}

var _ GraphNodeModulePath = (*nodeVariableValidation)(nil)
var _ GraphNodeReferenceable = (*nodeVariableValidation)(nil)
var _ GraphNodeReferencer = (*nodeVariableValidation)(nil)
var _ GraphNodeExecutable = (*nodeVariableValidation)(nil)

func (n *nodeVariableValidation) Name() string {
	return fmt.Sprintf("%s (validation)", n.configAddr.String())
}

// ModulePath implements [GraphNodeModulePath].
func (n *nodeVariableValidation) ModulePath() addrs.Module {
	return n.configAddr.Module
}

// ReferenceableAddrs implements [GraphNodeReferenceable], announcing that
// this node contributes to the value for the input variable that it's
// validating, and must therefore run before any nodes that refer to it.
func (n *nodeVariableValidation) ReferenceableAddrs() []addrs.Referenceable {
	return []addrs.Referenceable{n.configAddr.Variable}
}

// References implements [GraphNodeReferencer], announcing anything that
// the check rules refer to, other than the variable that's being validated
// (which gets its dependency connected by [variableValidationTransformer]
// instead).
func (n *nodeVariableValidation) References() []*addrs.Reference {
	var ret []*addrs.Reference
	for _, rule := range n.rules {
		// We ignore all diagnostics here because if an expression contains
		// invalid references then we'll catch them once we visit the
		// node (method Execute).
		condRefs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, rule.Condition)
		msgRefs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, rule.ErrorMessage)
		ret = n.appendRefsFilterSelf(ret, condRefs...)
		ret = n.appendRefsFilterSelf(ret, msgRefs...)
	}
	return ret
}

// appendRefsFilterSelf is a specialized version of builtin [append] that
// ignores any new references to the input variable represented by the
// reciever.
func (n *nodeVariableValidation) appendRefsFilterSelf(to []*addrs.Reference, new ...*addrs.Reference) []*addrs.Reference {
	// We need to filter out any self-references, because those would
	// make the resulting graph invalid and we don't need them because
	// variableValidationTransformer should've arranged for us to
	// already depend on whatever node provides the final value for
	// this variable.
	ret := slices.Grow(to, len(new))
	ourAddr := n.configAddr.Variable
	for _, ref := range new {
		if refAddr, ok := ref.Subject.(addrs.InputVariable); ok {
			if refAddr == ourAddr {
				continue
			}
		}
		ret = append(ret, ref)
	}
	return ret
}

func (n *nodeVariableValidation) Execute(globalCtx EvalContext, op walkOperation) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// We need to perform validation work separately for each instance of
	// the variable across expanded modules, because each one could potentially
	// have a different value assigned to it and other different data in scope.
	expander := globalCtx.InstanceExpander()
	for _, modInst := range expander.ExpandModule(n.configAddr.Module) {
		addr := n.configAddr.Variable.Absolute(modInst)
		moduleCtx := globalCtx.withScope(evalContextModuleInstance{Addr: addr.Module})
		if n.allowGeneralReferences {
			// This is a more general form that's currently available only
			// as an opt-in language experiment. Hopefully eventually this
			// evalVariableValidationsCrossRef function replaces the
			// old evalVariableValidations and we remove the experiment.
			diags = diags.Append(evalVariableValidationsCrossRef(
				addr,
				moduleCtx,
				n.rules,
				n.defnRange,
			))
		} else {
			diags = diags.Append(evalVariableValidations(
				addr,
				moduleCtx,
				n.rules,
				n.defnRange,
			))
		}
	}

	return diags
}
