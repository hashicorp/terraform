package configs

import (
	"io/ioutil"
	"testing"

	"github.com/go-test/deep"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/addrs"
)

func TestProviderReservedNames(t *testing.T) {
	src, err := ioutil.ReadFile("testdata/invalid-files/provider-reserved.tf")
	if err != nil {
		t.Fatal(err)
	}
	parser := testParser(map[string]string{
		"config.tf": string(src),
	})
	_, diags := parser.LoadConfigFile("config.tf")

	assertExactDiagnostics(t, diags, []string{
		//TODO: This deprecation warning will be removed in terraform v0.15.
		`config.tf:4,13-20: Version constraints inside provider configuration blocks are deprecated; Terraform 0.13 and earlier allowed provider version constraints inside the provider configuration block, but that is now deprecated and will be removed in a future version of Terraform. To silence this warning, move the provider version constraint into the required_providers block.`,
		`config.tf:10,3-8: Reserved argument name in provider block; The provider argument name "count" is reserved for use by Terraform in a future version.`,
		`config.tf:11,3-13: Reserved argument name in provider block; The provider argument name "depends_on" is reserved for use by Terraform in a future version.`,
		`config.tf:12,3-11: Reserved argument name in provider block; The provider argument name "for_each" is reserved for use by Terraform in a future version.`,
		`config.tf:14,3-12: Reserved block type name in provider block; The block type name "lifecycle" is reserved for use by Terraform in a future version.`,
		`config.tf:15,3-9: Reserved block type name in provider block; The block type name "locals" is reserved for use by Terraform in a future version.`,
		`config.tf:13,3-9: Reserved argument name in provider block; The provider argument name "source" is reserved for use by Terraform in a future version.`,
	})
}

func TestParseProviderConfigCompact(t *testing.T) {
	tests := []struct {
		Input    string
		Want     addrs.LocalProviderConfig
		WantDiag string
	}{
		{
			`aws`,
			addrs.LocalProviderConfig{
				LocalName: "aws",
			},
			``,
		},
		{
			`aws.foo`,
			addrs.LocalProviderConfig{
				LocalName: "aws",
				Alias:     "foo",
			},
			``,
		},
		{
			`aws["foo"]`,
			addrs.LocalProviderConfig{},
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

func TestParseProviderConfigCompactStr(t *testing.T) {
	tests := []struct {
		Input    string
		Want     addrs.LocalProviderConfig
		WantDiag string
	}{
		{
			`aws`,
			addrs.LocalProviderConfig{
				LocalName: "aws",
			},
			``,
		},
		{
			`aws.foo`,
			addrs.LocalProviderConfig{
				LocalName: "aws",
				Alias:     "foo",
			},
			``,
		},
		{
			`aws["foo"]`,
			addrs.LocalProviderConfig{},
			`The provider type name must either stand alone or be followed by an alias name separated with a dot.`,
		},
	}

	for _, test := range tests {
		t.Run(test.Input, func(t *testing.T) {
			got, diags := ParseProviderConfigCompactStr(test.Input)

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
