// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/terraform"
)

var _ terraform.GraphTransformer = (*TestVariablesTransformer)(nil)

type TestVariablesTransformer struct {
	File *moduletest.File
}

func (v *TestVariablesTransformer) Transform(graph *terraform.Graph) error {
	for name, config := range v.File.Config.VariableDefinitions {
		graph.Add(&NodeVariableDefinition{
			Address: name,
			Config:  config,
			File:    v.File,
		})
	}
	for name, expr := range v.File.Config.Variables {
		graph.Add(&NodeVariableExpression{
			Address: name,
			Expr:    expr,
			File:    v.File,
		})
	}
	return nil
}
