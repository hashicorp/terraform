package projectconfigs

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

func TestLoad(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		got, diags := Load("testdata/empty")
		if diags.HasErrors() {
			t.Fatalf("Unexpected problems: %s", diags.Err().Error())
		}

		want := &Config{
			ProjectRoot: "testdata/empty",
			ConfigFile:  "testdata/empty/.terraform-project.hcl",
			Source:      []byte{},
			Context:     map[string]*ContextValue{},
			Locals:      map[string]*LocalValue{},
			Workspaces:  map[string]*Workspace{},
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("unexpected result\n%s", diff)
		}
	})
	t.Run("context", func(t *testing.T) {
		cfg, diags := Load("testdata/context")
		if diags.HasErrors() {
			t.Fatalf("Unexpected problems: %s", diags.Err().Error())
		}

		got := cfg.Context
		want := map[string]*ContextValue{
			"foo": {
				Name:        "foo",
				Type:        cty.String,
				Description: "The foo thing.",
				Default: &hclsyntax.TemplateExpr{
					Parts: []hclsyntax.Expression{
						&hclsyntax.LiteralValueExpr{
							Val: cty.StringVal("bar"),
							SrcRange: hcl.Range{
								Filename: "testdata/context/.terraform-project.hcl",
								Start:    hcl.Pos{Line: 3, Column: 18, Byte: 56},
								End:      hcl.Pos{Line: 3, Column: 21, Byte: 59},
							},
						},
					},
					SrcRange: hcl.Range{
						Filename: "testdata/context/.terraform-project.hcl",
						Start:    hcl.Pos{Line: 3, Column: 17, Byte: 55},
						End:      hcl.Pos{Line: 3, Column: 22, Byte: 60},
					},
				},
				DeclRange: tfdiags.SourceRange{
					Filename: "testdata/context/.terraform-project.hcl",
					Start:    tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:      tfdiags.SourcePos{Line: 1, Column: 14, Byte: 13},
				},
				NameRange: tfdiags.SourceRange{
					Filename: "testdata/context/.terraform-project.hcl",
					Start:    tfdiags.SourcePos{Line: 1, Column: 9, Byte: 8},
					End:      tfdiags.SourcePos{Line: 1, Column: 14, Byte: 13},
				},
			},
		}
		diff := cmp.Diff(
			want, got,
			cmp.Comparer(cty.Type.Equals),
			cmp.Comparer(cty.Value.RawEquals),
		)
		if diff != "" {
			t.Errorf("unexpected result\n%s", diff)
		}
	})
	t.Run("locals", func(t *testing.T) {
		cfg, diags := Load("testdata/locals")
		if diags.HasErrors() {
			t.Fatalf("Unexpected problems: %s", diags.Err().Error())
		}

		got := cfg.Locals
		want := map[string]*LocalValue{
			"foo": {
				Name: "foo",
				Value: &hclsyntax.LiteralValueExpr{
					Val: cty.NullVal(cty.DynamicPseudoType),
					SrcRange: hcl.Range{
						Filename: "testdata/locals/.terraform-project.hcl",
						Start:    hcl.Pos{Line: 2, Column: 9, Byte: 17},
						End:      hcl.Pos{Line: 2, Column: 13, Byte: 21},
					},
				},
				SrcRange: tfdiags.SourceRange{
					Filename: "testdata/locals/.terraform-project.hcl",
					Start:    tfdiags.SourcePos{Line: 2, Column: 3, Byte: 11},
					End:      tfdiags.SourcePos{Line: 2, Column: 13, Byte: 21},
				},
				NameRange: tfdiags.SourceRange{
					Filename: "testdata/locals/.terraform-project.hcl",
					Start:    tfdiags.SourcePos{Line: 2, Column: 3, Byte: 11},
					End:      tfdiags.SourcePos{Line: 2, Column: 6, Byte: 14},
				},
			},
			"bar": {
				Name: "bar",
				Value: &hclsyntax.LiteralValueExpr{
					Val: cty.NullVal(cty.DynamicPseudoType),
					SrcRange: hcl.Range{
						Filename: "testdata/locals/.terraform-project.hcl",
						Start:    hcl.Pos{Line: 3, Column: 9, Byte: 30},
						End:      hcl.Pos{Line: 3, Column: 13, Byte: 34},
					},
				},
				SrcRange: tfdiags.SourceRange{
					Filename: "testdata/locals/.terraform-project.hcl",
					Start:    tfdiags.SourcePos{Line: 3, Column: 3, Byte: 24},
					End:      tfdiags.SourcePos{Line: 3, Column: 13, Byte: 34},
				},
				NameRange: tfdiags.SourceRange{
					Filename: "testdata/locals/.terraform-project.hcl",
					Start:    tfdiags.SourcePos{Line: 3, Column: 3, Byte: 24},
					End:      tfdiags.SourcePos{Line: 3, Column: 6, Byte: 27},
				},
			},
		}
		diff := cmp.Diff(
			want, got,
			cmp.Comparer(cty.Type.Equals),
			cmp.Comparer(cty.Value.RawEquals),
		)
		if diff != "" {
			t.Errorf("unexpected result\n%s", diff)
		}
	})

}

func TestFindProjectRoot(t *testing.T) {
	tests := []struct {
		StartDir string
		Want     string
		WantErr  string
	}{
		{
			"testdata/subdirs",
			"testdata/subdirs",
			``,
		},
		{
			"testdata/subdirs/",
			"testdata/subdirs",
			``,
		},
		{
			"./testdata/./subdirs",
			"testdata/subdirs",
			``,
		},
		{
			"testdata/subdirs/sub",
			"testdata/subdirs",
			``,
		},
		{
			// NOTE: This test will fail if for some reason the Terraform
			// module directory is cloned beneath some other directory
			// that has a .terraform-project.hcl directory in it. To make
			// the test pass, move your Terraform work tree somewhere else.
			"testdata/nonexist",
			"",
			`start directory "testdata/nonexist" does not exist`,
		},
		{
			"testdata/subdirs/.terraform-project.hcl",
			"",
			`invalid start directory "testdata/subdirs/.terraform-project.hcl": not a directory`,
		},
		{
			"testdata",
			"",
			`no parent directory of testdata contains either a .terraform-project.hcl or a .terraform-project.hcl.json file`,
		},
	}

	for _, test := range tests {
		t.Run(test.StartDir, func(t *testing.T) {
			got, err := FindProjectRoot(test.StartDir)

			if err != nil {
				if test.WantErr == "" {
					t.Fatalf("unexpected error\ngot:  %s\nwant: <nil>", err)
				}
				if got, want := err.Error(), test.WantErr; got != want {
					t.Fatalf("unexpected error\ngot:  %s\nwant: %s", got, want)
				}
				return
			}
			if test.WantErr != "" {
				t.Fatalf("success, but expected error\ngot:  <nil>\nwant: %s", test.WantErr)
			}

			// FindProjectRoot returns an absolute path, but our expectations
			// are relative, so we'll adjust in order to match them.
			want, err := filepath.Abs(test.Want)
			if err != nil {
				t.Fatal(err)
			}

			if got != want {
				t.Fatalf("unexpected result\ngot:  %s\nwant: %s", got, want)
			}
		})
	}
}
