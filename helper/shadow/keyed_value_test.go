package shadow

import (
	"testing"
	"time"
)

func TestKeyedValue(t *testing.T) {
	var v KeyedValue

	// Start trying to get the value
	valueCh := make(chan interface{})
	go func() {
		valueCh <- v.Value("foo")
	}()

	// We should not get the value
	select {
	case <-valueCh:
		t.Fatal("shouldn't receive value")
	case <-time.After(10 * time.Millisecond):
	}

	// Set the value
	v.SetValue("foo", 42)
	val := <-valueCh

	// Verify
	if val != 42 {
		t.Fatalf("bad: %#v", val)
	}

	// We should get the next value
	val = v.Value("foo")
	if val != 42 {
		t.Fatalf("bad: %#v", val)
	}
}

func TestKeyedValue_setFirst(t *testing.T) {
	var v KeyedValue

	// Set the value
	v.SetValue("foo", 42)
	val := v.Value("foo")

	// Verify
	if val != 42 {
		t.Fatalf("bad: %#v", val)
	}
}

func TestKeyedValueOk(t *testing.T) {
	var v KeyedValue

	// Try
	val, ok := v.ValueOk("foo")
	if ok {
		t.Fatal("should not be ok")
	}

	// Set
	v.SetValue("foo", 42)

	// Try again
	val, ok = v.ValueOk("foo")
	if !ok {
		t.Fatal("should be ok")
	}

	// Verify
	if val != 42 {
		t.Fatalf("bad: %#v", val)
	}
}
