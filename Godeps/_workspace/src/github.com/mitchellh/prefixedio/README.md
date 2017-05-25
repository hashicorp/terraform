# prefixedio

`prefixedio` (Golang package: `prefixedio`) is a package for Go
that takes an `io.Reader` and de-multiplexes line-oriented data based
on a line prefix to a set of readers.

## Installation and Usage

Install using `go get github.com/mitchellh/prefixedio`.

Full documentation is available at
http://godoc.org/github.com/mitchellh/prefixedio

Below is an example of its usage ignoring errors:

```go
// Assume r is some set io.Reader. Perhaps a file, network, anything.
var r io.Reader

// Initialize the prefixed reader
pr, _ := prefixedio.NewReader(r)

// Grab readers for a couple prefixes
errR, _ := pr.Prefix("err: ")
outR, _ := pr.Prefix("out: ")

// Copy the data to different places based on the prefix
go io.Copy(os.Stderr, errR)
go io.Copy(os.Stdout, outR)
```
