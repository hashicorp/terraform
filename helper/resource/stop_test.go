package resource

import (
	"testing"
	"time"
)

func TestStopCh_stop(t *testing.T) {
	stopCh := make(chan struct{})
	ch, _ := StopCh(stopCh, 0)

	// ch should block
	select {
	case <-ch:
		t.Fatal("ch should block")
	case <-time.After(10 * time.Millisecond):
	}

	// Close the stop channel
	close(stopCh)

	// ch should return
	select {
	case <-ch:
	case <-time.After(10 * time.Millisecond):
		t.Fatal("ch should not block")
	}
}

func TestStopCh_done(t *testing.T) {
	stopCh := make(chan struct{})
	ch, doneCh := StopCh(stopCh, 0)

	// ch should block
	select {
	case <-ch:
		t.Fatal("ch should block")
	case <-time.After(10 * time.Millisecond):
	}

	// Close the done channel
	close(doneCh)

	// ch should block
	select {
	case <-ch:
		t.Fatal("ch should block")
	case <-time.After(10 * time.Millisecond):
	}
}

func TestStopCh_timeout(t *testing.T) {
	stopCh := make(chan struct{})
	ch, _ := StopCh(stopCh, 10*time.Millisecond)

	// ch should return
	select {
	case <-ch:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("ch should not block")
	}
}
