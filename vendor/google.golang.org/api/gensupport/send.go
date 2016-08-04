// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gensupport

import (
	"net/http"

	"golang.org/x/net/context"
	"golang.org/x/net/context/ctxhttp"
)

// Hook is a function that is called once before each HTTP request that is sent
// by a generated API.  It returns a function that is called after the request
// returns.
// Hook is never called if the context is nil.
var Hook func(ctx context.Context, req *http.Request) func(resp *http.Response) = defaultHook

func defaultHook(ctx context.Context, req *http.Request) func(resp *http.Response) {
	return func(resp *http.Response) {}
}

// SendRequest sends a single HTTP request using the given client.
// If ctx is non-nil, uses ctxhttp.Do, and calls Hook beforehand.  The function
// returned by Hook is called after the request returns.
func SendRequest(ctx context.Context, client *http.Client, req *http.Request) (*http.Response, error) {
	if ctx != nil {
		fn := Hook(ctx, req)
		resp, err := ctxhttp.Do(ctx, client, req)
		fn(resp)
		return resp, err
	}
	return client.Do(req)
}
