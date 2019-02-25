package jsonprovider

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/configs/configschema"
)

func TestMarshalAttribute(t *testing.T) {
	tests := []struct {
		Input *configschema.Attribute
		Want  *attribute
	}{
		{
			&configschema.Attribute{Type: cty.String, Optional: true, Computed: true},
			&attribute{
				AttributeType: json.RawMessage(`"string"`),
				Optional:      true,
				Computed:      true,
			},
		},
		{ // collection types look a little odd.
			&configschema.Attribute{Type: cty.Map(cty.String), Optional: true, Computed: true},
			&attribute{
				AttributeType: json.RawMessage(`["map","string"]`),
				Optional:      true,
				Computed:      true,
			},
		},
	}

	for _, test := range tests {
		got := marshalAttribute(test.Input)
		if !cmp.Equal(got, test.Want) {
			t.Fatalf("wrong result:\n %v\n", cmp.Diff(got, test.Want))
		}
	}
}
