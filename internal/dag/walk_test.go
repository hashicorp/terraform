// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package dag

import (
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestWalker_basic(t *testing.T) {
	var g AcyclicGraph
	g.Add(testV(1))
	g.Add(testV(2))
	g.Connect(testV(1), testV(2))

	// Run it a bunch of times since it is timing dependent
	for range 50 {
		var order []any
		w := NewWalker(walkCbRecord(&order))
		if diags := w.Walk(&g); diags.HasErrors() {
			t.Fatalf("err: %s", diags.ErrWithWarnings())

		}

		// Check
		expected := []any{testV(2), testV(1)}
		if !reflect.DeepEqual(order, expected) {
			t.Errorf("wrong order\ngot:  %#v\nwant: %#v", order, expected)
		}
	}
}

func TestWalker_error(t *testing.T) {
	var g AcyclicGraph
	g.Add(testV(1))
	g.Add(testV(2))
	g.Add(testV(3))
	g.Add(testV(4))
	g.Connect(testV(2), testV(1))
	g.Connect(testV(3), testV(2))
	g.Connect(testV(4), testV(3))

	// Record function
	var order []any
	recordF := walkCbRecord(&order)

	// Build a callback that delays until we close a channel
	cb := func(v Vertex) tfdiags.Diagnostics {
		if v == testV(2) {
			var diags tfdiags.Diagnostics
			diags = diags.Append(fmt.Errorf("error"))
			return diags
		}

		return recordF(v)
	}

	w := NewWalker(cb)
	if diags := w.Walk(&g); !diags.HasErrors() {
		t.Fatal("expect error")
	}

	// Check
	expected := []any{testV(1)}
	if !reflect.DeepEqual(order, expected) {
		t.Errorf("wrong order\ngot:  %#v\nwant: %#v", order, expected)
	}
}

func TestWalker_alwaysRunVertex(t *testing.T) {
	// This tests that an [AlwaysRun] vertex whose dependencies fail still executes.
	// The graph is a > error > c > d.
	// When error is a failure, c executes
	var g AcyclicGraph
	resA := testNamedString("a")
	resErr := testNamedString("error")
	resC := testAlwaysRunVertex("c")
	resD := testNamedString("d")
	g.Add(resA)
	g.Add(resErr)
	g.Add(resC)
	g.Add(resD)
	g.Connect(resErr, resA)
	g.Connect(resC, resErr)
	g.Connect(resD, resC)

	var order []any

	w := NewWalker(func(v Vertex) tfdiags.Diagnostics {
		if v == testNamedString("error") {
			var diags tfdiags.Diagnostics
			diags = diags.Append(fmt.Errorf("error"))
			return diags
		}

		return walkCbRecord(&order)(v)
	})

	if diags := w.Walk(&g); !diags.HasErrors() {
		t.Fatal("expect error")
	}

	expected := []any{resA, resC, resD}
	if !reflect.DeepEqual(order, expected) {
		t.Errorf("wrong order\ngot:  %#v\nwant: %#v", order, expected)
	}
}

// walkCbRecord is a test helper callback that just records the order called.
func walkCbRecord(order *[]any) walkFunc {
	var l sync.Mutex
	return func(v Vertex) tfdiags.Diagnostics {
		l.Lock()
		defer l.Unlock()
		*order = append(*order, v)
		return nil
	}
}
