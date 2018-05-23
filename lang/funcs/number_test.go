package funcs

import (
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestCeil(t *testing.T) {
	tests := []struct {
		Num  cty.Value
		Want cty.Value
		Err  bool
	}{
		{
			cty.NumberFloatVal(-1.8),
			cty.NumberFloatVal(-1),
			false,
		},
		{
			cty.NumberFloatVal(1.2),
			cty.NumberFloatVal(2),
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("Ceil(%#v)", test.Num), func(t *testing.T) {
			got, err := Ceil(test.Num)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}
