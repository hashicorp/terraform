package funcs

import (
	"fmt"
	"os"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestEnv(t *testing.T) {
	const envKeySet, envValueSet = "ENV_KEY_SET", "ENV_VALUE_SET"
	const envKeyEmpty, envValueEmpty = "ENV_KEY_EMPTY", ""
	const envKeyUnset = "ENV_KEY_UNSET"

	tests := []struct {
		Value cty.Value
		Want  cty.Value
		Err   bool
	}{
		{
			cty.StringVal(envKeySet),
			cty.StringVal(envValueSet),
			false,
		},
		{
			cty.StringVal(envKeyEmpty),
			cty.StringVal(envValueEmpty),
			false,
		},
		{
			cty.StringVal(envKeyUnset),
			cty.StringVal(""),
			true,
		},
	}

	os.Setenv(envKeySet, envValueSet)
	defer os.Unsetenv(envKeySet)
	os.Setenv(envKeyEmpty, envValueEmpty)
	defer os.Unsetenv(envValueEmpty)

	for _, test := range tests {
		t.Run(fmt.Sprintf("Env(%#v...)", test.Value), func(t *testing.T) {
			got, err := Env(test.Value)

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

func TestEnvExists(t *testing.T) {
	const envKeySet = "ENVEXISTS_KEY_SET"
	const envKeyEmpty = "ENVEXISTS_KEY_EMPTY"
	const envKeyUnset = "ENVEXISTS_KEY_UNSET"

	tests := []struct {
		Value cty.Value
		Want  cty.Value
	}{
		{
			cty.StringVal(envKeySet),
			cty.BoolVal(true),
		},
		{
			cty.StringVal(envKeyEmpty),
			cty.BoolVal(true),
		},
		{
			cty.StringVal(envKeyUnset),
			cty.BoolVal(false),
		},
	}

	os.Setenv(envKeySet, envValueSet)
	defer os.Unsetenv(envKeySet)
	os.Setenv(envKeyEmpty, envValueEmpty)
	defer os.Unsetenv(envValueEmpty)

	for _, test := range tests {
		t.Run(fmt.Sprintf("EnvExists(%#v...)", test.Value), func(t *testing.T) {
			got, err := EnvExists(test.Value)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}
