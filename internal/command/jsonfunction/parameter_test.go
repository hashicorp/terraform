// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package jsonfunction

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

func TestMarshalParameter(t *testing.T) {
	tests := []struct {
		Name  string
		Input *function.Parameter
		Want  *parameter
	}{
		{
			"call with nil",
			nil,
			&parameter{},
		},
		{
			"parameter with description",
			&function.Parameter{
				Name:        "timestamp",
				Description: "`timestamp` returns a UTC timestamp string in [RFC 3339]",
				Type:        cty.String,
			},
			&parameter{
				Name:        "timestamp",
				Description: "`timestamp` returns a UTC timestamp string in [RFC 3339]",
				Type:        cty.String,
			},
		},
		{
			"parameter with additional properties",
			&function.Parameter{
				Name:             "value",
				Type:             cty.DynamicPseudoType,
				AllowUnknown:     true,
				AllowNull:        true,
				AllowMarked:      true,
				AllowDynamicType: true,
			},
			&parameter{
				Name:       "value",
				Type:       cty.DynamicPseudoType,
				IsNullable: true,
			},
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%d-%s", i, test.Name), func(t *testing.T) {
			got := marshalParameter(test.Input)

			if diff := cmp.Diff(test.Want, got, ctydebug.CmpOptions); diff != "" {
				t.Fatalf("mismatch of parameter signature: %s", diff)
			}
		})
	}
}
