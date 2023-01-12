package jsonfunction

import (
	"encoding/json"

	"github.com/zclconf/go-cty/cty/function"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

// parameter represents a parameter to a function.
type parameter struct {
	// Name is an optional name for the argument.
	Name string `json:"name,omitempty"`

	// Description is an optional human-readable description
	// of the argument
	Description string `json:"description,omitempty"`

	// IsNullable is true if null is acceptable value for the argument
	IsNullable bool `json:"is_nullable"`

	// A type that any argument for this parameter must conform to.
	// TODO? could we use cty.Type here instead of calling ctyjson.MarshalType manually?
	// TODO? see: https://github.com/zclconf/go-cty/blob/main/cty/json/type.go
	Type json.RawMessage `json:"type"`
}

func marshalParameter(p *function.Parameter) (*parameter, error) {
	if p == nil {
		return &parameter{}, nil
	}

	t, err := ctyjson.MarshalType(p.Type)
	if err != nil {
		return nil, err
	}

	return &parameter{
		Name:        p.Name,
		Description: p.Description,
		IsNullable:  p.AllowNull,
		Type:        t,
	}, nil
}

func marshalParameters(parameters []function.Parameter) ([]*parameter, error) {
	ret := make([]*parameter, len(parameters))
	for k, p := range parameters {
		mp, err := marshalParameter(&p)
		if err != nil {
			return nil, err
		}
		ret[k] = mp
	}
	return ret, nil
}
