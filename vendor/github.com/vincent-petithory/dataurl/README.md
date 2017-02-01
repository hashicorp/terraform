# Data URL Schemes for Go [![wercker status](https://app.wercker.com/status/6f9a2e144dfcc59e862c52459b452928/s "wercker status")](https://app.wercker.com/project/bykey/6f9a2e144dfcc59e862c52459b452928) [![GoDoc](https://godoc.org/github.com/vincent-petithory/dataurl?status.png)](https://godoc.org/github.com/vincent-petithory/dataurl)

This package parses and generates Data URL Schemes for the Go language, according to [RFC 2397](http://tools.ietf.org/html/rfc2397).

Data URLs are small chunks of data commonly used in browsers to display inline data,
typically like small images, or when you use the FileReader API of the browser.

Common use-cases:

 * generate a data URL out of a `string`, `[]byte`, `io.Reader` for inclusion in HTML templates,
 * parse a data URL sent by a browser in a http.Handler, and do something with the data (save to disk, etc.)
 * ...

Install the package with:
~~~
go get github.com/vincent-petithory/dataurl
~~~

## Usage

~~~ go
package main

import (
	"github.com/vincent-petithory/dataurl"
	"fmt"
)

func main() {
	dataURL, err := dataurl.DecodeString(`data:text/plain;charset=utf-8;base64,aGV5YQ==`)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("content type: %s, data: %s\n", dataURL.MediaType.ContentType(), string(dataURL.Data))
	// Output: content type: text/plain, data: heya
}
~~~

From a `http.Handler`:

~~~ go
func handleDataURLUpload(w http.ResponseWriter, r *http.Request) {
	dataURL, err := dataurl.Decode(r.Body)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if dataURL.ContentType() == "image/png" {
		ioutil.WriteFile("image.png", dataURL.Data, 0644)
	} else {
		http.Error(w, "not a png", http.StatusBadRequest)
	}
}
~~~

## Command

For convenience, a `dataurl` command is provided to encode/decode dataurl streams.

~~~
dataurl - Encode or decode dataurl data and print to standard output

Usage: dataurl [OPTION]... [FILE]

  dataurl encodes or decodes FILE or standard input if FILE is - or omitted, and prints to standard output.
  Unless -mimetype is used, when FILE is specified, dataurl will attempt to detect its mimetype using Go's mime.TypeByExtension (http://golang.org/pkg/mime/#TypeByExtension). If this fails or data is read from STDIN, the mimetype will default to application/octet-stream.

Options:
  -a=false: encode data using ascii instead of base64
  -ascii=false: encode data using ascii instead of base64
  -d=false: decode data instead of encoding
  -decode=false: decode data instead of encoding
  -m="": force the mimetype of the data to encode to this value
  -mimetype="": force the mimetype of the data to encode to this value
~~~

## Contributing

Feel free to file an issue/make a pull request if you find any bug, or want to suggest enhancements.
