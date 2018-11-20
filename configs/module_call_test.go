package configs

import (
	"io/ioutil"
	"testing"

	"github.com/go-test/deep"
	"github.com/hashicorp/hcl2/hcl"
)

func TestLoadModuleCall(t *testing.T) {
	src, err := ioutil.ReadFile("test-fixtures/invalid-files/module-calls.tf")
	if err != nil {
		t.Fatal(err)
	}

	parser := testParser(map[string]string{
		"module-calls.tf": string(src),
	})

	file, diags := parser.LoadConfigFile("module-calls.tf")
	assertExactDiagnostics(t, diags, []string{
		`module-calls.tf:19,3-8: Reserved argument name in module block; The name "count" is reserved for use in a future version of Terraform.`,
		`module-calls.tf:20,3-11: Reserved argument name in module block; The name "for_each" is reserved for use in a future version of Terraform.`,
		`module-calls.tf:22,3-13: Reserved argument name in module block; The name "depends_on" is reserved for use in a future version of Terraform.`,
	})

	gotModules := file.ModuleCalls
	wantModules := []*ModuleCall{
		{
			Name:       "foo",
			SourceAddr: "./foo",
			SourceSet:  true,
			SourceAddrRange: hcl.Range{
				Filename: "module-calls.tf",
				Start:    hcl.Pos{Line: 3, Column: 12, Byte: 27},
				End:      hcl.Pos{Line: 3, Column: 19, Byte: 34},
			},
			DeclRange: hcl.Range{
				Filename: "module-calls.tf",
				Start:    hcl.Pos{Line: 2, Column: 1, Byte: 1},
				End:      hcl.Pos{Line: 2, Column: 13, Byte: 13},
			},
		},
		{
			Name:       "bar",
			SourceAddr: "hashicorp/bar/aws",
			SourceSet:  true,
			SourceAddrRange: hcl.Range{
				Filename: "module-calls.tf",
				Start:    hcl.Pos{Line: 8, Column: 12, Byte: 113},
				End:      hcl.Pos{Line: 8, Column: 31, Byte: 132},
			},
			DeclRange: hcl.Range{
				Filename: "module-calls.tf",
				Start:    hcl.Pos{Line: 7, Column: 1, Byte: 87},
				End:      hcl.Pos{Line: 7, Column: 13, Byte: 99},
			},
		},
		{
			Name:       "baz",
			SourceAddr: "git::https://example.com/",
			SourceSet:  true,
			SourceAddrRange: hcl.Range{
				Filename: "module-calls.tf",
				Start:    hcl.Pos{Line: 15, Column: 12, Byte: 193},
				End:      hcl.Pos{Line: 15, Column: 39, Byte: 220},
			},
			DependsOn: []hcl.Traversal{
				{
					hcl.TraverseRoot{
						Name: "module",
						SrcRange: hcl.Range{
							Filename: "module-calls.tf",
							Start:    hcl.Pos{Line: 23, Column: 5, Byte: 295},
							End:      hcl.Pos{Line: 23, Column: 11, Byte: 301},
						},
					},
					hcl.TraverseAttr{
						Name: "bar",
						SrcRange: hcl.Range{
							Filename: "module-calls.tf",
							Start:    hcl.Pos{Line: 23, Column: 11, Byte: 301},
							End:      hcl.Pos{Line: 23, Column: 15, Byte: 305},
						},
					},
				},
			},
			Providers: []PassedProviderConfig{
				{
					InChild: &ProviderConfigRef{
						Name: "aws",
						NameRange: hcl.Range{
							Filename: "module-calls.tf",
							Start:    hcl.Pos{Line: 27, Column: 5, Byte: 332},
							End:      hcl.Pos{Line: 27, Column: 8, Byte: 335},
						},
					},
					InParent: &ProviderConfigRef{
						Name: "aws",
						NameRange: hcl.Range{
							Filename: "module-calls.tf",
							Start:    hcl.Pos{Line: 27, Column: 11, Byte: 338},
							End:      hcl.Pos{Line: 27, Column: 14, Byte: 341},
						},
						Alias: "foo",
						AliasRange: &hcl.Range{
							Filename: "module-calls.tf",
							Start:    hcl.Pos{Line: 27, Column: 14, Byte: 341},
							End:      hcl.Pos{Line: 27, Column: 18, Byte: 345},
						},
					},
				},
			},
			DeclRange: hcl.Range{
				Filename: "module-calls.tf",
				Start:    hcl.Pos{Line: 14, Column: 1, Byte: 167},
				End:      hcl.Pos{Line: 14, Column: 13, Byte: 179},
			},
		},
	}

	// We'll hide all of the bodies/exprs since we're treating them as opaque
	// here anyway... the point of this test is to ensure we handle everything
	// else properly.
	for _, m := range gotModules {
		m.Config = nil
		m.Count = nil
		m.ForEach = nil
	}

	for _, problem := range deep.Equal(gotModules, wantModules) {
		t.Error(problem)
	}
}
