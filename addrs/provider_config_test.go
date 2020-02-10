package addrs

import (
	"fmt"
	"testing"

	"github.com/go-test/deep"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func TestParseAbsProviderConfig(t *testing.T) {
	tests := []struct {
		Input    string
		Want     AbsProviderConfig
		WantDiag string
	}{
		{
			`provider.["registry.terraform.io/hashicorp/aws"]`,
			AbsProviderConfig{
				Module: RootModuleInstance,
				Provider: Provider{
					Type:      "aws",
					Namespace: "hashicorp",
					Hostname:  "registry.terraform.io",
				},
			},
			``,
		},
		{
			`provider.["registry.terraform.io/hashicorp/aws"].foo`,
			AbsProviderConfig{
				Module: RootModuleInstance,
				Provider: Provider{
					Type:      "aws",
					Namespace: "hashicorp",
					Hostname:  "registry.terraform.io",
				},
				Alias: "foo",
			},
			``,
		},
		{
			`module.baz.provider.["registry.terraform.io/hashicorp/aws"]`,
			AbsProviderConfig{
				Module: ModuleInstance{
					{
						Name: "baz",
					},
				},
				Provider: Provider{
					Type:      "aws",
					Namespace: "hashicorp",
					Hostname:  "registry.terraform.io",
				},
			},
			``,
		},
		{
			`module.baz.provider.["registry.terraform.io/hashicorp/aws"].foo`,
			AbsProviderConfig{
				Module: ModuleInstance{
					{
						Name: "baz",
					},
				},
				Provider: Provider{
					Type:      "aws",
					Namespace: "hashicorp",
					Hostname:  "registry.terraform.io",
				},
				Alias: "foo",
			},
			``,
		},
		{
			`module.baz["foo"].provider.["registry.terraform.io/hashicorp/aws"]`,
			AbsProviderConfig{
				Module: ModuleInstance{
					{
						Name:        "baz",
						InstanceKey: StringKey("foo"),
					},
				},
				Provider: Provider{
					Type:      "aws",
					Namespace: "hashicorp",
					Hostname:  "registry.terraform.io",
				},
			},
			``,
		},
		{
			`module.baz[1].provider.["registry.terraform.io/hashicorp/aws"]`,
			AbsProviderConfig{
				Module: ModuleInstance{
					{
						Name:        "baz",
						InstanceKey: IntKey(1),
					},
				},
				Provider: Provider{
					Type:      "aws",
					Namespace: "hashicorp",
					Hostname:  "registry.terraform.io",
				},
			},
			``,
		},
		{
			`module.baz[1].module.bar.provider.["registry.terraform.io/hashicorp/aws"]`,
			AbsProviderConfig{
				Module: ModuleInstance{
					{
						Name:        "baz",
						InstanceKey: IntKey(1),
					},
					{
						Name: "bar",
					},
				},
				Provider: Provider{
					Type:      "aws",
					Namespace: "hashicorp",
					Hostname:  "registry.terraform.io",
				},
			},
			``,
		},
		{
			`aws`,
			AbsProviderConfig{},
			`Provider address must begin with "provider.", followed by a provider type name.`,
		},
		{
			`aws.foo`,
			AbsProviderConfig{},
			`Provider address must begin with "provider.", followed by a provider type name.`,
		},
		{
			`provider`,
			AbsProviderConfig{},
			`Provider address must begin with "provider.", followed by a provider type name.`,
		},
		{
			`provider.aws.foo.bar`,
			AbsProviderConfig{},
			`Extraneous operators after provider configuration alias.`,
		},
		// This used to generate an error, but now the syntax is ok.
		// When NewLegacyProvider() is deprecated this will change.
		{
			`provider["aws"]`,
			AbsProviderConfig{
				Provider: Provider{
					Type:      "aws",
					Namespace: "-",
					Hostname:  "registry.terraform.io",
				},
			},
			``,
		},
		{
			`provider["aws"]["foo"]`,
			AbsProviderConfig{},
			`Provider type name must be followed by a configuration alias name.`,
		},
		{
			`module.foo`,
			AbsProviderConfig{},
			`Provider address must begin with "provider.", followed by a provider type name.`,
		},
	}

	for _, test := range tests {
		t.Run(test.Input, func(t *testing.T) {
			traversal, parseDiags := hclsyntax.ParseTraversalAbs([]byte(test.Input), "", hcl.Pos{})
			if len(parseDiags) != 0 {
				t.Errorf("unexpected diagnostics during parse")
				for _, diag := range parseDiags {
					t.Logf("- %s", diag)
				}
				return
			}

			got, diags := ParseAbsProviderConfig(traversal)

			if test.WantDiag != "" {
				if len(diags) != 1 {
					t.Fatalf("got %d diagnostics; want 1", len(diags))
				}
				gotDetail := diags[0].Description().Detail
				if gotDetail != test.WantDiag {
					t.Fatalf("wrong diagnostic detail\ngot:  %s\nwant: %s", gotDetail, test.WantDiag)
				}
				return
			} else {
				if len(diags) != 0 {
					t.Fatalf("got %d diagnostics; want 0", len(diags))
				}
			}

			for _, problem := range deep.Equal(got, test.Want) {
				fmt.Printf("%#v\n", got)
				t.Error(problem)
			}
		})
	}
}
