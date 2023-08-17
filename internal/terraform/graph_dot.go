// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mnptu

import "github.com/hashicorp/mnptu/internal/dag"

// GraphDot returns the dot formatting of a visual representation of
// the given mnptu graph.
func GraphDot(g *Graph, opts *dag.DotOpts) (string, error) {
	return string(g.Dot(opts)), nil
}
