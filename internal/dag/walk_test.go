package dag

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform/tfdiags"
)

func TestWalker_basic(t *testing.T) {
	var g AcyclicGraph
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
			t.Errorf("wrong order\ngot:  %#v\nwant: %#v", order, expected)
		}
	}
}

func TestWalker_updateNilGraph(t *testing.T) {
	var g AcyclicGraph
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
	var g AcyclicGraph
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
	cb := func(v Vertex) tfdiags.Diagnostics {
		if v == 2 {
			var diags tfdiags.Diagnostics
			diags = diags.Append(fmt.Errorf("error"))
			return diags
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
		t.Errorf("wrong order\ngot:  %#v\nwant: %#v", order, expected)
	}
}

func TestWalker_newVertex(t *testing.T) {
	var g AcyclicGraph
	g.Add(1)
	g.Add(2)
	g.Connect(BasicEdge(1, 2))

	// Record function
	var order []interface{}
	recordF := walkCbRecord(&order)
	done2 := make(chan int)

	// Build a callback that notifies us when 2 has been walked
	var w *Walker
	cb := func(v Vertex) tfdiags.Diagnostics {
		if v == 2 {
			defer close(done2)
		}
		return recordF(v)
	}

	// Add the initial vertices
	w = &Walker{Callback: cb}
	w.Update(&g)

	// if 2 has been visited, the walk is complete so far
	<-done2

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
		t.Errorf("wrong order\ngot:  %#v\nwant: %#v", order, expected)
	}
}

func TestWalker_removeVertex(t *testing.T) {
	var g AcyclicGraph
	g.Add(1)
	g.Add(2)
	g.Connect(BasicEdge(1, 2))

	// Record function
	var order []interface{}
	recordF := walkCbRecord(&order)

	var w *Walker
	cb := func(v Vertex) tfdiags.Diagnostics {
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
		t.Errorf("wrong order\ngot:  %#v\nwant: %#v", order, expected)
	}
}

func TestWalker_newEdge(t *testing.T) {
	var g AcyclicGraph
	g.Add(1)
	g.Add(2)
	g.Connect(BasicEdge(1, 2))

	// Record function
	var order []interface{}
	recordF := walkCbRecord(&order)

	var w *Walker
	cb := func(v Vertex) tfdiags.Diagnostics {
		// record where we are first, otherwise the Updated vertex may get
		// walked before the first visit.
		diags := recordF(v)

		if v == 1 {
			g.Add(3)
			g.Connect(BasicEdge(3, 2))
			w.Update(&g)
		}
		return diags
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
		t.Errorf("wrong order\ngot:  %#v\nwant: %#v", order, expected)
	}
}

func TestWalker_removeEdge(t *testing.T) {
	var g AcyclicGraph
	g.Add(1)
	g.Add(2)
	g.Add(3)
	g.Connect(BasicEdge(1, 2))
	g.Connect(BasicEdge(1, 3))
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
	cb := func(v Vertex) tfdiags.Diagnostics {
		t.Logf("visit vertex %#v", v)
		switch v {
		case 1:
			g.RemoveEdge(BasicEdge(3, 2))
			w.Update(&g)
			t.Logf("removed edge from 3 to 2")

		case 2:
			// this visit isn't completed until we've recorded it
			// Once the visit is official, we can then close the gate to
			// let 3 continue.
			defer close(gateCh)
			defer t.Logf("2 unblocked 3")

		case 3:
			select {
			case <-gateCh:
				t.Logf("vertex 3 gate channel is now closed")
			case <-time.After(500 * time.Millisecond):
				t.Logf("vertex 3 timed out waiting for the gate channel to close")
				var diags tfdiags.Diagnostics
				diags = diags.Append(fmt.Errorf("timeout 3 waiting for 2"))
				return diags
			}
		}

		return recordF(v)
	}

	// Add the initial vertices
	w = &Walker{Callback: cb}
	w.Update(&g)

	// Wait
	if diags := w.Wait(); diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	// Check
	expected := []interface{}{1, 2, 3}
	if !reflect.DeepEqual(order, expected) {
		t.Errorf("wrong order\ngot:  %#v\nwant: %#v", order, expected)
	}
}

// walkCbRecord is a test helper callback that just records the order called.
func walkCbRecord(order *[]interface{}) WalkFunc {
	var l sync.Mutex
	return func(v Vertex) tfdiags.Diagnostics {
		l.Lock()
		defer l.Unlock()
		*order = append(*order, v)
		return nil
	}
}
