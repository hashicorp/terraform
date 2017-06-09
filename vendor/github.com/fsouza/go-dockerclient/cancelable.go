// Copyright 2016 go-dockerclient authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build go1.5

package docker

import "net/http"

func cancelable(client *http.Client, req *http.Request) func() {
	ch := make(chan struct{})
	req.Cancel = ch
	return func() {
		close(ch)
	}
}
