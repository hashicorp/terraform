// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type NodeTestRun struct {
	file *moduletest.File
	run  *moduletest.Run

	// requiredProviders is a map of provider names that the test run depends on.
	requiredProviders map[string]bool
}

func (n *NodeTestRun) Run() *moduletest.Run {
	return n.run
}

func (n *NodeTestRun) File() *moduletest.File {
	return n.file
}

func (n *NodeTestRun) Name() string {
	return fmt.Sprintf("%s.%s", n.file.Name, n.run.Name)
}

// Execute adds the providers required by the test run to the context.
func (n *NodeTestRun) Execute(ctx *EvalContext) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	ctx.SetProviders(n.run, n.requiredProviders)
	return diags
}
