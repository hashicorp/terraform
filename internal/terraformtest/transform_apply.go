// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraformtest

import (
	"sort"

	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/terraform"
)

// ApplyNoParallelTransformer ensures that all apply operations are run in sequential order.
// If we do not apply this transformer, the apply operations will be run in parallel, which
// may result in multiple runs acting on a particular resource at the same time.
type ApplyNoParallelTransformer struct{}

func (t *ApplyNoParallelTransformer) Transform(g *terraform.Graph) error {
	// find all the apply nodes
	runs := make([]*NodeTestRun, 0)
	for _, v := range g.Vertices() {
		run, ok := v.(*NodeTestRun)
		if !ok {
			continue
		}

		if run.run.Config.Command == configs.ApplyTestCommand {
			runs = append(runs, run)
		}
	}

	// sort them in descending order
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].run.Index > runs[j].run.Index
	})

	// connect them all in serial
	for i := 0; i < len(runs)-1; i++ {
		g.Connect(dag.BasicEdge(runs[i], runs[i+1]))
	}

	return nil
}
