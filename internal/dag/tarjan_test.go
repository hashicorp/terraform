// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package dag

import (
	"sort"
	"strings"
	"testing"
)

func TestGraphStronglyConnected(t *testing.T) {
	var g Graph
	g.Add(testV(1))
	g.Add(testV(2))
	g.Connect(testV(1), testV(2))
	g.Connect(testV(2), testV(1))

	actual := strings.TrimSpace(testSCCStr(StronglyConnected(&g)))
	expected := strings.TrimSpace(testGraphStronglyConnectedStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestGraphStronglyConnected_two(t *testing.T) {
	var g Graph
	g.Add(testV(1))
	g.Add(testV(2))
	g.Connect(testV(1), testV(2))
	g.Connect(testV(2), testV(1))
	g.Add(testV(3))

	actual := strings.TrimSpace(testSCCStr(StronglyConnected(&g)))
	expected := strings.TrimSpace(testGraphStronglyConnectedTwoStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestGraphStronglyConnected_three(t *testing.T) {
	var g Graph
	g.Add(testV(1))
	g.Add(testV(2))
	g.Connect(testV(1), testV(2))
	g.Connect(testV(2), testV(1))
	g.Add(testV(3))
	g.Add(testV(4))
	g.Add(testV(5))
	g.Add(testV(6))
	g.Connect(testV(4), testV(5))
	g.Connect(testV(5), testV(6))
	g.Connect(testV(6), testV(4))

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
