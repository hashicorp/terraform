// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package dag

import (
	"strings"
	"testing"
)

func TestGraphDot_empty(t *testing.T) {
	var g Graph
	g.Add(testV(1))
	g.Add(testV(2))
	g.Add(testV(3))

	actual := strings.TrimSpace(string(g.Dot(nil)))
	expected := strings.TrimSpace(testGraphDotEmptyStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestGraphDot_basic(t *testing.T) {
	var g Graph
	g.Add(testV(1))
	g.Add(testV(2))
	g.Add(testV(3))
	g.Connect(testV(1), testV(3))

	actual := strings.TrimSpace(string(g.Dot(nil)))
	expected := strings.TrimSpace(testGraphDotBasicStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestGraphDot_quoted(t *testing.T) {
	var g Graph
	quoted := testNamedString(`name["with-quotes"]`)
	other := testNamedString(`other`)
	g.Add(quoted)
	g.Add(other)
	g.Connect(quoted, other)

	actual := strings.TrimSpace(string(g.Dot(nil)))
	expected := strings.TrimSpace(testGraphDotQuotedStr)
	if actual != expected {
		t.Fatalf("\ngot:   %q\nwanted %q\n", actual, expected)
	}
}

func TestGraphDot_attrs(t *testing.T) {
	var g Graph
	g.Add(&testGraphNodeDotter{
		Result: &DotNode{
			Name:  "foo",
			Attrs: map[string]string{"foo": "bar"},
		},
	})

	actual := strings.TrimSpace(string(g.Dot(nil)))
	expected := strings.TrimSpace(testGraphDotAttrsStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

type testGraphNodeDotter struct{ Result *DotNode }

func (n *testGraphNodeDotter) Name() string                      { return n.Result.Name }
func (n *testGraphNodeDotter) DotNode(string, *DotOpts) *DotNode { return n.Result }

const testGraphDotQuotedStr = `digraph {
	compound = "true"
	newrank = "true"
	subgraph "root" {
		"[root] name[\"with-quotes\"]" -> "[root] other"
	}
}`

const testGraphDotBasicStr = `digraph {
	compound = "true"
	newrank = "true"
	subgraph "root" {
		"[root] 1" -> "[root] 3"
	}
}
`

const testGraphDotEmptyStr = `digraph {
	compound = "true"
	newrank = "true"
	subgraph "root" {
	}
}`

const testGraphDotAttrsStr = `digraph {
	compound = "true"
	newrank = "true"
	subgraph "root" {
		"[root] foo" [foo = "bar"]
	}
}`
