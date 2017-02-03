package dag

import (
	"fmt"
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
		w := &Walker{Callback: walkCbRecord(&order)}
		w.Update(&g)

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

func TestWalker_updateNilGraph(t *testing.T) {
	var g Graph
	g.Add(1)
	g.Add(2)
	g.Connect(BasicEdge(1, 2))

	// Run it a bunch of times since it is timing dependent
	for i := 0; i < 50; i++ {
		var order []interface{}
		w := &Walker{Callback: walkCbRecord(&order)}
		w.Update(&g)
		w.Update(nil)

		// Wait
		if err := w.Wait(); err != nil {
			t.Fatalf("err: %s", err)
		}
	}
}

func TestWalker_error(t *testing.T) {
	var g Graph
	g.Add(1)
	g.Add(2)
	g.Add(3)
	g.Add(4)
	g.Connect(BasicEdge(1, 2))
	g.Connect(BasicEdge(2, 3))
	g.Connect(BasicEdge(3, 4))

	// Record function
	var order []interface{}
	recordF := walkCbRecord(&order)

	// Build a callback that delays until we close a channel
	cb := func(v Vertex) error {
		if v == 2 {
			return fmt.Errorf("error!")
		}

		return recordF(v)
	}

	w := &Walker{Callback: cb}
	w.Update(&g)

	// Wait
	if err := w.Wait(); err == nil {
		t.Fatal("expect error")
	}

	// Check
	expected := []interface{}{1}
	if !reflect.DeepEqual(order, expected) {
		t.Fatalf("bad: %#v", order)
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
		w := &Walker{Callback: walkCbRecord(&order)}
		w.Update(&g)

		// Wait a bit
		time.Sleep(10 * time.Millisecond)

		// Update the graph
		g.Add(3)
		w.Update(&g)

		// Update the graph again but with the same vertex
		g.Add(3)
		w.Update(&g)

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
		var w *Walker
		cb := func(v Vertex) error {
			if v == 1 {
				g.Remove(2)
				w.Update(&g)
			}

			return recordF(v)
		}

		// Add the initial vertices
		w = &Walker{Callback: cb}
		w.Update(&g)

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
		var w *Walker
		cb := func(v Vertex) error {
			if v == 1 {
				g.Add(3)
				g.Connect(BasicEdge(3, 2))
				w.Update(&g)
			}

			return recordF(v)
		}

		// Add the initial vertices
		w = &Walker{Callback: cb}
		w.Update(&g)

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

func TestWalker_removeEdge(t *testing.T) {
	// Run it a bunch of times since it is timing dependent
	for i := 0; i < 50; i++ {
		var g Graph
		g.Add(1)
		g.Add(2)
		g.Add(3)
		g.Connect(BasicEdge(1, 2))
		g.Connect(BasicEdge(3, 2))

		// Record function
		var order []interface{}
		recordF := walkCbRecord(&order)

		// The way this works is that our original graph forces
		// the order of 1 => 3 => 2. During the execution of 1, we
		// remove the edge forcing 3 before 2. Then, during the execution
		// of 3, we wait on a channel that is only closed by 2, implicitly
		// forcing 2 before 3 via the callback (and not the graph). If
		// 2 cannot execute before 3 (edge removal is non-functional), then
		// this test will timeout.
		var w *Walker
		gateCh := make(chan struct{})
		cb := func(v Vertex) error {
			if v == 1 {
				g.RemoveEdge(BasicEdge(3, 2))
				w.Update(&g)
			}

			if v == 2 {
				close(gateCh)
			}

			if v == 3 {
				select {
				case <-gateCh:
				case <-time.After(50 * time.Millisecond):
					return fmt.Errorf("timeout 3 waiting for 2")
				}
			}

			return recordF(v)
		}

		// Add the initial vertices
		w = &Walker{Callback: cb}
		w.Update(&g)

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
