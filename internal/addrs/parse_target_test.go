// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"testing"

	"github.com/go-test/deep"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/tfdiags"
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
			`resource.aws_instance.foo`,
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
					End:   tfdiags.SourcePos{Line: 1, Column: 26, Byte: 25},
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
			`ephemeral.aws_instance.foo`,
			&Target{
				Subject: AbsResource{
					Resource: Resource{
						Mode: EphemeralResourceMode,
						Type: "aws_instance",
						Name: "foo",
					},
					Module: RootModuleInstance,
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 27, Byte: 26},
				},
			},
			``,
		},
		{
			`ephemeral.aws_instance.foo[1]`,
			&Target{
				Subject: AbsResourceInstance{
					Resource: ResourceInstance{
						Resource: Resource{
							Mode: EphemeralResourceMode,
							Type: "aws_instance",
							Name: "foo",
						},
						Key: IntKey(1),
					},
					Module: RootModuleInstance,
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 30, Byte: 29},
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
			`module.foo.module.bar.ephemeral.aws_instance.baz`,
			&Target{
				Subject: AbsResource{
					Resource: Resource{
						Mode: EphemeralResourceMode,
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
					End:   tfdiags.SourcePos{Line: 1, Column: 49, Byte: 48},
				},
			},
			``,
		},
		{
			`module.foo.module.bar[0].data.aws_instance.baz`,
			&Target{
				Subject: AbsResource{
					Resource: Resource{
						Mode: DataResourceMode,
						Type: "aws_instance",
						Name: "baz",
					},
					Module: ModuleInstance{
						{Name: "foo", InstanceKey: NoKey},
						{Name: "bar", InstanceKey: IntKey(0)},
					},
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 47, Byte: 46},
				},
			},
			``,
		},
		{
			`module.foo.module.bar["a"].data.aws_instance.baz["hello"]`,
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
						{Name: "foo", InstanceKey: NoKey},
						{Name: "bar", InstanceKey: StringKey("a")},
					},
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 58, Byte: 57},
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
		{
			`each.key`,
			nil,
			`The keyword "each" is reserved and cannot be used to target a resource address. If you are targeting a resource type that uses a reserved keyword, please prefix your address with "resource.".`,
		},
		{
			`module.foo[1].each`,
			nil,
			`The keyword "each" is reserved and cannot be used to target a resource address. If you are targeting a resource type that uses a reserved keyword, please prefix your address with "resource.".`,
		},
		{
			`count.index`,
			nil,
			`The keyword "count" is reserved and cannot be used to target a resource address. If you are targeting a resource type that uses a reserved keyword, please prefix your address with "resource.".`,
		},
		{
			`local.value`,
			nil,
			`The keyword "local" is reserved and cannot be used to target a resource address. If you are targeting a resource type that uses a reserved keyword, please prefix your address with "resource.".`,
		},
		{
			`path.root`,
			nil,
			`The keyword "path" is reserved and cannot be used to target a resource address. If you are targeting a resource type that uses a reserved keyword, please prefix your address with "resource.".`,
		},
		{
			`self.id`,
			nil,
			`The keyword "self" is reserved and cannot be used to target a resource address. If you are targeting a resource type that uses a reserved keyword, please prefix your address with "resource.".`,
		},
		{
			`terraform.planning`,
			nil,
			`The keyword "terraform" is reserved and cannot be used to target a resource address. If you are targeting a resource type that uses a reserved keyword, please prefix your address with "resource.".`,
		},
		{
			`var.foo`,
			nil,
			`The keyword "var" is reserved and cannot be used to target a resource address. If you are targeting a resource type that uses a reserved keyword, please prefix your address with "resource.".`,
		},
		{
			`template`,
			nil,
			`The keyword "template" is reserved and cannot be used to target a resource address. If you are targeting a resource type that uses a reserved keyword, please prefix your address with "resource.".`,
		},
		{
			`lazy`,
			nil,
			`The keyword "lazy" is reserved and cannot be used to target a resource address. If you are targeting a resource type that uses a reserved keyword, please prefix your address with "resource.".`,
		},
		{
			`arg`,
			nil,
			`The keyword "arg" is reserved and cannot be used to target a resource address. If you are targeting a resource type that uses a reserved keyword, please prefix your address with "resource.".`,
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
				t.Error(problem)
			}
		})
	}
}
