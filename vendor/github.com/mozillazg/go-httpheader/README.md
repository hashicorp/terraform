# go-httpheader

go-httpheader is a Go library for encoding structs into Header fields.

[![Build Status](https://img.shields.io/travis/mozillazg/go-httpheader/master.svg)](https://travis-ci.org/mozillazg/go-httpheader)
[![Coverage Status](https://img.shields.io/coveralls/mozillazg/go-httpheader/master.svg)](https://coveralls.io/r/mozillazg/go-httpheader?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/mozillazg/go-httpheader)](https://goreportcard.com/report/github.com/mozillazg/go-httpheader)
[![GoDoc](https://godoc.org/github.com/mozillazg/go-httpheader?status.svg)](https://godoc.org/github.com/mozillazg/go-httpheader)

## install

`go get -u github.com/mozillazg/go-httpheader`


## usage

```go
package main

import (
	"fmt"
	"net/http"

	"github.com/mozillazg/go-httpheader"
)

type Options struct {
	hide         string
	ContentType  string `header:"Content-Type"`
	Length       int
	XArray       []string `header:"X-Array"`
	TestHide     string   `header:"-"`
	IgnoreEmpty  string   `header:"X-Empty,omitempty"`
	IgnoreEmptyN string   `header:"X-Empty-N,omitempty"`
	CustomHeader http.Header
}

func main() {
	opt := Options{
		hide:         "hide",
		ContentType:  "application/json",
		Length:       2,
		XArray:       []string{"test1", "test2"},
		TestHide:     "hide",
		IgnoreEmptyN: "n",
		CustomHeader: http.Header{
			"X-Test-1": []string{"233"},
			"X-Test-2": []string{"666"},
		},
	}
	h, _ := httpheader.Header(opt)
	fmt.Printf("%#v", h)
	// h:
	// http.Header{
	//	"X-Test-1":     []string{"233"},
	//	"X-Test-2":     []string{"666"},
	//	"Content-Type": []string{"application/json"},
	//	"Length":       []string{"2"},
	//	"X-Array":      []string{"test1", "test2"},
	//	"X-Empty-N":    []string{"n"},
	//}
}
```
