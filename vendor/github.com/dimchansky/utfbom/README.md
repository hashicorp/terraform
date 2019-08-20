# utfbom [![Godoc](https://godoc.org/github.com/dimchansky/utfbom?status.png)](https://godoc.org/github.com/dimchansky/utfbom) [![License](https://img.shields.io/:license-apache-blue.svg)](https://opensource.org/licenses/Apache-2.0) [![Build Status](https://travis-ci.org/dimchansky/utfbom.svg?branch=master)](https://travis-ci.org/dimchansky/utfbom) [![Go Report Card](https://goreportcard.com/badge/github.com/dimchansky/utfbom)](https://goreportcard.com/report/github.com/dimchansky/utfbom) [![Coverage Status](https://coveralls.io/repos/github/dimchansky/utfbom/badge.svg?branch=master)](https://coveralls.io/github/dimchansky/utfbom?branch=master)

The package utfbom implements the detection of the BOM (Unicode Byte Order Mark) and removing as necessary. It can also return the encoding detected by the BOM.

## Installation

    go get -u github.com/dimchansky/utfbom
    
## Example

```go
package main

import (
	"bytes"
	"fmt"
	"io/ioutil"

	"github.com/dimchansky/utfbom"
)

func main() {
	trySkip([]byte("\xEF\xBB\xBFhello"))
	trySkip([]byte("hello"))
}

func trySkip(byteData []byte) {
	fmt.Println("Input:", byteData)

	// just skip BOM
	output, err := ioutil.ReadAll(utfbom.SkipOnly(bytes.NewReader(byteData)))
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("ReadAll with BOM skipping", output)

	// skip BOM and detect encoding
	sr, enc := utfbom.Skip(bytes.NewReader(byteData))
	var encStr string
	switch enc {
	case utfbom.UTF8:
		encStr = "UTF8"
	case utfbom.UTF16BigEndian:
		encStr = "UTF16 big endian"
	case utfbom.UTF16LittleEndian:
		encStr = "UTF16 little endian"
	case utfbom.UTF32BigEndian:
		encStr = "UTF32 big endian"
	case utfbom.UTF32LittleEndian:
		encStr = "UTF32 little endian"
	default:
		encStr = "Unknown, no byte-order mark found"
	}
	fmt.Println("Detected encoding:", encStr)
	output, err = ioutil.ReadAll(sr)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("ReadAll with BOM detection and skipping", output)
	fmt.Println()
}
```

Output:

```
$ go run main.go
Input: [239 187 191 104 101 108 108 111]
ReadAll with BOM skipping [104 101 108 108 111]
Detected encoding: UTF8
ReadAll with BOM detection and skipping [104 101 108 108 111]

Input: [104 101 108 108 111]
ReadAll with BOM skipping [104 101 108 108 111]
Detected encoding: Unknown, no byte-order mark found
ReadAll with BOM detection and skipping [104 101 108 108 111]
```


