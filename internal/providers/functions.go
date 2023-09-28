// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs/configschema"
)

type FunctionDecl struct {
	Parameters        []FunctionParam
	VariadicParameter *FunctionParam
	ReturnType        cty.Type

	Description     string
	DescriptionKind configschema.StringKind
}

type FunctionParam struct {
	Name string // Only for documentation and UI, because arguments are positional
	Type cty.Type

	Nullable           bool
	AllowUnknownValues bool

	Description     string
	DescriptionKind configschema.StringKind
}
