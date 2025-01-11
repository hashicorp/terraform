// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraformtest

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/terraform"
)

// TestFileTransformer is a GraphTransformer that adds all the test runs to the graph.
type TestFileTransformer struct {
	File       *moduletest.File
	config     *configs.Config
	globalVars map[string]backendrun.UnparsedVariableValue
}

func (t *TestFileTransformer) Transform(g *terraform.Graph) error {
	// add the file top-level variables
	for name, expr := range t.File.Config.Variables {
		node := &nodeFileVariable{
			Addr:   addrs.InputVariable{Name: name},
			Expr:   expr,
			config: t.config,
			Module: t.config.Path,
		}
		g.Add(node)
	}
	return nil
}
