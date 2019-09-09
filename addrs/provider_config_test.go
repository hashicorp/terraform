package addrs

import (
	"testing"

	"github.com/go-test/deep"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func TestParseProviderConfigCompact(t *testing.T) {
	tests := []struct {
		Input    string
		Want     ProviderConfig
		WantDiag string
	}{
		{
			`aws`,
			ProviderConfig{
				Type: "aws",
			},
			``,
		},
		{
			`aws.foo`,
			ProviderConfig{
				Type:  "aws",
				Alias: "foo",
			},
			``,
		},
		{
			`aws["foo"]`,
			ProviderConfig{},
			`The provider type name must either stand alone or be followed by an alias name separated with a dot.`,
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

			got, diags := ParseProviderConfigCompact(traversal)

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
				t.Error(problem)
			}
		})
	}
}
func TestParseAbsProviderConfig(t *testing.T) {
	tests := []struct {
		Input    string
		Want     AbsProviderConfig
		WantDiag string
	}{
		{
			`provider.aws`,
			AbsProviderConfig{
				Module: RootModuleInstance,
				ProviderConfig: ProviderConfig{
					Type: "aws",
				},
			},
			``,
		},
		{
			`provider.aws.foo`,
			AbsProviderConfig{
				Module: RootModuleInstance,
				ProviderConfig: ProviderConfig{
					Type:  "aws",
					Alias: "foo",
				},
			},
			``,
		},
		{
			`module.baz.provider.aws`,
			AbsProviderConfig{
				Module: ModuleInstance{
					{
						Name: "baz",
					},
				},
				ProviderConfig: ProviderConfig{
					Type: "aws",
				},
			},
			``,
		},
		{
			`module.baz.provider.aws.foo`,
			AbsProviderConfig{
				Module: ModuleInstance{
					{
						Name: "baz",
					},
				},
				ProviderConfig: ProviderConfig{
					Type:  "aws",
					Alias: "foo",
				},
			},
			``,
		},
		{
			`module.baz["foo"].provider.aws`,
			AbsProviderConfig{
				Module: ModuleInstance{
					{
						Name:        "baz",
						InstanceKey: StringKey("foo"),
					},
				},
				ProviderConfig: ProviderConfig{
					Type: "aws",
				},
			},
			``,
		},
		{
			`module.baz[1].provider.aws`,
			AbsProviderConfig{
				Module: ModuleInstance{
					{
						Name:        "baz",
						InstanceKey: IntKey(1),
					},
				},
				ProviderConfig: ProviderConfig{
					Type: "aws",
				},
			},
			``,
		},
		{
			`module.baz[1].module.bar.provider.aws`,
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
				ProviderConfig: ProviderConfig{
					Type: "aws",
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
		{
			`provider["aws"]`,
			AbsProviderConfig{},
			`The prefix "provider." must be followed by a provider type name.`,
		},
		{
			`provider.aws["foo"]`,
			AbsProviderConfig{},
			`Provider type name must be followed by a configuration alias name.`,
		},
		{
			`module.foo`,
			AbsProviderConfig{},
			`Provider address must begin with "provider.", followed by a provider type name.`,
		},
		{
			`module.foo["provider"]`,
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
				t.Error(problem)
			}
		})
	}
}
