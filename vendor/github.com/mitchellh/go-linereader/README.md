# go-linereader

`go-linereader` (Golang package: `linereader`) is a package for Go that
breaks up the input from an io.Reader into multiple lines. It is
a lot like `bufio.Scanner`, except you can specify timeouts that will push
"lines" through after a certain amount of time. This lets you read lines,
but return any data if a line isn't updated for some time.

## Installation and Usage

Install using `go get github.com/mitchellh/go-linereader`.

Full documentation is available at
http://godoc.org/github.com/mitchellh/go-linereader

Below is an example of its usage ignoring errors:

```go
// Assume r is some set io.Reader. Perhaps a file, network, anything.
var r io.Reader

// Initialize the line reader
lr := linereader.New(r)

// Get all the lines
for line := <-lr.Ch {
	// Do something with the line. This line will have the line separator
	// removed.
}
```
