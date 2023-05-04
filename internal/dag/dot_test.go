// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package dag

import (
	"reflect"
	"testing"
)

func TestGraphDot_opts(t *testing.T) {
	var v testDotVertex
	var g Graph
	g.Add(&v)

	opts := &DotOpts{MaxDepth: 42}
	actual := g.Dot(opts)
	if len(actual) == 0 {
		t.Fatal("should not be empty")
	}

	if !v.DotNodeCalled {
		t.Fatal("should call DotNode")
	}
	if !reflect.DeepEqual(v.DotNodeOpts, opts) {
		t.Fatalf("bad; %#v", v.DotNodeOpts)
	}
}

type testDotVertex struct {
	DotNodeCalled bool
	DotNodeTitle  string
	DotNodeOpts   *DotOpts
	DotNodeReturn *DotNode
}

func (v *testDotVertex) DotNode(title string, opts *DotOpts) *DotNode {
	v.DotNodeCalled = true
	v.DotNodeTitle = title
	v.DotNodeOpts = opts
	return v.DotNodeReturn
}
