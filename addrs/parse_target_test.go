package addrs

import (
	"testing"

	"github.com/go-test/deep"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/tfdiags"
)

func TestParseTarget(t *testing.T) {
	tests := []struct {
		Input   string
		Want    *Target
		WantErr string
	}{
		{
			`module.foo`,
			&Target{
				Subject: ModuleInstance{
					{
						Name: "foo",
					},
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 11, Byte: 10},
				},
			},
			``,
		},
		{
			`module.foo[2]`,
			&Target{
				Subject: ModuleInstance{
					{
						Name:        "foo",
						InstanceKey: IntKey(2),
					},
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 14, Byte: 13},
				},
			},
			``,
		},
		{
			`module.foo[2].module.bar`,
			&Target{
				Subject: ModuleInstance{
					{
						Name:        "foo",
						InstanceKey: IntKey(2),
					},
					{
						Name: "bar",
					},
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 25, Byte: 24},
				},
			},
			``,
		},
		{
			`aws_instance.foo`,
			&Target{
				Subject: AbsResource{
					Resource: Resource{
						Mode: ManagedResourceMode,
						Type: "aws_instance",
						Name: "foo",
					},
					Module: RootModuleInstance,
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 17, Byte: 16},
				},
			},
			``,
		},
		{
			`aws_instance.foo[1]`,
			&Target{
				Subject: AbsResourceInstance{
					Resource: ResourceInstance{
						Resource: Resource{
							Mode: ManagedResourceMode,
							Type: "aws_instance",
							Name: "foo",
						},
						Key: IntKey(1),
					},
					Module: RootModuleInstance,
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 20, Byte: 19},
				},
			},
			``,
		},
		{
			`data.aws_instance.foo`,
			&Target{
				Subject: AbsResource{
					Resource: Resource{
						Mode: DataResourceMode,
						Type: "aws_instance",
						Name: "foo",
					},
					Module: RootModuleInstance,
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 22, Byte: 21},
				},
			},
			``,
		},
		{
			`data.aws_instance.foo[1]`,
			&Target{
				Subject: AbsResourceInstance{
					Resource: ResourceInstance{
						Resource: Resource{
							Mode: DataResourceMode,
							Type: "aws_instance",
							Name: "foo",
						},
						Key: IntKey(1),
					},
					Module: RootModuleInstance,
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 25, Byte: 24},
				},
			},
			``,
		},
		{
			`module.foo.aws_instance.bar`,
			&Target{
				Subject: AbsResource{
					Resource: Resource{
						Mode: ManagedResourceMode,
						Type: "aws_instance",
						Name: "bar",
					},
					Module: ModuleInstance{
						{Name: "foo"},
					},
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 28, Byte: 27},
				},
			},
			``,
		},
		{
			`module.foo.module.bar.aws_instance.baz`,
			&Target{
				Subject: AbsResource{
					Resource: Resource{
						Mode: ManagedResourceMode,
						Type: "aws_instance",
						Name: "baz",
					},
					Module: ModuleInstance{
						{Name: "foo"},
						{Name: "bar"},
					},
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 39, Byte: 38},
				},
			},
			``,
		},
		{
			`module.foo.module.bar.aws_instance.baz["hello"]`,
			&Target{
				Subject: AbsResourceInstance{
					Resource: ResourceInstance{
						Resource: Resource{
							Mode: ManagedResourceMode,
							Type: "aws_instance",
							Name: "baz",
						},
						Key: StringKey("hello"),
					},
					Module: ModuleInstance{
						{Name: "foo"},
						{Name: "bar"},
					},
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 48, Byte: 47},
				},
			},
			``,
		},
		{
			`module.foo.data.aws_instance.bar`,
			&Target{
				Subject: AbsResource{
					Resource: Resource{
						Mode: DataResourceMode,
						Type: "aws_instance",
						Name: "bar",
					},
					Module: ModuleInstance{
						{Name: "foo"},
					},
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 33, Byte: 32},
				},
			},
			``,
		},
		{
			`module.foo.module.bar.data.aws_instance.baz`,
			&Target{
				Subject: AbsResource{
					Resource: Resource{
						Mode: DataResourceMode,
						Type: "aws_instance",
						Name: "baz",
					},
					Module: ModuleInstance{
						{Name: "foo"},
						{Name: "bar"},
					},
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 44, Byte: 43},
				},
			},
			``,
		},
		{
			`module.foo.module.bar.data.aws_instance.baz["hello"]`,
			&Target{
				Subject: AbsResourceInstance{
					Resource: ResourceInstance{
						Resource: Resource{
							Mode: DataResourceMode,
							Type: "aws_instance",
							Name: "baz",
						},
						Key: StringKey("hello"),
					},
					Module: ModuleInstance{
						{Name: "foo"},
						{Name: "bar"},
					},
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 53, Byte: 52},
				},
			},
			``,
		},

		{
			`aws_instance`,
			nil,
			`Resource specification must include a resource type and name.`,
		},
		{
			`module`,
			nil,
			`Prefix "module." must be followed by a module name.`,
		},
		{
			`module["baz"]`,
			nil,
			`Prefix "module." must be followed by a module name.`,
		},
		{
			`module.baz.bar`,
			nil,
			`Resource specification must include a resource type and name.`,
		},
		{
			`aws_instance.foo.bar`,
			nil,
			`Resource instance key must be given in square brackets.`,
		},
		{
			`aws_instance.foo[1].baz`,
			nil,
			`Unexpected extra operators after address.`,
		},
	}

	for _, test := range tests {
		t.Run(test.Input, func(t *testing.T) {
			traversal, travDiags := hclsyntax.ParseTraversalAbs([]byte(test.Input), "", hcl.Pos{Line: 1, Column: 1})
			if travDiags.HasErrors() {
				t.Fatal(travDiags.Error())
			}

			got, diags := ParseTarget(traversal)

			switch len(diags) {
			case 0:
				if test.WantErr != "" {
					t.Fatalf("succeeded; want error: %s", test.WantErr)
				}
			case 1:
				if test.WantErr == "" {
					t.Fatalf("unexpected diagnostics: %s", diags.Err())
				}
				if got, want := diags[0].Description().Detail, test.WantErr; got != want {
					t.Fatalf("wrong error\ngot:  %s\nwant: %s", got, want)
				}
			default:
				t.Fatalf("too many diagnostics: %s", diags.Err())
			}

			if diags.HasErrors() {
				return
			}

			for _, problem := range deep.Equal(got, test.Want) {
				t.Errorf(problem)
			}
		})
	}
}
