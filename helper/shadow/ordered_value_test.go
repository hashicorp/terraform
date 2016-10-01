package shadow

import (
	"testing"
	"time"
)

func TestOrderedValue(t *testing.T) {
	var v OrderedValue

	// Start trying to get the value
	valueCh := make(chan interface{})
	go func() {
		valueCh <- v.Value()
	}()

	// We should not get the value
	select {
	case <-valueCh:
		t.Fatal("shouldn't receive value")
	case <-time.After(10 * time.Millisecond):
	}

	// Set the value
	v.SetValue(42)
	val := <-valueCh

	// Verify
	if val != 42 {
		t.Fatalf("bad: %#v", val)
	}

	// We should not get the value again
	go func() {
		valueCh <- v.Value()
	}()
	select {
	case <-valueCh:
		t.Fatal("shouldn't receive value")
	case <-time.After(10 * time.Millisecond):
	}

	// We should get the next value
	v.SetValue(21)
	val = <-valueCh
	if val != 21 {
		t.Fatalf("bad: %#v", val)
	}
}

func TestOrderedValue_setFirst(t *testing.T) {
	var v OrderedValue

	// Set the value
	v.SetValue(42)
	val := v.Value()

	// Verify
	if val != 42 {
		t.Fatalf("bad: %#v", val)
	}

	// We should not get the value again
	valueCh := make(chan interface{})
	go func() {
		valueCh <- v.Value()
	}()
	select {
	case <-valueCh:
		t.Fatal("shouldn't receive value")
	case <-time.After(10 * time.Millisecond):
	}

	// Set a value so the goroutine doesn't hang around
	v.SetValue(1)
}
