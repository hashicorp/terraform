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

type TestRequestHandleFunc func(w http.ResponseWriter, r *http.Request)

type TestHTTPBackend struct {
	Data   []byte
	Locked bool

	methodFuncs map[string]TestRequestHandleFunc
	methodCalls map[string]int
}

func (h *TestHTTPBackend) Handle(w http.ResponseWriter, r *http.Request) {
	h.countMethodCall(r.Method)
	called := h.callMethod(r.Method, w, r)
	if called {
		return
	}

	switch r.Method {
	case "GET":
		w.Write(h.Data)
	case "PUT":
		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, r.Body); err != nil {
			w.WriteHeader(500)
		}
		w.WriteHeader(201)
		h.Data = buf.Bytes()
	case "POST":
		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, r.Body); err != nil {
			w.WriteHeader(500)
		}
		h.Data = buf.Bytes()
	case "LOCK":
		if h.Locked {
			w.WriteHeader(423)
		} else {
			h.Locked = true
		}
	case "UNLOCK":
		h.Locked = false
	case "DELETE":
		h.Data = nil
		w.WriteHeader(200)
	default:
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(fmt.Sprintf("Unknown method: %s", r.Method)))
	}
}

func (h *TestHTTPBackend) countMethodCall(method string) {
	if h.methodCalls == nil {
		h.methodCalls = make(map[string]int)
	}
	if _, ok := h.methodCalls[method]; !ok {
		h.methodCalls[method] = 0
	}
	h.methodCalls[method]++
}

func (h *TestHTTPBackend) CallCount(method string) int {
	if h.methodCalls == nil {
		return 0
	}
	callCount, ok := h.methodCalls[method]
	if !ok {
		return 0
	}
	return callCount
}

func (h *TestHTTPBackend) callMethod(method string, w http.ResponseWriter, r *http.Request) bool {
	if h.methodFuncs == nil {
		return false
	}
	f, ok := h.methodFuncs[method]
	if ok {
		f(w, r)
	}
	return ok
}

func (h *TestHTTPBackend) SetMethodFunc(method string, impl TestRequestHandleFunc) {
	if h.methodFuncs == nil {
		h.methodFuncs = make(map[string]TestRequestHandleFunc)
	}
	h.methodFuncs[method] = impl
}

// mod_dav-ish behavior
func (h *TestHTTPBackend) HandleWebDAV(w http.ResponseWriter, r *http.Request) {
	h.countMethodCall(r.Method)
	if f, ok := h.methodFuncs[r.Method]; ok {
		f(w, r)
		return
	}

	switch r.Method {
	case "GET":
		w.Write(h.Data)
	case "PUT":
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
		h.Data = nil
		w.WriteHeader(200)
	default:
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(fmt.Sprintf("Unknown method: %s", r.Method)))
	}
}
