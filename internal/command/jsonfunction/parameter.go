// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package jsonfunction

import (
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// parameter represents a parameter to a function.
type parameter struct {
	// Name is an optional name for the argument.
	Name string `json:"name,omitempty"`

	// Description is an optional human-readable description
	// of the argument
	Description string `json:"description,omitempty"`

	// IsNullable is true if null is acceptable value for the argument
	IsNullable bool `json:"is_nullable,omitempty"`

	// A type that any argument for this parameter must conform to.
	Type cty.Type `json:"type"`
}

func marshalParameter(p *function.Parameter) *parameter {
	if p == nil {
		return &parameter{}
	}

	return &parameter{
		Name:        p.Name,
		Description: p.Description,
		IsNullable:  p.AllowNull,
		Type:        p.Type,
	}
}

func marshalParameters(parameters []function.Parameter) []*parameter {
	ret := make([]*parameter, len(parameters))
	for k, p := range parameters {
		ret[k] = marshalParameter(&p)
	}
	return ret
}

func marshalProviderParameter(p providers.FunctionParam) *parameter {
	return &parameter{
		Name:        p.Name,
		Description: p.Description,
		IsNullable:  p.AllowNullValue,
		Type:        p.Type,
	}
}

func marshalProviderParameters(parameters []providers.FunctionParam) []*parameter {
	ret := make([]*parameter, len(parameters))
	for k, p := range parameters {
		ret[k] = marshalProviderParameter(p)
	}
	return ret
}
