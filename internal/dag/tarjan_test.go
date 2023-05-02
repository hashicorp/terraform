// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package dag

import (
	"sort"
	"strings"
	"testing"
)

func TestGraphStronglyConnected(t *testing.T) {
	var g Graph
	g.Add(1)
	g.Add(2)
	g.Connect(BasicEdge(1, 2))
	g.Connect(BasicEdge(2, 1))

	actual := strings.TrimSpace(testSCCStr(StronglyConnected(&g)))
	expected := strings.TrimSpace(testGraphStronglyConnectedStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestGraphStronglyConnected_two(t *testing.T) {
	var g Graph
	g.Add(1)
	g.Add(2)
	g.Connect(BasicEdge(1, 2))
	g.Connect(BasicEdge(2, 1))
	g.Add(3)

	actual := strings.TrimSpace(testSCCStr(StronglyConnected(&g)))
	expected := strings.TrimSpace(testGraphStronglyConnectedTwoStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestGraphStronglyConnected_three(t *testing.T) {
	var g Graph
	g.Add(1)
	g.Add(2)
	g.Connect(BasicEdge(1, 2))
	g.Connect(BasicEdge(2, 1))
	g.Add(3)
	g.Add(4)
	g.Add(5)
	g.Add(6)
	g.Connect(BasicEdge(4, 5))
	g.Connect(BasicEdge(5, 6))
	g.Connect(BasicEdge(6, 4))

	actual := strings.TrimSpace(testSCCStr(StronglyConnected(&g)))
	expected := strings.TrimSpace(testGraphStronglyConnectedThreeStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func testSCCStr(list [][]Vertex) string {
	var lines []string
	for _, vs := range list {
		result := make([]string, len(vs))
		for i, v := range vs {
			result[i] = VertexName(v)
		}

		sort.Strings(result)
		lines = append(lines, strings.Join(result, ","))
	}

	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

const testGraphStronglyConnectedStr = `1,2`

const testGraphStronglyConnectedTwoStr = `
1,2
3
`

const testGraphStronglyConnectedThreeStr = `
1,2
3
4,5,6
`
