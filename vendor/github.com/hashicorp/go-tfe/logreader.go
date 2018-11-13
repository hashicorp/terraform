package tfe

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"time"
)

// LogReader implements io.Reader for streaming logs.
type LogReader struct {
	client      *Client
	ctx         context.Context
	done        func() (bool, error)
	logURL      *url.URL
	offset      int64
	reads       int
	startOfText bool
	endOfText   bool
}

// backoff will perform exponential backoff based on the iteration and
// limited by the provided min and max (in milliseconds) durations.
func backoff(min, max float64, iter int) time.Duration {
	backoff := math.Pow(2, float64(iter)/5) * min
	if backoff > max {
		backoff = max
	}
	return time.Duration(backoff) * time.Millisecond
}

func (r *LogReader) Read(l []byte) (int, error) {
	if written, err := r.read(l); err != io.ErrNoProgress {
		return written, err
	}

	// Loop until we can any data, the context is canceled or the
	// run is finsished. If we would return right away without any
	// data, we could and up causing a io.ErrNoProgress error.
	for r.reads = 1; ; r.reads++ {
		select {
		case <-r.ctx.Done():
			return 0, r.ctx.Err()
		case <-time.After(backoff(500, 2000, r.reads)):
			if written, err := r.read(l); err != io.ErrNoProgress {
				return written, err
			}
		}
	}
}

func (r *LogReader) read(l []byte) (int, error) {
	// Update the query string.
	r.logURL.RawQuery = fmt.Sprintf("limit=%d&offset=%d", len(l), r.offset)

	// Create a new request.
	req, err := http.NewRequest("GET", r.logURL.String(), nil)
	if err != nil {
		return 0, err
	}
	req = req.WithContext(r.ctx)

	// Attach the default headers.
	for k, v := range r.client.headers {
		req.Header[k] = v
	}

	// Retrieve the next chunk.
	resp, err := r.client.http.HTTPClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// Basic response checking.
	if err := checkResponseCode(resp); err != nil {
		return 0, err
	}

	// Read the retrieved chunk.
	written, err := resp.Body.Read(l)
	if err != nil && err != io.EOF {
		// Ignore io.EOF errors returned when reading from the response
		// body as this indicates the end of the chunk and not the end
		// of the logfile.
		return written, err
	}

	if written > 0 {
		// Check for an STX (Start of Text) ASCII control marker.
		if !r.startOfText && l[0] == byte(2) {
			r.startOfText = true

			// Remove the STX marker from the received chunk.
			copy(l[:written-1], l[1:])
			l[written-1] = byte(0)
			r.offset++
			written--

			// Return early if we only received the STX marker.
			if written == 0 {
				return 0, io.ErrNoProgress
			}
		}

		// If we found an STX ASCII control character, start looking for
		// the ETX (End of Text) control character.
		if r.startOfText && l[written-1] == byte(3) {
			r.endOfText = true

			// Remove the ETX marker from the received chunk.
			l[written-1] = byte(0)
			r.offset++
			written--
		}
	}

	// Check if we need to continue the loop and wait 500 miliseconds
	// before checking if there is a new chunk available or that the
	// run is finished and we are done reading all chunks.
	if written == 0 {
		if (r.startOfText && r.endOfText) || // The logstream finished without issues.
			(r.startOfText && r.reads%10 == 0) || // The logstream terminated unexpectedly.
			(!r.startOfText && r.reads > 1) { // The logstream doesn't support STX/ETX.
			done, err := r.done()
			if err != nil {
				return 0, err
			}
			if done {
				return 0, io.EOF
			}
		}
		return 0, io.ErrNoProgress
	}

	// Update the offset for the next read.
	r.offset += int64(written)

	return written, nil
}
