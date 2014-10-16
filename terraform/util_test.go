package terraform

import (
	"testing"
	"time"
)

func TestSemaphore(t *testing.T) {
	s := NewSemaphore(2)
	timer := time.AfterFunc(time.Second, func() {
		panic("deadlock")
	})
	defer timer.Stop()

	s.Acquire()
	if !s.TryAcquire() {
		t.Fatalf("should acquire")
	}
	if s.TryAcquire() {
		t.Fatalf("should not acquire")
	}
	s.Release()
	s.Release()

	// This release should panic
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("should panic")
		}
	}()
	s.Release()
}

func TestStrSliceContains(t *testing.T) {
	if strSliceContains(nil, "foo") {
		t.Fatalf("Bad")
	}
	if strSliceContains([]string{}, "foo") {
		t.Fatalf("Bad")
	}
	if strSliceContains([]string{"bar"}, "foo") {
		t.Fatalf("Bad")
	}
	if !strSliceContains([]string{"bar", "foo"}, "foo") {
		t.Fatalf("Bad")
	}
}
