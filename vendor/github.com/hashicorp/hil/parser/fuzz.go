// +build gofuzz

package parser

import (
	"github.com/hashicorp/hil/ast"
	"github.com/hashicorp/hil/scanner"
)

// This is a fuzz testing function designed to be used with go-fuzz:
//    https://github.com/dvyukov/go-fuzz
//
// It's not included in a normal build due to the gofuzz build tag above.
//
// There are some input files that you can use as a seed corpus for go-fuzz
// in the directory ./fuzz-corpus .

func Fuzz(data []byte) int {
	str := string(data)

	ch := scanner.Scan(str, ast.Pos{Line: 1, Column: 1})
	_, err := Parse(ch)
	if err != nil {
		return 0
	}

	return 1
}
