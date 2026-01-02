// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1
package http

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"reflect"
)

type TestHTTPBackend struct {
	Data   []byte
	Locked bool

	GetCalled    int
	PutCalled    int
	PostCalled   int
	LockCalled   int
	UnlockCalled int
	DeleteCalled int
}

func (h *TestHTTPBackend) Handle(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.GetCalled++
		w.Write(h.Data)
	case "PUT":
		h.PutCalled++
		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, r.Body); err != nil {
			w.WriteHeader(500)
		}
		w.WriteHeader(201)
		h.Data = buf.Bytes()
	case "POST":
		h.PostCalled++
		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, r.Body); err != nil {
			w.WriteHeader(500)
		}
		h.Data = buf.Bytes()
	case "LOCK":
		h.LockCalled++
		if h.Locked {
			w.WriteHeader(423)
		} else {
			h.Locked = true
		}
	case "UNLOCK":
		h.UnlockCalled++
		h.Locked = false
	case "DELETE":
		h.DeleteCalled++
		h.Data = nil
		w.WriteHeader(200)
	default:
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("Unknown method: %s", r.Method)))
	}
}

// mod_dav-ish behavior
func (h *TestHTTPBackend) HandleWebDAV(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.GetCalled++
		w.Write(h.Data)
	case "PUT":
		h.PutCalled++
		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, r.Body); err != nil {
			w.WriteHeader(500)
		}
		if reflect.DeepEqual(h.Data, buf.Bytes()) {
			h.Data = buf.Bytes()
			w.WriteHeader(204)
		} else {
			h.Data = buf.Bytes()
			w.WriteHeader(201)
		}
	case "DELETE":
		h.DeleteCalled++
		h.Data = nil
		w.WriteHeader(200)
	default:
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("Unknown method: %s", r.Method)))
	}
}
