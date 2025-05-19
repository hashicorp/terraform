// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plans

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/zclconf/go-cty/cty"
)

func TestChangeEncodeSensitive(t *testing.T) {
	testVals := []struct {
		val    cty.Value
		schema providers.Schema
	}{
		{
			val: cty.ObjectVal(map[string]cty.Value{
				"ding": cty.StringVal("dong").Mark(marks.Sensitive),
			}),
			schema: providers.Schema{
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"ding": {
							Type:     cty.String,
							Required: true,
						},
					},
				},
			},
		},
		{
			val: cty.ObjectVal(map[string]cty.Value{
				"ding": cty.StringVal("bleep").Mark(marks.Sensitive),
			}),
			schema: providers.Schema{
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"ding": {
							Type:     cty.String,
							Required: true,
						},
					},
				},
			},
		},
		{
			val: cty.ObjectVal(map[string]cty.Value{
				"ding": cty.ListVal([]cty.Value{cty.UnknownVal(cty.String).Mark(marks.Sensitive)}),
			}),
			schema: providers.Schema{
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"ding": {
							Type:     cty.List(cty.String),
							Required: false,
						},
					},
				},
			},
		},
	}

	for _, v := range testVals {
		t.Run(fmt.Sprintf("%#v", v), func(t *testing.T) {
			change := Change{
				Before: cty.NullVal(v.val.Type()),
				After:  v.val,
			}

			encoded, err := change.Encode(&v.schema)
			if err != nil {
				t.Fatal(err)
			}

			decoded, err := encoded.Decode(&v.schema)
			if err != nil {
				t.Fatal(err)
			}

			if !v.val.RawEquals(decoded.After) {
				t.Fatalf("%#v != %#v\n", decoded.After, v)
			}
		})
	}
}

// make sure we get a valid value back even when faced with an error
func TestChangeEncodeError(t *testing.T) {
	changes := &Changes{
		Outputs: []*OutputChange{
			{
				// Missing Addr
				Change: Change{
					Before: cty.NullVal(cty.DynamicPseudoType),
					// can't encode a marked value
					After: cty.StringVal("test").Mark("shoult not be here"),
				},
			},
		},
	}
	// no resources so we can get by with no schemas
	changesSrc, err := changes.Encode(nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if changesSrc == nil {
		t.Fatal("changesSrc should not be nil")
	}
}
