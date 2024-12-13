// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plans

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/zclconf/go-cty/cty"
)

func TestChangeEncodeSensitive(t *testing.T) {
	testVals := []cty.Value{
		cty.ObjectVal(map[string]cty.Value{
			"ding": cty.StringVal("dong").Mark(marks.Sensitive),
		}),
		cty.StringVal("bleep").Mark(marks.Sensitive),
		cty.ListVal([]cty.Value{cty.UnknownVal(cty.String).Mark(marks.Sensitive)}),
	}

	for _, v := range testVals {
		t.Run(fmt.Sprintf("%#v", v), func(t *testing.T) {
			change := Change{
				Before: cty.NullVal(v.Type()),
				After:  v,
			}

			encoded, err := change.Encode(v.Type())
			if err != nil {
				t.Fatal(err)
			}

			decoded, err := encoded.Decode(v.Type())
			if err != nil {
				t.Fatal(err)
			}

			if !v.RawEquals(decoded.After) {
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
