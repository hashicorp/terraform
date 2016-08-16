package signalwrapper

import (
	"errors"
	"testing"
	"time"
)

func TestWrapped_goodCh(t *testing.T) {
	errVal := errors.New("hi")
	f := func(<-chan struct{}) error { return errVal }
	err := <-Run(f).ErrCh
	if err != errVal {
		t.Fatalf("bad: %#v", err)
	}
}

func TestWrapped_goodWait(t *testing.T) {
	errVal := errors.New("hi")
	f := func(<-chan struct{}) error {
		time.Sleep(10 * time.Millisecond)
		return errVal
	}

	wrapped := Run(f)

	// Once
	{
		err := wrapped.Wait()
		if err != errVal {
			t.Fatalf("bad: %#v", err)
		}
	}

	// Again
	{
		err := wrapped.Wait()
		if err != errVal {
			t.Fatalf("bad: %#v", err)
		}
	}
}

func TestWrapped_cancel(t *testing.T) {
	errVal := errors.New("hi")
	f := func(ch <-chan struct{}) error {
		<-ch
		return errVal
	}

	wrapped := Run(f)

	// Once
	{
		err := wrapped.Cancel()
		if err != errVal {
			t.Fatalf("bad: %#v", err)
		}
	}

	// Again
	{
		err := wrapped.Cancel()
		if err != errVal {
			t.Fatalf("bad: %#v", err)
		}
	}
}
