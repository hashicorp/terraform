package shadow

import (
	"testing"
	"time"
)

func TestClose(t *testing.T) {
	var foo struct {
		A Value
		B KeyedValue
	}

	if err := Close(&foo); err != nil {
		t.Fatalf("err: %s", err)
	}

	if v := foo.A.Value(); v != ErrClosed {
		t.Fatalf("bad: %#v", v)
	}
	if v := foo.B.Value("foo"); v != ErrClosed {
		t.Fatalf("bad: %#v", v)
	}
}

func TestClose_nonPtr(t *testing.T) {
	var foo struct{}

	if err := Close(foo); err == nil {
		t.Fatal("should error")
	}
}

func TestClose_unexported(t *testing.T) {
	var foo struct {
		A Value
		b Value
	}

	if err := Close(&foo); err != nil {
		t.Fatalf("err: %s", err)
	}

	if v := foo.A.Value(); v != ErrClosed {
		t.Fatalf("bad: %#v", v)
	}

	// Start trying to get the value
	valueCh := make(chan interface{})
	go func() {
		valueCh <- foo.b.Value()
	}()

	// We should not get the value
	select {
	case <-valueCh:
		t.Fatal("shouldn't receive value")
	case <-time.After(10 * time.Millisecond):
	}

	// Set the value
	foo.b.Close()
	val := <-valueCh

	// Verify
	if val != ErrClosed {
		t.Fatalf("bad: %#v", val)
	}
}
