package stackconfig

import (
	"github.com/zclconf/go-cty/cty"
)

// InputVariable is a declaration of an input variable within a stack
// configuration. Callers must provide the values for these variables.
type InputVariable struct {
	Name           string
	TypeConstraint cty.Type
}

// LocalValue is a declaration of a private local value within a particular
// stack configuration. These are visible only within the scope of a particular
// [Stack].
type LocalValue struct {
	Name string
}

// OutputValue is a declaration of a result from a stack configuration, which
// can be read by the stack's caller.
type OutputValue struct {
	Name           string
	TypeConstraint cty.Type
}
