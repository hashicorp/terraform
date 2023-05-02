// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package addrs

import (
	"reflect"
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
			`provider["registry.terraform.io/hashicorp/aws"]`,
			AbsProviderConfig{
				Module: RootModule,
				Provider: Provider{
					Type:      "aws",
					Namespace: "hashicorp",
					Hostname:  "registry.terraform.io",
				},
			},
			``,
		},
		{
			`provider["registry.terraform.io/hashicorp/aws"].foo`,
			AbsProviderConfig{
				Module: RootModule,
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
			`module.baz.provider["registry.terraform.io/hashicorp/aws"]`,
			AbsProviderConfig{
				Module: Module{"baz"},
				Provider: Provider{
					Type:      "aws",
					Namespace: "hashicorp",
					Hostname:  "registry.terraform.io",
				},
			},
			``,
		},
		{
			`module.baz.provider["registry.terraform.io/hashicorp/aws"].foo`,
			AbsProviderConfig{
				Module: Module{"baz"},
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
			`module.baz["foo"].provider["registry.terraform.io/hashicorp/aws"]`,
			AbsProviderConfig{},
			`Provider address cannot contain module indexes`,
		},
		{
			`module.baz[1].provider["registry.terraform.io/hashicorp/aws"]`,
			AbsProviderConfig{},
			`Provider address cannot contain module indexes`,
		},
		{
			`module.baz[1].module.bar.provider["registry.terraform.io/hashicorp/aws"]`,
			AbsProviderConfig{},
			`Provider address cannot contain module indexes`,
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
			`provider["aws"]["foo"]`,
			AbsProviderConfig{},
			`Provider type name must be followed by a configuration alias name.`,
		},
		{
			`module.foo`,
			AbsProviderConfig{},
			`Provider address must begin with "provider.", followed by a provider type name.`,
		},
		{
			`provider[0]`,
			AbsProviderConfig{},
			`The prefix "provider." must be followed by a provider type name.`,
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

func TestAbsProviderConfigString(t *testing.T) {
	tests := []struct {
		Config AbsProviderConfig
		Want   string
	}{
		{
			AbsProviderConfig{
				Module:   RootModule,
				Provider: NewLegacyProvider("foo"),
			},
			`provider["registry.terraform.io/-/foo"]`,
		},
		{
			AbsProviderConfig{
				Module:   RootModule.Child("child_module"),
				Provider: NewDefaultProvider("foo"),
			},
			`module.child_module.provider["registry.terraform.io/hashicorp/foo"]`,
		},
		{
			AbsProviderConfig{
				Module:   RootModule,
				Alias:    "bar",
				Provider: NewDefaultProvider("foo"),
			},
			`provider["registry.terraform.io/hashicorp/foo"].bar`,
		},
		{
			AbsProviderConfig{
				Module:   RootModule.Child("child_module"),
				Alias:    "bar",
				Provider: NewDefaultProvider("foo"),
			},
			`module.child_module.provider["registry.terraform.io/hashicorp/foo"].bar`,
		},
	}

	for _, test := range tests {
		got := test.Config.String()
		if got != test.Want {
			t.Errorf("wrong result. Got %s, want %s\n", got, test.Want)
		}
	}
}

func TestAbsProviderConfigLegacyString(t *testing.T) {
	tests := []struct {
		Config AbsProviderConfig
		Want   string
	}{
		{
			AbsProviderConfig{
				Module:   RootModule,
				Provider: NewLegacyProvider("foo"),
			},
			`provider.foo`,
		},
		{
			AbsProviderConfig{
				Module:   RootModule.Child("child_module"),
				Provider: NewLegacyProvider("foo"),
			},
			`module.child_module.provider.foo`,
		},
		{
			AbsProviderConfig{
				Module:   RootModule,
				Alias:    "bar",
				Provider: NewLegacyProvider("foo"),
			},
			`provider.foo.bar`,
		},
		{
			AbsProviderConfig{
				Module:   RootModule.Child("child_module"),
				Alias:    "bar",
				Provider: NewLegacyProvider("foo"),
			},
			`module.child_module.provider.foo.bar`,
		},
	}

	for _, test := range tests {
		got := test.Config.LegacyString()
		if got != test.Want {
			t.Errorf("wrong result. Got %s, want %s\n", got, test.Want)
		}
	}
}

func TestParseLegacyAbsProviderConfigStr(t *testing.T) {
	tests := []struct {
		Config string
		Want   AbsProviderConfig
	}{
		{
			`provider.foo`,
			AbsProviderConfig{
				Module:   RootModule,
				Provider: NewLegacyProvider("foo"),
			},
		},
		{
			`module.child_module.provider.foo`,
			AbsProviderConfig{
				Module:   RootModule.Child("child_module"),
				Provider: NewLegacyProvider("foo"),
			},
		},
		{
			`provider.terraform`,
			AbsProviderConfig{
				Module:   RootModule,
				Provider: NewBuiltInProvider("terraform"),
			},
		},
	}

	for _, test := range tests {
		got, _ := ParseLegacyAbsProviderConfigStr(test.Config)
		if !reflect.DeepEqual(got, test.Want) {
			t.Errorf("wrong result. Got %s, want %s\n", got, test.Want)
		}
	}
}
