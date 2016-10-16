package structs

import (
	"reflect"
	"testing"
)

func TestBitmap(t *testing.T) {
	// Check invalid sizes
	_, err := NewBitmap(0)
	if err == nil {
		t.Fatalf("bad")
	}
	_, err = NewBitmap(7)
	if err == nil {
		t.Fatalf("bad")
	}

	// Create a normal bitmap
	var s uint = 256
	b, err := NewBitmap(s)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if b.Size() != s {
		t.Fatalf("bad size")
	}

	// Set a few bits
	b.Set(0)
	b.Set(255)

	// Verify the bytes
	if b[0] == 0 {
		t.Fatalf("bad")
	}
	if !b.Check(0) {
		t.Fatalf("bad")
	}

	// Verify the bytes
	if b[len(b)-1] == 0 {
		t.Fatalf("bad")
	}
	if !b.Check(255) {
		t.Fatalf("bad")
	}

	// All other bits should be unset
	for i := 1; i < 255; i++ {
		if b.Check(uint(i)) {
			t.Fatalf("bad")
		}
	}

	// Check the indexes
	idxs := b.IndexesInRange(true, 0, 500)
	expected := []int{0, 255}
	if !reflect.DeepEqual(idxs, expected) {
		t.Fatalf("bad: got %#v; want %#v", idxs, expected)
	}

	idxs = b.IndexesInRange(true, 1, 255)
	expected = []int{255}
	if !reflect.DeepEqual(idxs, expected) {
		t.Fatalf("bad: got %#v; want %#v", idxs, expected)
	}

	idxs = b.IndexesInRange(false, 0, 256)
	if len(idxs) != 254 {
		t.Fatalf("bad")
	}

	idxs = b.IndexesInRange(false, 100, 200)
	if len(idxs) != 101 {
		t.Fatalf("bad")
	}

	// Check the copy is correct
	b2, err := b.Copy()
	if err != nil {
		t.Fatalf("bad: %v", err)
	}

	if !reflect.DeepEqual(b, b2) {
		t.Fatalf("bad")
	}

	// Clear
	b.Clear()

	// All bits should be unset
	for i := 0; i < 256; i++ {
		if b.Check(uint(i)) {
			t.Fatalf("bad")
		}
	}
}
