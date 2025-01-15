// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraformtest

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/moduletest"
)

type NodeTestRun struct {
	file   *moduletest.File
	run    *moduletest.Run
	config *configs.Config
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
