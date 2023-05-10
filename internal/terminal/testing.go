// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package terminal

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
)

// StreamsForTesting is a helper for test code that is aiming to test functions
// that interact with the input and output streams.
//
// This particular function is for the simple case of a function that only
// produces output: the returned input stream is connected to the system's
// "null device", as if a user had run Terraform with I/O redirection like
// </dev/null on Unix. It also configures the output as a pipe rather than
// as a terminal, and so can't be used to test whether code is able to adapt
// to different terminal widths.
//
// The return values are a Streams object ready to pass into a function under
// test, and a callback function for the test itself to call afterwards
// in order to obtain any characters that were written to the streams. Once
// you call the close function, the Streams object becomes invalid and must
// not be used anymore. Any caller of this function _must_ call close before
// its test concludes, even if it doesn't intend to check the output, or else
// it will leak resources.
//
// Since this function is for testing only, for convenience it will react to
// any setup errors by logging a message to the given testing.T object and
// then failing the test, preventing any later code from running.
func StreamsForTesting(t *testing.T) (streams *Streams, close func(*testing.T) *TestOutput) {
	stdinR, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("failed to open /dev/null to represent stdin: %s", err)
	}

	// (Although we only have StreamsForTesting right now, it seems plausible
	// that we'll want some other similar helpers for more complicated
	// situations, such as codepaths that need to read from Stdin or
	// tests for whether a function responds properly to terminal width.
	// In that case, we'd probably want to factor out the core guts of this
	// which set up the pipe *os.File values and the goroutines, but then
	// let each caller produce its own Streams wrapping around those. For
	// now though, it's simpler to just have this whole implementation together
	// in one function.)

	// Our idea of streams is only a very thin wrapper around OS-level file
	// descriptors, so in order to produce a realistic implementation for
	// the code under test while still allowing us to capture the output
	// we'll OS-level pipes and concurrently copy anything we read from
	// them into the output object.
	outp := &TestOutput{}
	var lock sync.Mutex // hold while appending to outp
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %s", err)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stderr pipe: %s", err)
	}
	var wg sync.WaitGroup // for waiting until our goroutines have exited

	// We need an extra goroutine for each of the pipes so we can block
	// on reading both of them alongside the caller hopefully writing to
	// the write sides.
	wg.Add(2)
	consume := func(r *os.File, isErr bool) {
		var buf [1024]byte
		for {
			n, err := r.Read(buf[:])
			if err != nil {
				if err != io.EOF {
					// We aren't allowed to write to the testing.T from
					// a different goroutine than it was created on, but
					// encountering other errors would be weird here anyway
					// so we'll just panic. (If we were to just ignore this
					// and then drop out of the loop then we might deadlock
					// anyone still trying to write to the write end.)
					panic(fmt.Sprintf("failed to read from pipe: %s", err))
				}
				break
			}
			lock.Lock()
			outp.parts = append(outp.parts, testOutputPart{
				isErr: isErr,
				bytes: append(([]byte)(nil), buf[:n]...), // copy so we can reuse the buffer
			})
			lock.Unlock()
		}
		wg.Done()
	}
	go consume(stdoutR, false)
	go consume(stderrR, true)

	close = func(t *testing.T) *TestOutput {
		err := stdinR.Close()
		if err != nil {
			t.Errorf("failed to close stdin handle: %s", err)
		}

		// We'll close both of the writer streams now, which should in turn
		// cause both of the "consume" goroutines above to terminate by
		// encountering io.EOF.
		err = stdoutW.Close()
		if err != nil {
			t.Errorf("failed to close stdout pipe: %s", err)
		}
		err = stderrW.Close()
		if err != nil {
			t.Errorf("failed to close stderr pipe: %s", err)
		}

		// The above error cases still allow this to complete and thus
		// potentially allow the test to report its own result, but will
		// ensure that the test doesn't pass while also leaking resources.

		// Wait for the stream-copying goroutines to finish anything they
		// are working on before we return, or else we might miss some
		// late-arriving writes.
		wg.Wait()
		return outp
	}

	return &Streams{
		Stdout: &OutputStream{
			File: stdoutW,
		},
		Stderr: &OutputStream{
			File: stderrW,
		},
		Stdin: &InputStream{
			File: stdinR,
		},
	}, close
}

// TestOutput is a type used to return the results from the various stream
// testing helpers. It encapsulates any captured writes to the output and
// error streams, and has methods to consume that data in some different ways
// to allow for a few different styles of testing.
type TestOutput struct {
	parts []testOutputPart
}

type testOutputPart struct {
	// isErr is true if this part was written to the error stream, or false
	// if it was written to the output stream.
	isErr bool

	// bytes are the raw bytes that were written
	bytes []byte
}

// All returns the output written to both the Stdout and Stderr streams,
// interleaved together in the order of writing in a single string.
func (o TestOutput) All() string {
	buf := &strings.Builder{}
	for _, part := range o.parts {
		buf.Write(part.bytes)
	}
	return buf.String()
}

// Stdout returns the output written to just the Stdout stream, ignoring
// anything that was written to the Stderr stream.
func (o TestOutput) Stdout() string {
	buf := &strings.Builder{}
	for _, part := range o.parts {
		if part.isErr {
			continue
		}
		buf.Write(part.bytes)
	}
	return buf.String()
}

// Stderr returns the output written to just the Stderr stream, ignoring
// anything that was written to the Stdout stream.
func (o TestOutput) Stderr() string {
	buf := &strings.Builder{}
	for _, part := range o.parts {
		if !part.isErr {
			continue
		}
		buf.Write(part.bytes)
	}
	return buf.String()
}
