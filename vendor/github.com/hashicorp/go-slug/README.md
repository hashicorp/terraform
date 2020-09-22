# go-slug

[![Build Status](https://travis-ci.org/hashicorp/go-slug.svg?branch=master)](https://travis-ci.org/hashicorp/go-slug)
[![GitHub license](https://img.shields.io/github/license/hashicorp/go-slug.svg)](https://github.com/hashicorp/go-slug/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/hashicorp/go-slug?status.svg)](https://godoc.org/github.com/hashicorp/go-slug)
[![Go Report Card](https://goreportcard.com/badge/github.com/hashicorp/go-slug)](https://goreportcard.com/report/github.com/hashicorp/go-slug)
[![GitHub issues](https://img.shields.io/github/issues/hashicorp/go-slug.svg)](https://github.com/hashicorp/go-slug/issues)

Package `go-slug` offers functions for packing and unpacking Terraform Enterprise
compatible slugs. Slugs are gzip compressed tar files containing Terraform configuration files.

## Installation

Installation can be done with a normal `go get`:

```
go get -u github.com/hashicorp/go-slug
```

## Documentation

For the complete usage of `go-slug`, see the full [package docs](https://godoc.org/github.com/hashicorp/go-slug).

## Example

Packing or unpacking a slug is pretty straight forward as shown in the
following example:

```go
package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"

	slug "github.com/hashicorp/go-slug"
)

func main() {
	// First create a buffer for storing the slug.
	buf := bytes.NewBuffer(nil)

	// Then call the Pack function with a directory path containing the
	// configuration files and an io.Writer to write the slug to.
	if _, err := slug.Pack("testdata/archive-dir", buf, false); err != nil {
		log.Fatal(err)
	}

	// Create a directory to unpack the slug contents into.
	dst, err := ioutil.TempDir("", "slug")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dst)

	// Unpacking a slug is done by calling the Unpack function with an
	// io.Reader to read the slug from and a directory path of an existing
	// directory to store the unpacked configuration files.
	if err := slug.Unpack(buf, dst); err != nil {
		log.Fatal(err)
	}
}
```

## Issues and Contributing

If you find an issue with this package, please report an issue. If you'd like,
we welcome any contributions. Fork this repository and submit a pull request.
