package linereader

import (
	"io"
	"bufio"
	"sync/atomic"
	"time"
)

// Reader takes an io.Reader and pushes the lines out onto the channel.
type Reader struct {
	Reader  io.Reader
	Timeout time.Duration

	// Ch is the output channel. This will be closed when there are no
	// more lines (io.EOF).
	Ch chan string

	started uint32
}

// New creates a new Reader that reads lines from the io.Reader.
//
// The Reader is already started when returned, so it is unsafe to modify
// any struct fields.
func New(r io.Reader) *Reader {
	result := &Reader{
		Reader:  r,
		Timeout: 100 * time.Millisecond,
		Ch:      make(chan string),
	}

	go result.Run()
	return result
}

// Run reads from the Reader and dispatches lines on the Ch channel.
//
// This blocks and is usually called with `go` prefixed to dispatch onto
// a goroutine. It is safe to call this function multiple times; subsequent
// calls to Run will exit without running.
func (r *Reader) Run() {
	if !atomic.CompareAndSwapUint32(&r.started, 0, 1) {
		return
	}

	// When we're done, close the channel
	defer close(r.Ch)

	// Listen for bytes in a goroutine. We do this so that if we're blocking
	// we can flush the bytes we have after some configured time. There is
	// probably a way to make this a lot faster but this works for now.
	//
	// NOTE: This isn't particularly performant. I'm sure there is a better
	// way to do this instead of sending single bytes on a channel, but it
	// works fine.
	buf := bufio.NewReader(r.Reader)
	byteCh := make(chan byte)
	doneCh := make(chan error)
	go func() {
		defer close(doneCh)
		for {
			b, err := buf.ReadByte()
			if err != nil {
				doneCh <- err
				return
			}

			byteCh <- b
		}
	}()

	lineBuf := make([]byte, 0, 80)
	for {
		var err error
		line := lineBuf[0:0]
		for {
			brk := false

			select {
			case b := <-byteCh:
				brk = b == '\n'
				if !brk {
					line = append(line, b)
				}
			case err = <-doneCh:
				brk = true
			case <-time.After(r.Timeout):
				if len(line) > 0 {
					brk = true
				}
			}

			if brk {
				break
			}
		}

		// If an error occurred and its not an EOF, then report that
		// error to all pipes and exit.
		if err != nil && err != io.EOF {
			break
		}

		// If we're at the end and the line is empty, then return.
		if err == io.EOF && len(line) == 0 {
			break
		}

		// Write out the line
		r.Ch <- string(line)

		// If we hit the end, we're done
		if err == io.EOF {
			break
		}
	}
}
