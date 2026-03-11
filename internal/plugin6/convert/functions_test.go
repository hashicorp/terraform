// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package convert

import (
	"testing"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/zclconf/go-cty/cty"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty-debug/ctydebug"
)

func TestFunctionDeclsToFromProto(t *testing.T) {
	fns := map[string]providers.FunctionDecl{
		"basic": providers.FunctionDecl{
			Parameters: []providers.FunctionParam{
				providers.FunctionParam{
					Name:               "string",
					Type:               cty.String,
					AllowNullValue:     true,
					AllowUnknownValues: true,
					Description:        "must be a string",
					DescriptionKind:    configschema.StringPlain,
				},
			},
			ReturnType:      cty.String,
			Description:     "returns a string",
			DescriptionKind: configschema.StringPlain,
		},
		"variadic": providers.FunctionDecl{
			VariadicParameter: &providers.FunctionParam{
				Name:            "string",
				Type:            cty.String,
				Description:     "must be a string",
				DescriptionKind: configschema.StringMarkdown,
			},
			ReturnType:      cty.String,
			Description:     "returns a string",
			DescriptionKind: configschema.StringMarkdown,
		},
	}

	protoFns, err := FunctionDeclsToProto(fns)
	if err != nil {
		t.Fatal(err)
	}

	gotFns, err := FunctionDeclsFromProto(protoFns)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(fns, gotFns, ctydebug.CmpOptions); diff != "" {
		t.Fatal(diff)
	}
}
