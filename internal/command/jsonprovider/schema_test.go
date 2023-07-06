// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package jsonprovider

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hashicorp/terraform/internal/providers"
)

func TestMarshalSchemas(t *testing.T) {
	tests := []struct {
		Input map[string]providers.Schema
		Want  map[string]*Schema
	}{
		{
			nil,
			map[string]*Schema{},
		},
	}

	for _, test := range tests {
		got := marshalSchemas(test.Input)
		if !cmp.Equal(got, test.Want) {
			t.Fatalf("wrong result:\n %v\n", cmp.Diff(got, test.Want))
		}
	}
}

func TestMarshalSchema(t *testing.T) {
	tests := map[string]struct {
		Input providers.Schema
		Want  *Schema
	}{
		"nil_block": {
			providers.Schema{},
			&Schema{},
		},
	}

	for _, test := range tests {
		got := marshalSchema(test.Input)
		if !cmp.Equal(got, test.Want) {
			t.Fatalf("wrong result:\n %v\n", cmp.Diff(got, test.Want))
		}
	}
}
