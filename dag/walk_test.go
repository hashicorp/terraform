package dag

import (
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestWalker_basic(t *testing.T) {
	var g Graph
	g.Add(1)
	g.Add(2)
	g.Connect(BasicEdge(1, 2))

	// Run it a bunch of times since it is timing dependent
	for i := 0; i < 50; i++ {
		var order []interface{}
		w := &walker{Callback: walkCbRecord(&order)}
		w.Update(g.vertices, g.edges)

		// Wait
		if err := w.Wait(); err != nil {
			t.Fatalf("err: %s", err)
		}

		// Check
		expected := []interface{}{1, 2}
		if !reflect.DeepEqual(order, expected) {
			t.Fatalf("bad: %#v", order)
		}
	}
}

func TestWalker_newVertex(t *testing.T) {
	// Run it a bunch of times since it is timing dependent
	for i := 0; i < 50; i++ {
		var g Graph
		g.Add(1)
		g.Add(2)
		g.Connect(BasicEdge(1, 2))

		var order []interface{}
		w := &walker{Callback: walkCbRecord(&order)}
		w.Update(g.vertices, g.edges)

		// Wait a bit
		time.Sleep(10 * time.Millisecond)

		// Update the graph
		g.Add(3)
		w.Update(g.vertices, g.edges)

		// Update the graph again but with the same vertex
		g.Add(3)
		w.Update(g.vertices, g.edges)

		// Wait
		if err := w.Wait(); err != nil {
			t.Fatalf("err: %s", err)
		}

		// Check
		expected := []interface{}{1, 2, 3}
		if !reflect.DeepEqual(order, expected) {
			t.Fatalf("bad: %#v", order)
		}
	}
}

func TestWalker_removeVertex(t *testing.T) {
	// Run it a bunch of times since it is timing dependent
	for i := 0; i < 50; i++ {
		var g Graph
		g.Add(1)
		g.Add(2)
		g.Connect(BasicEdge(1, 2))

		// Record function
		var order []interface{}
		recordF := walkCbRecord(&order)

		// Build a callback that delays until we close a channel
		gateCh := make(chan struct{})
		cb := func(v Vertex) error {
			if v == 1 {
				<-gateCh
			}

			return recordF(v)
		}

		// Add the initial vertices
		w := &walker{Callback: cb}
		w.Update(g.vertices, g.edges)

		// Remove a vertex
		g.Remove(2)
		w.Update(g.vertices, g.edges)

		// Open gate
		close(gateCh)

		// Wait
		if err := w.Wait(); err != nil {
			t.Fatalf("err: %s", err)
		}

		// Check
		expected := []interface{}{1}
		if !reflect.DeepEqual(order, expected) {
			t.Fatalf("bad: %#v", order)
		}
	}
}

func TestWalker_newEdge(t *testing.T) {
	// Run it a bunch of times since it is timing dependent
	for i := 0; i < 50; i++ {
		var g Graph
		g.Add(1)
		g.Add(2)
		g.Connect(BasicEdge(1, 2))

		// Record function
		var order []interface{}
		recordF := walkCbRecord(&order)

		// Build a callback that delays until we close a channel
		var w *walker
		cb := func(v Vertex) error {
			if v == 1 {
				g.Add(3)
				g.Connect(BasicEdge(3, 2))
				w.Update(g.vertices, g.edges)
			}

			return recordF(v)
		}

		// Add the initial vertices
		w = &walker{Callback: cb}
		w.Update(g.vertices, g.edges)

		// Wait
		if err := w.Wait(); err != nil {
			t.Fatalf("err: %s", err)
		}

		// Check
		expected := []interface{}{1, 3, 2}
		if !reflect.DeepEqual(order, expected) {
			t.Fatalf("bad: %#v", order)
		}
	}
}

// walkCbRecord is a test helper callback that just records the order called.
func walkCbRecord(order *[]interface{}) WalkFunc {
	var l sync.Mutex
	return func(v Vertex) error {
		l.Lock()
		defer l.Unlock()
		*order = append(*order, v)
		return nil
	}
}
