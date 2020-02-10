package addrs

import (
	"testing"

	"github.com/go-test/deep"
)

func TestParseProviderSourceStr(t *testing.T) {
	tests := map[string]struct {
		Want Provider
		Err  bool
	}{
		"registry.terraform.io/hashicorp/aws": {
			Provider{
				Type:      "aws",
				Namespace: "hashicorp",
				Hostname:  DefaultRegistryHost,
			},
			false,
		},
		"hashicorp/aws": {
			Provider{
				Type:      "aws",
				Namespace: "hashicorp",
				Hostname:  DefaultRegistryHost,
			},
			false,
		},
		"aws": {
			Provider{
				Type:      "aws",
				Namespace: "-",
				Hostname:  DefaultRegistryHost,
			},
			false,
		},
		"example.com/too/many/parts/here": {
			Provider{},
			true,
		},
		"/too///many//slashes": {
			Provider{},
			true,
		},
		"///": {
			Provider{},
			true,
		},
	}

	for name, test := range tests {
		got, diags := ParseProviderSourceString(name)
		for _, problem := range deep.Equal(got, test.Want) {
			t.Errorf(problem)
		}
		if len(diags) > 0 {
			if test.Err == false {
				t.Errorf("got error, expected success")
			}
		} else {
			if test.Err {
				t.Errorf("got success, expected error")
			}
		}
	}
}
