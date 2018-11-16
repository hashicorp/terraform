package schema

import (
	"github.com/zclconf/go-cty/cty"
)

type AttributeError struct {
	attributePath cty.Path
	origErr       error
}

func (ae *AttributeError) Error() string {
	return ae.origErr.Error()
}

func (ae *AttributeError) Path() cty.Path {
	return ae.attributePath
}

func NewAttributeError(path cty.Path, origErr error) *AttributeError {
	return &AttributeError{path, origErr}
}
