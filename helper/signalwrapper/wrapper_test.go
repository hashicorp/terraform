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

func TestWrapped_waitAndCancel(t *testing.T) {
	errVal := errors.New("hi")
	readyCh := make(chan struct{})
	f := func(ch <-chan struct{}) error {
		<-ch
		<-readyCh
		return errVal
	}

	wrapped := Run(f)

	// Run both cancel and wait and wait some time to hope they're
	// scheduled. We could _ensure_ both are scheduled by using some
	// more lines of code but this is probably just good enough.
	go wrapped.Cancel()
	go wrapped.Wait()
	close(readyCh)
	time.Sleep(10 * time.Millisecond)

	// Check it by calling Cancel again
	err := wrapped.Cancel()
	if err != errVal {
		t.Fatalf("bad: %#v", err)
	}
}
