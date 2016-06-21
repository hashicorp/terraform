// Copyright 2013 Martini Authors
// Copyright 2014 Unknwon
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package macaron

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"net"
	"net/http"
	"strings"
)

const (
	HeaderAcceptEncoding  = "Accept-Encoding"
	HeaderContentEncoding = "Content-Encoding"
	HeaderContentLength   = "Content-Length"
	HeaderContentType     = "Content-Type"
	HeaderVary            = "Vary"
)

// GzipOptions represents a struct for specifying configuration options for the GZip middleware.
type GzipOptions struct {
	// Compression level. Can be DefaultCompression(-1) or any integer value between BestSpeed(1) and BestCompression(9) inclusive.
	CompressionLevel int
}

func isCompressionLevelValid(level int) bool {
	return level == gzip.DefaultCompression ||
		(level >= gzip.BestSpeed && level <= gzip.BestCompression)
}

func prepareGzipOptions(options []GzipOptions) GzipOptions {
	var opt GzipOptions
	if len(options) > 0 {
		opt = options[0]
	}

	if !isCompressionLevelValid(opt.CompressionLevel) {
		opt.CompressionLevel = gzip.DefaultCompression
	}
	return opt
}

// Gziper returns a Handler that adds gzip compression to all requests.
// Make sure to include the Gzip middleware above other middleware
// that alter the response body (like the render middleware).
func Gziper(options ...GzipOptions) Handler {
	opt := prepareGzipOptions(options)

	return func(ctx *Context) {
		if !strings.Contains(ctx.Req.Header.Get(HeaderAcceptEncoding), "gzip") {
			return
		}

		headers := ctx.Resp.Header()
		headers.Set(HeaderContentEncoding, "gzip")
		headers.Set(HeaderVary, HeaderAcceptEncoding)

		// We've made sure compression level is valid in prepareGzipOptions,
		// no need to check same error again.
		gz, err := gzip.NewWriterLevel(ctx.Resp, opt.CompressionLevel)
		if err != nil {
			panic(err.Error())
		}
		defer gz.Close()

		gzw := gzipResponseWriter{gz, ctx.Resp}
		ctx.Resp = gzw
		ctx.MapTo(gzw, (*http.ResponseWriter)(nil))
		if ctx.Render != nil {
			ctx.Render.SetResponseWriter(gzw)
		}

		ctx.Next()

		// delete content length after we know we have been written to
		gzw.Header().Del("Content-Length")
	}
}

type gzipResponseWriter struct {
	w *gzip.Writer
	ResponseWriter
}

func (grw gzipResponseWriter) Write(p []byte) (int, error) {
	if len(grw.Header().Get(HeaderContentType)) == 0 {
		grw.Header().Set(HeaderContentType, http.DetectContentType(p))
	}
	return grw.w.Write(p)
}

func (grw gzipResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := grw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("the ResponseWriter doesn't support the Hijacker interface")
	}
	return hijacker.Hijack()
}
