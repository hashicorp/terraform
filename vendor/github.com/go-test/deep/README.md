# Deep Variable Equality for Humans

[![Go Report Card](https://goreportcard.com/badge/github.com/go-test/deep)](https://goreportcard.com/report/github.com/go-test/deep) [![Build Status](https://travis-ci.org/go-test/deep.svg?branch=master)](https://travis-ci.org/go-test/deep) [![Coverage Status](https://coveralls.io/repos/github/go-test/deep/badge.svg?branch=master)](https://coveralls.io/github/go-test/deep?branch=master) [![GoDoc](https://godoc.org/github.com/go-test/deep?status.svg)](https://godoc.org/github.com/go-test/deep)

This package provides a single function: `deep.Equal`. It's like [reflect.DeepEqual](http://golang.org/pkg/reflect/#DeepEqual) but much friendlier to humans (or any sentient being) for two reason:

* `deep.Equal` returns a list of differences
* `deep.Equal` does not compare unexported fields (by default)

`reflect.DeepEqual` is good (like all things Golang!), but it's a game of [Hunt the Wumpus](https://en.wikipedia.org/wiki/Hunt_the_Wumpus). For large maps, slices, and structs, finding the difference is difficult.

`deep.Equal` doesn't play games with you, it lists the differences:

```go
package main_test

import (
	"testing"
	"github.com/go-test/deep"
)

type T struct {
	Name    string
	Numbers []float64
}

func TestDeepEqual(t *testing.T) {
	// Can you spot the difference?
	t1 := T{
		Name:    "Isabella",
		Numbers: []float64{1.13459, 2.29343, 3.010100010},
	}
	t2 := T{
		Name:    "Isabella",
		Numbers: []float64{1.13459, 2.29843, 3.010100010},
	}

	if diff := deep.Equal(t1, t2); diff != nil {
		t.Error(diff)
	}
}
```


```
$ go test
--- FAIL: TestDeepEqual (0.00s)
        main_test.go:25: [Numbers.slice[1]: 2.29343 != 2.29843]
```

The difference is in `Numbers.slice[1]`: the two values aren't equal using Go `==`.
