// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type NodeTestRun struct {
	file *moduletest.File
	run  *moduletest.Run
}

func (n *NodeTestRun) Execute(ctx EvalContext, op walkOperation) tfdiags.Diagnostics {
	fmt.Println("NodeTestRun.Execute", n.run.Name)
	return nil
}

func (n *NodeTestRun) Run() *moduletest.Run {
	return n.run
}

func (n *NodeTestRun) File() *moduletest.File {
	return n.file
}
