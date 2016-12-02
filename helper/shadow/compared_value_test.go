package shadow

import (
	"testing"
	"time"
)

func TestComparedValue(t *testing.T) {
	v := &ComparedValue{
		Func: func(k, v interface{}) bool { return k == v },
	}

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
	v.SetValue("foo")
	val := <-valueCh

	// Verify
	if val != "foo" {
		t.Fatalf("bad: %#v", val)
	}

	// We should get the next value
	val = v.Value("foo")
	if val != "foo" {
		t.Fatalf("bad: %#v", val)
	}
}

func TestComparedValue_setFirst(t *testing.T) {
	v := &ComparedValue{
		Func: func(k, v interface{}) bool { return k == v },
	}

	// Set the value
	v.SetValue("foo")
	val := v.Value("foo")

	// Verify
	if val != "foo" {
		t.Fatalf("bad: %#v", val)
	}
}

func TestComparedValueOk(t *testing.T) {
	v := &ComparedValue{
		Func: func(k, v interface{}) bool { return k == v },
	}

	// Try
	val, ok := v.ValueOk("foo")
	if ok {
		t.Fatal("should not be ok")
	}

	// Set
	v.SetValue("foo")

	// Try again
	val, ok = v.ValueOk("foo")
	if !ok {
		t.Fatal("should be ok")
	}

	// Verify
	if val != "foo" {
		t.Fatalf("bad: %#v", val)
	}
}

func TestComparedValueClose(t *testing.T) {
	v := &ComparedValue{
		Func: func(k, v interface{}) bool { return k == v },
	}

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

func TestComparedValueClose_blocked(t *testing.T) {
	var v ComparedValue

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

func TestComparedValueClose_existing(t *testing.T) {
	var v ComparedValue

	// Set a value
	v.SetValue("foo")

	// Close
	v.Close()

	// Try again
	val, ok := v.ValueOk("foo")
	if !ok {
		t.Fatal("should be ok")
	}

	// Verify
	if val != "foo" {
		t.Fatalf("bad: %#v", val)
	}
}

func TestComparedValueClose_existingBlocked(t *testing.T) {
	var v ComparedValue

	// Start reading this should be blocking
	valueCh := make(chan interface{})
	go func() {
		valueCh <- v.Value("foo")
	}()

	// Wait
	time.Sleep(10 * time.Millisecond)

	// Set a value
	v.SetValue("foo")

	// Close
	v.Close()

	// Try again
	val, ok := v.ValueOk("foo")
	if !ok {
		t.Fatal("should be ok")
	}

	// Verify
	if val != "foo" {
		t.Fatalf("bad: %#v", val)
	}
}
