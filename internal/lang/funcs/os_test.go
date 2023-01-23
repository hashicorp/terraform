package funcs

import (
	"fmt"
	"os"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestEnvVar(t *testing.T) {
	tests := []struct {
		EnvVar     cty.Value
		DefaultVal cty.Value
		EnvVars    map[string]string
		Want       cty.Value
		Err        bool
	}{
		{
			cty.StringVal("NON_EXISTING"),
			cty.StringVal("defaultValue"),
			map[string]string{},
			cty.StringVal("defaultValue"),
			false,
		},
		{
			cty.StringVal("TEST"),
			cty.StringVal("defaultValue"),
			map[string]string{
				"TEST": "VALUE",
			},
			cty.StringVal("VALUE"),
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("envvar(%#v, %#v)", test.EnvVar, test.DefaultVal), func(t *testing.T) {
			for k, v := range test.EnvVars {
				os.Setenv(k, v)
			}

			got, err := EnvVar(test.EnvVar, test.DefaultVal)

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
