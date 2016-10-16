package api

import (
	"io"
	"reflect"
	"testing"
	"time"
)

func TestFS_FrameReader(t *testing.T) {
	// Create a channel of the frames and a cancel channel
	framesCh := make(chan *StreamFrame, 3)
	cancelCh := make(chan struct{})

	r := NewFrameReader(framesCh, cancelCh)

	// Create some frames and send them
	f1 := &StreamFrame{
		File:   "foo",
		Offset: 5,
		Data:   []byte("hello"),
	}
	f2 := &StreamFrame{
		File:   "foo",
		Offset: 10,
		Data:   []byte(", wor"),
	}
	f3 := &StreamFrame{
		File:   "foo",
		Offset: 12,
		Data:   []byte("ld"),
	}
	framesCh <- f1
	framesCh <- f2
	framesCh <- f3
	close(framesCh)

	expected := []byte("hello, world")

	// Read a little
	p := make([]byte, 12)

	n, err := r.Read(p[:5])
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if off := r.Offset(); off != n {
		t.Fatalf("unexpected read bytes: got %v; wanted %v", n, off)
	}

	off := n
	for {
		n, err = r.Read(p[off:])
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("Read failed: %v", err)
		}
		off += n
	}

	if !reflect.DeepEqual(p, expected) {
		t.Fatalf("read %q, wanted %q", string(p), string(expected))
	}

	if err := r.Close(); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}
	if _, ok := <-cancelCh; ok {
		t.Fatalf("Close() didn't close cancel channel")
	}
	if len(expected) != r.Offset() {
		t.Fatalf("offset %d, wanted %d", r.Offset(), len(expected))
	}
}

func TestFS_FrameReader_Unblock(t *testing.T) {
	// Create a channel of the frames and a cancel channel
	framesCh := make(chan *StreamFrame, 3)
	cancelCh := make(chan struct{})

	r := NewFrameReader(framesCh, cancelCh)
	r.SetUnblockTime(10 * time.Millisecond)

	// Read a little
	p := make([]byte, 12)

	n, err := r.Read(p)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if n != 0 {
		t.Fatalf("should have unblocked")
	}

	// Unset the unblock
	r.SetUnblockTime(0)

	resultCh := make(chan struct{})
	go func() {
		r.Read(p)
		close(resultCh)
	}()

	select {
	case <-resultCh:
		t.Fatalf("shouldn't have unblocked")
	case <-time.After(300 * time.Millisecond):
	}
}
