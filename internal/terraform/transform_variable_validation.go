// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
)

// graphNodeValidatableVariable is implemented by nodes that represent
// input variables, and which must therefore have variable validation
// nodes created alongside them to verify that the final value matches
// the author's validation rules.
type graphNodeValidatableVariable interface {
	variableValidationRules() (addrs.ConfigInputVariable, []*configs.CheckRule)
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
}

var _ GraphTransformer = (*variableValidationTransformer)(nil)

func (t *variableValidationTransformer) Transform(g *Graph) error {
	for _, v := range g.Vertices() {
		v, ok := v.(graphNodeValidatableVariable)
		if !ok {
			continue // irrelevant node
		}

		configAddr, rules := v.variableValidationRules()

		newV := &nodeVariableValidation{
			configAddr: configAddr,
			rules:      rules,
		}

		g.Add(newV)
		g.Connect(dag.BasicEdge(newV, v))
	}
	return nil
}
