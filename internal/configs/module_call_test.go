package configs

import (
	"io/ioutil"
	"testing"

	"github.com/go-test/deep"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
)

func TestLoadModuleCall(t *testing.T) {
	src, err := ioutil.ReadFile("testdata/invalid-files/module-calls.tf")
	if err != nil {
		t.Fatal(err)
	}

	parser := testParser(map[string]string{
		"module-calls.tf": string(src),
	})

	file, diags := parser.LoadConfigFile("module-calls.tf")
	assertExactDiagnostics(t, diags, []string{
		`module-calls.tf:20,3-11: Invalid combination of "count" and "for_each"; The "count" and "for_each" meta-arguments are mutually-exclusive, only one should be used to be explicit about the number of resources to be created.`,
	})

	gotModules := file.ModuleCalls
	wantModules := []*ModuleCall{
		{
			Name:          "foo",
			SourceAddr:    addrs.ModuleSourceLocal("./foo"),
			SourceAddrRaw: "./foo",
			SourceSet:     true,
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
			Name: "bar",
			SourceAddr: addrs.ModuleSourceRegistry{
				PackageAddr: addrs.ModuleRegistryPackage{
					Host:         addrs.DefaultModuleRegistryHost,
					Namespace:    "hashicorp",
					Name:         "bar",
					TargetSystem: "aws",
				},
			},
			SourceAddrRaw: "hashicorp/bar/aws",
			SourceSet:     true,
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
			Name: "baz",
			SourceAddr: addrs.ModuleSourceRemote{
				PackageAddr: addrs.ModulePackage("git::https://example.com/"),
			},
			SourceAddrRaw: "git::https://example.com/",
			SourceSet:     true,
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

func TestModuleSourceAddrEntersNewPackage(t *testing.T) {
	tests := []struct {
		Addr string
		Want bool
	}{
		{
			"./",
			false,
		},
		{
			"../bork",
			false,
		},
		{
			"/absolute/path",
			true,
		},
		{
			"github.com/example/foo",
			true,
		},
		{
			"hashicorp/subnets/cidr", // registry module
			true,
		},
		{
			"registry.terraform.io/hashicorp/subnets/cidr", // registry module
			true,
		},
	}

	for _, test := range tests {
		t.Run(test.Addr, func(t *testing.T) {
			addr, err := addrs.ParseModuleSource(test.Addr)
			if err != nil {
				t.Fatalf("parsing failed for %q: %s", test.Addr, err)
			}

			got := moduleSourceAddrEntersNewPackage(addr)
			if got != test.Want {
				t.Errorf("wrong result for %q\ngot:  %#v\nwant:  %#v", addr, got, test.Want)
			}
		})
	}
}
