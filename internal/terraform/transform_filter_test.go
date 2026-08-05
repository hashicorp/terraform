// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/dag"
)

func TestTransformFilter(t *testing.T) {
	t.Run("empty graph", func(t *testing.T) {
		var g Graph
		tf := &TransformFilter{
			Keep: func(v dag.Vertex) bool {
				return true
			},
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}

		actual := strings.TrimSpace(g.String())
		if actual != "" {
			t.Fatalf("expected empty graph, got:\n%s", actual)
		}
	})

	t.Run("keep all", func(t *testing.T) {
		var g Graph
		g.Add("a")
		g.Add("b")
		g.Add("c")
		g.Connect(dag.BasicEdge("a", "b"))
		g.Connect(dag.BasicEdge("b", "c"))

		tf := &TransformFilter{
			Keep: func(v dag.Vertex) bool {
				return true
			},
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}

		actual := strings.TrimSpace(g.String())
		expected := strings.TrimSpace(`
a
  b
b
  c
c
`)
		if actual != expected {
			t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
		}
	})

	t.Run("remove all", func(t *testing.T) {
		var g Graph
		g.Add("a")
		g.Add("b")
		g.Add("c")
		g.Connect(dag.BasicEdge("a", "b"))
		g.Connect(dag.BasicEdge("b", "c"))

		tf := &TransformFilter{
			Keep: func(v dag.Vertex) bool {
				return false
			},
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}

		actual := strings.TrimSpace(g.String())
		if actual != "" {
			t.Fatalf("expected empty graph, got:\n%s", actual)
		}
	})

	t.Run("keep node preserves its dependencies", func(t *testing.T) {
		// a -> b -> c
		// Keep only "a"; "b" and "c" should be preserved as ancestors.
		var g Graph
		g.Add("a")
		g.Add("b")
		g.Add("c")
		g.Connect(dag.BasicEdge("a", "b"))
		g.Connect(dag.BasicEdge("b", "c"))

		tf := &TransformFilter{
			Keep: func(v dag.Vertex) bool {
				return v.(string) == "a"
			},
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}

		actual := strings.TrimSpace(g.String())
		expected := strings.TrimSpace(`
a
  b
b
  c
c
`)
		if actual != expected {
			t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
		}
	})

	t.Run("keep leaf removes dependents", func(t *testing.T) {
		// a -> b -> c
		// Keep only "c"; "a" and "b" are not ancestors of "c" so they
		// should be removed.
		var g Graph
		g.Add("a")
		g.Add("b")
		g.Add("c")
		g.Connect(dag.BasicEdge("a", "b"))
		g.Connect(dag.BasicEdge("b", "c"))

		tf := &TransformFilter{
			Keep: func(v dag.Vertex) bool {
				return v.(string) == "c"
			},
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}

		actual := strings.TrimSpace(g.String())
		expected := "c"
		if actual != expected {
			t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
		}
	})

	t.Run("keep middle preserves dependencies and removes dependents", func(t *testing.T) {
		// a -> b -> c
		// Keep "b"; "c" is an ancestor and stays, "a" depends on "b"
		// but is not an ancestor so it is removed.
		var g Graph
		g.Add("a")
		g.Add("b")
		g.Add("c")
		g.Connect(dag.BasicEdge("a", "b"))
		g.Connect(dag.BasicEdge("b", "c"))

		tf := &TransformFilter{
			Keep: func(v dag.Vertex) bool {
				return v.(string) == "b"
			},
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}

		actual := strings.TrimSpace(g.String())
		expected := strings.TrimSpace(`
b
  c
c
`)
		if actual != expected {
			t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
		}
	})

	t.Run("diamond keep root preserves all", func(t *testing.T) {
		// a -> b -> d
		// a -> c -> d
		// Keep "a"; everything is an ancestor of "a" so nothing is removed.
		var g Graph
		g.Add("a")
		g.Add("b")
		g.Add("c")
		g.Add("d")
		g.Connect(dag.BasicEdge("a", "b"))
		g.Connect(dag.BasicEdge("a", "c"))
		g.Connect(dag.BasicEdge("b", "d"))
		g.Connect(dag.BasicEdge("c", "d"))

		tf := &TransformFilter{
			Keep: func(v dag.Vertex) bool {
				return v.(string) == "a"
			},
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}

		actual := strings.TrimSpace(g.String())
		expected := strings.TrimSpace(`
a
  b
  c
b
  d
c
  d
d
`)
		if actual != expected {
			t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
		}
	})

	t.Run("diamond keep one branch", func(t *testing.T) {
		// a -> b -> d
		// a -> c -> d
		// Keep "b"; "d" is an ancestor of "b" so it stays. "a" and "c"
		// are not ancestors of "b" so they are removed.
		var g Graph
		g.Add("a")
		g.Add("b")
		g.Add("c")
		g.Add("d")
		g.Connect(dag.BasicEdge("a", "b"))
		g.Connect(dag.BasicEdge("a", "c"))
		g.Connect(dag.BasicEdge("b", "d"))
		g.Connect(dag.BasicEdge("c", "d"))

		tf := &TransformFilter{
			Keep: func(v dag.Vertex) bool {
				return v.(string) == "b"
			},
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}

		actual := strings.TrimSpace(g.String())
		expected := strings.TrimSpace(`
b
  d
d
`)
		if actual != expected {
			t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
		}
	})

	t.Run("disconnected nodes are removed", func(t *testing.T) {
		// a -> b, c (standalone)
		// Keep "a"; "b" is preserved as ancestor, "c" has no connection
		// and is removed.
		var g Graph
		g.Add("a")
		g.Add("b")
		g.Add("c")
		g.Connect(dag.BasicEdge("a", "b"))

		tf := &TransformFilter{
			Keep: func(v dag.Vertex) bool {
				return v.(string) == "a"
			},
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}

		actual := strings.TrimSpace(g.String())
		expected := strings.TrimSpace(`
a
  b
b
`)
		if actual != expected {
			t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
		}
	})

	t.Run("multiple kept nodes merge their ancestors", func(t *testing.T) {
		// a -> b -> d
		// c -> d
		// Keep "a" and "c"; their combined ancestors are "b" and "d",
		// so the entire graph is preserved.
		var g Graph
		g.Add("a")
		g.Add("b")
		g.Add("c")
		g.Add("d")
		g.Connect(dag.BasicEdge("a", "b"))
		g.Connect(dag.BasicEdge("b", "d"))
		g.Connect(dag.BasicEdge("c", "d"))

		tf := &TransformFilter{
			Keep: func(v dag.Vertex) bool {
				s := v.(string)
				return s == "a" || s == "c"
			},
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}

		actual := strings.TrimSpace(g.String())
		expected := strings.TrimSpace(`
a
  b
b
  d
c
  d
d
`)
		if actual != expected {
			t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
		}
	})

	t.Run("shared dependency kept through one branch", func(t *testing.T) {
		// a -> c
		// b -> c
		// Keep "a"; "c" is an ancestor and stays, "b" is removed.
		var g Graph
		g.Add("a")
		g.Add("b")
		g.Add("c")
		g.Connect(dag.BasicEdge("a", "c"))
		g.Connect(dag.BasicEdge("b", "c"))

		tf := &TransformFilter{
			Keep: func(v dag.Vertex) bool {
				return v.(string) == "a"
			},
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}

		actual := strings.TrimSpace(g.String())
		expected := strings.TrimSpace(`
a
  c
c
`)
		if actual != expected {
			t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
		}
	})

	t.Run("single node kept", func(t *testing.T) {
		var g Graph
		g.Add("a")

		tf := &TransformFilter{
			Keep: func(v dag.Vertex) bool {
				return true
			},
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}

		actual := strings.TrimSpace(g.String())
		expected := "a"
		if actual != expected {
			t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
		}
	})

	t.Run("single node removed", func(t *testing.T) {
		var g Graph
		g.Add("a")

		tf := &TransformFilter{
			Keep: func(v dag.Vertex) bool {
				return false
			},
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}

		actual := strings.TrimSpace(g.String())
		if actual != "" {
			t.Fatalf("expected empty graph, got:\n%s", actual)
		}
	})
}
