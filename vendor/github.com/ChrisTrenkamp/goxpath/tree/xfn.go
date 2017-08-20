package tree

import (
	"fmt"
)

//Ctx represents the current context position, size, node, and the current filtered result
type Ctx struct {
	NodeSet
	Pos  int
	Size int
}

//Fn is a XPath function, written in Go
type Fn func(c Ctx, args ...Result) (Result, error)

//LastArgOpt sets whether the last argument in a function is optional, variadic, or neither
type LastArgOpt int

//LastArgOpt options
const (
	None LastArgOpt = iota
	Optional
	Variadic
)

//Wrap interfaces XPath function calls with Go
type Wrap struct {
	Fn Fn
	//NArgs represents the number of arguments to the XPath function.  -1 represents a single optional argument
	NArgs      int
	LastArgOpt LastArgOpt
}

//Call checks the arguments and calls Fn if they are valid
func (w Wrap) Call(c Ctx, args ...Result) (Result, error) {
	switch w.LastArgOpt {
	case Optional:
		if len(args) == w.NArgs || len(args) == w.NArgs-1 {
			return w.Fn(c, args...)
		}
	case Variadic:
		if len(args) >= w.NArgs-1 {
			return w.Fn(c, args...)
		}
	default:
		if len(args) == w.NArgs {
			return w.Fn(c, args...)
		}
	}
	return nil, fmt.Errorf("Invalid number of arguments")
}
