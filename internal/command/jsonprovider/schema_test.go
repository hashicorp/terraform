package jsonprovider

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hashicorp/terraform/internal/configs/configschema"
)

func TestMarshalSchemas(t *testing.T) {
	tests := []struct {
		Input    map[string]*configschema.Block
		Versions map[string]uint64
		Want     map[string]*Schema
	}{
		{
			nil,
			map[string]uint64{},
			map[string]*Schema{},
		},
	}

	for _, test := range tests {
		got := marshalSchemas(test.Input, test.Versions)
		if !cmp.Equal(got, test.Want) {
			t.Fatalf("wrong result:\n %v\n", cmp.Diff(got, test.Want))
		}
	}
}

func TestMarshalSchema(t *testing.T) {
	tests := map[string]struct {
		Input *configschema.Block
		Want  *Schema
	}{
		"nil_block": {
			nil,
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
