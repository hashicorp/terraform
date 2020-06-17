# iochan

iochan is a Go library for treating `io` readers and writers like channels.
This is useful when sometimes you wish to use `io.Reader` and such in `select`
statements.

## Installation

Standard `go get`:

```
$ go get github.com/mitchellh/iochan
```
