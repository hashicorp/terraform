package dag

import (
	"reflect"
	"sync"
	"testing"
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
