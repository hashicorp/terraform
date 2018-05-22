package funcs

import (
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestBase64Decode(t *testing.T) {
	tests := []struct {
		String cty.Value
		Want   cty.Value
		Err    bool
	}{
		{
			cty.StringVal("YWJjMTIzIT8kKiYoKSctPUB+"),
			cty.StringVal("abc123!?$*&()'-=@~"),
			false,
		},
		{ // Invalid base64 data decoding
			cty.StringVal("this-is-an-invalid-base64-data"),
			cty.UnknownVal(cty.String),
			true,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("base64decode(%#v)", test.String), func(t *testing.T) {
			got, err := Base64Decode(test.String)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
				}
				return
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}
