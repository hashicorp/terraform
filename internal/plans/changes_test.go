package plans

import (
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestChangeEncodeSensitive(t *testing.T) {
	testVals := []cty.Value{
		cty.ObjectVal(map[string]cty.Value{
			"ding": cty.StringVal("dong").Mark("sensitive"),
		}),
		cty.StringVal("bleep").Mark("bloop"),
		cty.ListVal([]cty.Value{cty.UnknownVal(cty.String).Mark("sup?")}),
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
