package watch

import (
	"testing"
)

func TestWatchItems(t *testing.T) {
	// Creates an empty set of items
	wi := NewItems()
	if len(wi) != 0 {
		t.Fatalf("expect 0 items, got: %#v", wi)
	}

	// Creates a new set of supplied items
	wi = NewItems(Item{Table: "foo"})
	if len(wi) != 1 {
		t.Fatalf("expected 1 item, got: %#v", wi)
	}

	// Adding items works
	wi.Add(Item{Node: "bar"})
	if len(wi) != 2 {
		t.Fatalf("expected 2 items, got: %#v", wi)
	}

	// Adding duplicates auto-dedupes
	wi.Add(Item{Table: "foo"})
	if len(wi) != 2 {
		t.Fatalf("expected 2 items, got: %#v", wi)
	}
}
