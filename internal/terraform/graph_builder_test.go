// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"

	"github.com/hashicorp/terraform/internal/dag"
)

func TestBasicGraphBuilder_impl(t *testing.T) {
	var _ GraphBuilder = new(BasicGraphBuilder)
}

type testV int

func (v testV) Name() string {
	return fmt.Sprint(v)
}

func TestBasicGraphBuilder(t *testing.T) {
	b := &BasicGraphBuilder{
		Steps: []GraphTransformer{
			&testBasicGraphBuilderTransform{testV(1)},
		},
	}

	g, err := b.Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if g.Path.String() != addrs.RootModuleInstance.String() {
		t.Fatalf("wrong module path %q", g.Path)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testBasicGraphBuilderStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestBasicGraphBuilder_validate(t *testing.T) {
	b := &BasicGraphBuilder{
		Steps: []GraphTransformer{
			&testBasicGraphBuilderTransform{testV(1)},
			&testBasicGraphBuilderTransform{testV(2)},
		},
	}

	_, err := b.Build(addrs.RootModuleInstance)
	if err == nil {
		t.Fatal("should error")
	}
}

type testBasicGraphBuilderTransform struct {
	V dag.Vertex
}

func (t *testBasicGraphBuilderTransform) Transform(g *Graph) error {
	g.Add(t.V)
	return nil
}

const testBasicGraphBuilderStr = `
1
`
