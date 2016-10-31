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

func TestKeyedValueClose(t *testing.T) {
	var v KeyedValue

	// Close
	v.Close()

	// Try again
	val, ok := v.ValueOk("foo")
	if !ok {
		t.Fatal("should be ok")
	}

	// Verify
	if val != ErrClosed {
		t.Fatalf("bad: %#v", val)
	}
}

func TestKeyedValueClose_blocked(t *testing.T) {
	var v KeyedValue

	// Start reading this should be blocking
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

	// Close
	v.Close()

	// Verify
	val := <-valueCh
	if val != ErrClosed {
		t.Fatalf("bad: %#v", val)
	}
}

func TestKeyedValueClose_existing(t *testing.T) {
	var v KeyedValue

	// Set a value
	v.SetValue("foo", "bar")

	// Close
	v.Close()

	// Try again
	val, ok := v.ValueOk("foo")
	if !ok {
		t.Fatal("should be ok")
	}

	// Verify
	if val != "bar" {
		t.Fatalf("bad: %#v", val)
	}
}

func TestKeyedValueClose_existingBlocked(t *testing.T) {
	var v KeyedValue

	// Start reading this should be blocking
	valueCh := make(chan interface{})
	go func() {
		valueCh <- v.Value("foo")
	}()

	// Wait
	time.Sleep(10 * time.Millisecond)

	// Set a value
	v.SetValue("foo", "bar")

	// Close
	v.Close()

	// Try again
	val, ok := v.ValueOk("foo")
	if !ok {
		t.Fatal("should be ok")
	}

	// Verify
	if val != "bar" {
		t.Fatalf("bad: %#v", val)
	}
}

func TestKeyedValueInit(t *testing.T) {
	var v KeyedValue

	v.Init("foo", 42)

	// We should get the value
	val := v.Value("foo")
	if val != 42 {
		t.Fatalf("bad: %#v", val)
	}

	// We should get the value
	val = v.Value("foo")
	if val != 42 {
		t.Fatalf("bad: %#v", val)
	}

	// This should do nothing
	v.Init("foo", 84)

	// We should get the value
	val = v.Value("foo")
	if val != 42 {
		t.Fatalf("bad: %#v", val)
	}
}

func TestKeyedValueInit_set(t *testing.T) {
	var v KeyedValue

	v.SetValue("foo", 42)

	// We should get the value
	val := v.Value("foo")
	if val != 42 {
		t.Fatalf("bad: %#v", val)
	}

	// We should get the value
	val = v.Value("foo")
	if val != 42 {
		t.Fatalf("bad: %#v", val)
	}

	// This should do nothing
	v.Init("foo", 84)

	// We should get the value
	val = v.Value("foo")
	if val != 42 {
		t.Fatalf("bad: %#v", val)
	}
}

func TestKeyedValueWaitForChange(t *testing.T) {
	var v KeyedValue

	// Set a value
	v.SetValue("foo", 42)

	// Start reading this should be blocking
	valueCh := make(chan interface{})
	go func() {
		valueCh <- v.WaitForChange("foo")
	}()

	// We should not get the value
	select {
	case <-valueCh:
		t.Fatal("shouldn't receive value")
	case <-time.After(10 * time.Millisecond):
	}

	// Set a new value
	v.SetValue("foo", 84)

	// Verify
	val := <-valueCh
	if val != 84 {
		t.Fatalf("bad: %#v", val)
	}
}

func TestKeyedValueWaitForChange_initial(t *testing.T) {
	var v KeyedValue

	// Start reading this should be blocking
	valueCh := make(chan interface{})
	go func() {
		valueCh <- v.WaitForChange("foo")
	}()

	// We should not get the value
	select {
	case <-valueCh:
		t.Fatal("shouldn't receive value")
	case <-time.After(10 * time.Millisecond):
	}

	// Set a new value
	v.SetValue("foo", 84)

	// Verify
	val := <-valueCh
	if val != 84 {
		t.Fatalf("bad: %#v", val)
	}
}

func TestKeyedValueWaitForChange_closed(t *testing.T) {
	var v KeyedValue

	// Start reading this should be blocking
	valueCh := make(chan interface{})
	go func() {
		valueCh <- v.WaitForChange("foo")
	}()

	// We should not get the value
	select {
	case <-valueCh:
		t.Fatal("shouldn't receive value")
	case <-time.After(10 * time.Millisecond):
	}

	// Close
	v.Close()

	// Verify
	val := <-valueCh
	if val != ErrClosed {
		t.Fatalf("bad: %#v", val)
	}

	// Set a value
	v.SetValue("foo", 42)

	// Try again
	val = v.WaitForChange("foo")
	if val != ErrClosed {
		t.Fatalf("bad: %#v", val)
	}
}

func TestKeyedValueWaitForChange_closedFirst(t *testing.T) {
	var v KeyedValue

	// Close
	v.Close()

	// Verify
	val := v.WaitForChange("foo")
	if val != ErrClosed {
		t.Fatalf("bad: %#v", val)
	}

	// Set a value
	v.SetValue("foo", 42)

	// Try again
	val = v.WaitForChange("foo")
	if val != ErrClosed {
		t.Fatalf("bad: %#v", val)
	}
}
