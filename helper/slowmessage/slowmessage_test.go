package slowmessage

import (
	"errors"
	"testing"
	"time"
)

func TestDo(t *testing.T) {
	var sfErr error
	cbCalled := false
	sfCalled := false
	sfSleep := 0 * time.Second

	reset := func() {
		cbCalled = false
		sfCalled = false
		sfErr = nil
	}
	sf := func() error {
		sfCalled = true
		time.Sleep(sfSleep)
		return sfErr
	}
	cb := func() { cbCalled = true }

	// SF is not slow
	reset()
	if err := Do(10*time.Millisecond, sf, cb); err != nil {
		t.Fatalf("err: %s", err)
	}

	if !sfCalled {
		t.Fatal("should call")
	}
	if cbCalled {
		t.Fatal("should not call")
	}

	// SF is not slow (with error)
	reset()
	sfErr = errors.New("error")
	if err := Do(10*time.Millisecond, sf, cb); err == nil {
		t.Fatalf("err: %s", err)
	}

	if !sfCalled {
		t.Fatal("should call")
	}
	if cbCalled {
		t.Fatal("should not call")
	}

	// SF is slow
	reset()
	sfSleep = 50 * time.Millisecond
	if err := Do(10*time.Millisecond, sf, cb); err != nil {
		t.Fatalf("err: %s", err)
	}

	if !sfCalled {
		t.Fatal("should call")
	}
	if !cbCalled {
		t.Fatal("should call")
	}

	// SF is slow (with error)
	reset()
	sfErr = errors.New("error")
	sfSleep = 50 * time.Millisecond
	if err := Do(10*time.Millisecond, sf, cb); err == nil {
		t.Fatalf("err: %s", err)
	}

	if !sfCalled {
		t.Fatal("should call")
	}
	if !cbCalled {
		t.Fatal("should call")
	}
}
