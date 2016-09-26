package shadow

import (
	"testing"
	"time"
)

func TestValue(t *testing.T) {
	var v Value

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

	// We should be able to ask for the value again immediately
	if val := v.Value(); val != 42 {
		t.Fatalf("bad: %#v", val)
	}

	// We can change the value
	v.SetValue(84)
	if val := v.Value(); val != 84 {
		t.Fatalf("bad: %#v", val)
	}
}
