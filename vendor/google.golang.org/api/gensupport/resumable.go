// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gensupport

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/net/context/ctxhttp"
)

const (
	// statusResumeIncomplete is the code returned by the Google uploader when the transfer is not yet complete.
	statusResumeIncomplete = 308
)

// uploadPause determines the delay between failed upload attempts
// TODO(mcgreevy): improve this retry mechanism.
var uploadPause = 1 * time.Second

// ResumableUpload is used by the generated APIs to provide resumable uploads.
// It is not used by developers directly.
type ResumableUpload struct {
	Client *http.Client
	// URI is the resumable resource destination provided by the server after specifying "&uploadType=resumable".
	URI       string
	UserAgent string // User-Agent for header of the request
	// Media is the object being uploaded.
	Media *ResumableBuffer
	// MediaType defines the media type, e.g. "image/jpeg".
	MediaType string

	mu       sync.Mutex // guards progress
	progress int64      // number of bytes uploaded so far

	// Callback is an optional function that will be periodically called with the cumulative number of bytes uploaded.
	Callback func(int64)
}

// Progress returns the number of bytes uploaded at this point.
func (rx *ResumableUpload) Progress() int64 {
	rx.mu.Lock()
	defer rx.mu.Unlock()
	return rx.progress
}

func (rx *ResumableUpload) transferChunks(ctx context.Context) (*http.Response, error) {
	var res *http.Response
	var err error

	for {
		select { // Check for cancellation
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		chunk, off, size, e := rx.Media.Chunk()
		reqSize := int64(size)
		done := e == io.EOF

		if !done && e != nil {
			return nil, e
		}

		req, _ := http.NewRequest("POST", rx.URI, chunk)
		req.ContentLength = reqSize
		var contentRange string
		if done {
			if reqSize == 0 {
				contentRange = fmt.Sprintf("bytes */%v", off)
			} else {
				contentRange = fmt.Sprintf("bytes %v-%v/%v", off, off+reqSize-1, off+reqSize)
			}
		} else {
			contentRange = fmt.Sprintf("bytes %v-%v/*", off, off+reqSize-1)
		}
		req.Header.Set("Content-Range", contentRange)
		req.Header.Set("Content-Type", rx.MediaType)
		req.Header.Set("User-Agent", rx.UserAgent)
		res, err = ctxhttp.Do(ctx, rx.Client, req)

		success := err == nil && res.StatusCode == statusResumeIncomplete || res.StatusCode == http.StatusOK
		if success && reqSize > 0 {
			rx.mu.Lock()
			rx.progress = off + reqSize // number of bytes sent so far
			rx.mu.Unlock()
			if rx.Callback != nil {
				rx.Callback(off + reqSize)
			}
		}
		if err != nil || res.StatusCode != statusResumeIncomplete {
			break
		}
		rx.Media.Next()
		res.Body.Close()
	}
	return res, err
}

// Upload starts the process of a resumable upload with a cancellable context.
// It retries indefinitely (with a pause of uploadPause between attempts) until cancelled.
// It is called from the auto-generated API code and is not visible to the user.
// rx is private to the auto-generated API code.
// Exactly one of resp or err will be nil.  If resp is non-nil, the caller must call resp.Body.Close.
func (rx *ResumableUpload) Upload(ctx context.Context) (resp *http.Response, err error) {
	for {
		resp, err = rx.transferChunks(ctx)
		// It's possible for err and resp to both be non-nil here, but we expose a simpler
		// contract to our callers: exactly one of resp and err will be non-nil.  This means
		// that any response body must be closed here before returning a non-nil error.
		if err != nil {
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
			}
			return nil, err
		}
		if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
			return resp, nil
		}
		resp.Body.Close()
		select { // Check for cancellation
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(uploadPause):
		}
	}
}
