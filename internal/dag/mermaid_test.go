// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package dag

import (
	"strings"
	"testing"
)

type testVertex struct{ n string }

func (t *testVertex) Name() string { return t.n }

func (t *testVertex) DotNode(title string, opts *DotOpts) *DotNode {
	return &DotNode{Name: title, Attrs: map[string]string{"label": t.n}}
}

func TestMermaidBasic(t *testing.T) {
	g := &Graph{}

	a := &testVertex{n: "a"}
	b := &testVertex{n: "b"}
	c := &testVertex{n: "c"}

	g.Add(a)
	g.Add(b)
	g.Add(c)

	g.Connect(BasicEdge(a, b))
	g.Connect(BasicEdge(b, c))

	mg := newMarshalGraph("root", g)
	out := string(mg.Mermaid(&DotOpts{DrawCycles: true, MaxDepth: -1, Verbose: true}))

	if !strings.Contains(out, "a") || !strings.Contains(out, "b") || !strings.Contains(out, "c") {
		t.Fatalf("expected nodes a, b, c in output, got: %s", out)
	}

	if !strings.Contains(out, "-->") {
		t.Fatalf("expected edges in mermaid output, got: %s", out)
	}
}
