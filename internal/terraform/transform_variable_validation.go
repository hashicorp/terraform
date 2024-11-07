// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
)

// graphNodeValidatableVariable is implemented by nodes that represent
// input variables, and which must therefore have variable validation
// nodes created alongside them to verify that the final value matches
// the author's validation rules.
type graphNodeValidatableVariable interface {
	// variableValidationRules returns the information required to validate
	// the final value produced by the implementing node.
	//
	// configAddr is the address of the static declaration of the variable
	// that is to be validated.
	//
	// rules is the set of validation rules to use to check the final value
	// of the variable.
	//
	// defnRange is the source range to "blame" for any problems. This
	// should ideally cover the source code of the expression that was evaluated
	// to produce the variable's value, but if there is no such expression --
	// for example, if the value came from an environment variable -- then
	// the location of the variable declaration is a plausible substitute.
	variableValidationRules() (configAddr addrs.ConfigInputVariable, rules []*configs.CheckRule, defnRange hcl.Range)
}

// Correct behavior requires both of the input variable node types to
// announce themselves as producing final input variable values that need
// to be validated.
var _ graphNodeValidatableVariable = (*NodeRootVariable)(nil)
var _ graphNodeValidatableVariable = (*nodeExpandModuleVariable)(nil)

// variableValidationTransformer searches the given graph for any nodes
// that implement [graphNodeValidatableVariable]. For each one found, it
// inserts a new [nodeVariableValidation] and makes it depend on the original
// node, to cause the validation action to happen only after the variable's
// final value has been registered.
//
// This transformer should run after any transformer that might insert a
// node that implements [graphNodeValidatableVariable], and before the
// [ReferenceTransformer] because references like "var.foo" must be connected
// with the new [nodeVariableValidation] nodes to prevent downstream nodes
// from relying on unvalidated values.
type variableValidationTransformer struct {
	validateWalk bool
}

var _ GraphTransformer = (*variableValidationTransformer)(nil)

func (t *variableValidationTransformer) Transform(g *Graph) error {
	log.Printf("[TRACE] variableValidationTransformer: adding validation nodes for any existing variable evaluation nodes")
	for _, v := range g.Vertices() {
		v, ok := v.(graphNodeValidatableVariable)
		if !ok {
			continue // irrelevant node
		}

		configAddr, rules, defnRange := v.variableValidationRules()
		newV := &nodeVariableValidation{
			configAddr:   configAddr,
			rules:        rules,
			defnRange:    defnRange,
			validateWalk: t.validateWalk,
		}

		if len(rules) != 0 {
			log.Printf("[TRACE] variableValidationTransformer: %s has %d validation rule(s)", configAddr, len(rules))
			g.Add(newV)
			g.Connect(dag.BasicEdge(newV, v))
		} else {
			log.Printf("[TRACE] variableValidationTransformer: %s has no validation rules", configAddr)
		}
	}
	return nil
}
