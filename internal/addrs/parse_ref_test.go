// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"testing"

	"github.com/go-test/deep"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseRefInTestingScope(t *testing.T) {
	tests := []struct {
		Input   string
		Want    *Reference
		WantErr string
	}{
		{
			`output.value`,
			&Reference{
				Subject: OutputValue{
					Name: "value",
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 13, Byte: 12},
				},
			},
			``,
		},
		{
			`output`,
			nil,
			`The "output" object cannot be accessed directly. Instead, access one of its attributes.`,
		},
		{
			`output["foo"]`,
			nil,
			`The "output" object does not support this operation.`,
		},

		{
			`check.health`,
			&Reference{
				Subject: Check{
					Name: "health",
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 13, Byte: 12},
				},
			},
			``,
		},
		{
			`check`,
			nil,
			`The "check" object cannot be accessed directly. Instead, access one of its attributes.`,
		},
		{
			`check["foo"]`,
			nil,
			`The "check" object does not support this operation.`,
		},
		{
			`run.zero`,
			&Reference{
				Subject: Run{
					Name: "zero",
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 9, Byte: 8},
				},
			},
			``,
		},
		{
			`run.zero.value`,
			&Reference{
				Subject: Run{
					Name: "zero",
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 9, Byte: 8},
				},
				Remaining: hcl.Traversal{
					hcl.TraverseAttr{
						Name: "value",
						SrcRange: hcl.Range{
							Start: hcl.Pos{Line: 1, Column: 9, Byte: 8},
							End:   hcl.Pos{Line: 1, Column: 15, Byte: 14},
						},
					},
				},
			},
			``,
		},
		{
			`run`,
			nil,
			`The "run" object cannot be accessed directly. Instead, access one of its attributes.`,
		},
		{
			`run["foo"]`,
			nil,
			`The "run" object does not support this operation.`,
		},

		// Sanity check at least one of the others works to verify it does
		// fall through to the core function.
		{
			`count.index`,
			&Reference{
				Subject: CountAttr{
					Name: "index",
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 12, Byte: 11},
				},
			},
			``,
		},
	}
	for _, test := range tests {
		t.Run(test.Input, func(t *testing.T) {
			traversal, travDiags := hclsyntax.ParseTraversalAbs([]byte(test.Input), "", hcl.Pos{Line: 1, Column: 1})
			if travDiags.HasErrors() {
				t.Fatal(travDiags.Error())
			}

			got, diags := ParseRefFromTestingScope(traversal)

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

func TestParseRef(t *testing.T) {
	tests := []struct {
		Input   string
		Want    *Reference
		WantErr string
	}{

		// count
		{
			`count.index`,
			&Reference{
				Subject: CountAttr{
					Name: "index",
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 12, Byte: 11},
				},
			},
			``,
		},
		{
			`count.index.blah`,
			&Reference{
				Subject: CountAttr{
					Name: "index",
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 12, Byte: 11},
				},
				Remaining: hcl.Traversal{
					hcl.TraverseAttr{
						Name: "blah",
						SrcRange: hcl.Range{
							Start: hcl.Pos{Line: 1, Column: 12, Byte: 11},
							End:   hcl.Pos{Line: 1, Column: 17, Byte: 16},
						},
					},
				},
			},
			``, // valid at this layer, but will fail during eval because "index" is a number
		},
		{
			`count`,
			nil,
			`The "count" object cannot be accessed directly. Instead, access one of its attributes.`,
		},
		{
			`count["hello"]`,
			nil,
			`The "count" object does not support this operation.`,
		},

		// each
		{
			`each.key`,
			&Reference{
				Subject: ForEachAttr{
					Name: "key",
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 9, Byte: 8},
				},
			},
			``,
		},
		{
			`each.value.blah`,
			&Reference{
				Subject: ForEachAttr{
					Name: "value",
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 11, Byte: 10},
				},
				Remaining: hcl.Traversal{
					hcl.TraverseAttr{
						Name: "blah",
						SrcRange: hcl.Range{
							Start: hcl.Pos{Line: 1, Column: 11, Byte: 10},
							End:   hcl.Pos{Line: 1, Column: 16, Byte: 15},
						},
					},
				},
			},
			``,
		},
		{
			`each`,
			nil,
			`The "each" object cannot be accessed directly. Instead, access one of its attributes.`,
		},
		{
			`each["hello"]`,
			nil,
			`The "each" object does not support this operation.`,
		},
		// data
		{
			`data.external.foo`,
			&Reference{
				Subject: Resource{
					Mode: DataResourceMode,
					Type: "external",
					Name: "foo",
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 18, Byte: 17},
				},
			},
			``,
		},
		{
			`data.external.foo.bar`,
			&Reference{
				Subject: ResourceInstance{
					Resource: Resource{
						Mode: DataResourceMode,
						Type: "external",
						Name: "foo",
					},
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 18, Byte: 17},
				},
				Remaining: hcl.Traversal{
					hcl.TraverseAttr{
						Name: "bar",
						SrcRange: hcl.Range{
							Start: hcl.Pos{Line: 1, Column: 18, Byte: 17},
							End:   hcl.Pos{Line: 1, Column: 22, Byte: 21},
						},
					},
				},
			},
			``,
		},
		{
			`data.external.foo["baz"].bar`,
			&Reference{
				Subject: ResourceInstance{
					Resource: Resource{
						Mode: DataResourceMode,
						Type: "external",
						Name: "foo",
					},
					Key: StringKey("baz"),
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 25, Byte: 24},
				},
				Remaining: hcl.Traversal{
					hcl.TraverseAttr{
						Name: "bar",
						SrcRange: hcl.Range{
							Start: hcl.Pos{Line: 1, Column: 25, Byte: 24},
							End:   hcl.Pos{Line: 1, Column: 29, Byte: 28},
						},
					},
				},
			},
			``,
		},
		{
			`data.external.foo["baz"]`,
			&Reference{
				Subject: ResourceInstance{
					Resource: Resource{
						Mode: DataResourceMode,
						Type: "external",
						Name: "foo",
					},
					Key: StringKey("baz"),
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 25, Byte: 24},
				},
			},
			``,
		},
		{
			`data`,
			nil,
			`The "data" object must be followed by two attribute names: the data source type and the resource name.`,
		},
		{
			`data.external`,
			nil,
			`The "data" object must be followed by two attribute names: the data source type and the resource name.`,
		},

		// ephemeral
		{
			`ephemeral.external.foo`,
			&Reference{
				Subject: Resource{
					Mode: EphemeralResourceMode,
					Type: "external",
					Name: "foo",
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 23, Byte: 22},
				},
			},
			``,
		},
		{
			`ephemeral.external.foo.bar`,
			&Reference{
				Subject: ResourceInstance{
					Resource: Resource{
						Mode: EphemeralResourceMode,
						Type: "external",
						Name: "foo",
					},
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 23, Byte: 22},
				},
				Remaining: hcl.Traversal{
					hcl.TraverseAttr{
						Name: "bar",
						SrcRange: hcl.Range{
							Start: hcl.Pos{Line: 1, Column: 23, Byte: 22},
							End:   hcl.Pos{Line: 1, Column: 27, Byte: 26},
						},
					},
				},
			},
			``,
		},
		{
			`ephemeral.external.foo["baz"].bar`,
			&Reference{
				Subject: ResourceInstance{
					Resource: Resource{
						Mode: EphemeralResourceMode,
						Type: "external",
						Name: "foo",
					},
					Key: StringKey("baz"),
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 30, Byte: 29},
				},
				Remaining: hcl.Traversal{
					hcl.TraverseAttr{
						Name: "bar",
						SrcRange: hcl.Range{
							Start: hcl.Pos{Line: 1, Column: 30, Byte: 29},
							End:   hcl.Pos{Line: 1, Column: 34, Byte: 33},
						},
					},
				},
			},
			``,
		},
		{
			`ephemeral.external.foo["baz"]`,
			&Reference{
				Subject: ResourceInstance{
					Resource: Resource{
						Mode: EphemeralResourceMode,
						Type: "external",
						Name: "foo",
					},
					Key: StringKey("baz"),
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 30, Byte: 29},
				},
			},
			``,
		},
		{
			`ephemeral`,
			nil,
			`The "ephemeral" object must be followed by two attribute names: the ephemeral resource type and the resource name.`,
		},
		{
			`ephemeral.external`,
			nil,
			`The "ephemeral" object must be followed by two attribute names: the ephemeral resource type and the resource name.`,
		},

		// local
		{
			`local.foo`,
			&Reference{
				Subject: LocalValue{
					Name: "foo",
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 10, Byte: 9},
				},
			},
			``,
		},
		{
			`local.foo.blah`,
			&Reference{
				Subject: LocalValue{
					Name: "foo",
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 10, Byte: 9},
				},
				Remaining: hcl.Traversal{
					hcl.TraverseAttr{
						Name: "blah",
						SrcRange: hcl.Range{
							Start: hcl.Pos{Line: 1, Column: 10, Byte: 9},
							End:   hcl.Pos{Line: 1, Column: 15, Byte: 14},
						},
					},
				},
			},
			``,
		},
		{
			`local.foo["blah"]`,
			&Reference{
				Subject: LocalValue{
					Name: "foo",
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 10, Byte: 9},
				},
				Remaining: hcl.Traversal{
					hcl.TraverseIndex{
						Key: cty.StringVal("blah"),
						SrcRange: hcl.Range{
							Start: hcl.Pos{Line: 1, Column: 10, Byte: 9},
							End:   hcl.Pos{Line: 1, Column: 18, Byte: 17},
						},
					},
				},
			},
			``,
		},
		{
			`local`,
			nil,
			`The "local" object cannot be accessed directly. Instead, access one of its attributes.`,
		},
		{
			`local["foo"]`,
			nil,
			`The "local" object does not support this operation.`,
		},

		// module
		{
			`module.foo`,
			&Reference{
				Subject: ModuleCall{
					Name: "foo",
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 11, Byte: 10},
				},
			},
			``,
		},
		{
			`module.foo.bar`,
			&Reference{
				Subject: ModuleCallInstanceOutput{
					Call: ModuleCallInstance{
						Call: ModuleCall{
							Name: "foo",
						},
					},
					Name: "bar",
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 15, Byte: 14},
				},
			},
			``,
		},
		{
			`module.foo.bar.baz`,
			&Reference{
				Subject: ModuleCallInstanceOutput{
					Call: ModuleCallInstance{
						Call: ModuleCall{
							Name: "foo",
						},
					},
					Name: "bar",
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 15, Byte: 14},
				},
				Remaining: hcl.Traversal{
					hcl.TraverseAttr{
						Name: "baz",
						SrcRange: hcl.Range{
							Start: hcl.Pos{Line: 1, Column: 15, Byte: 14},
							End:   hcl.Pos{Line: 1, Column: 19, Byte: 18},
						},
					},
				},
			},
			``,
		},
		{
			`module.foo["baz"]`,
			&Reference{
				Subject: ModuleCallInstance{
					Call: ModuleCall{
						Name: "foo",
					},
					Key: StringKey("baz"),
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 18, Byte: 17},
				},
			},
			``,
		},
		{
			`module.foo["baz"].bar`,
			&Reference{
				Subject: ModuleCallInstanceOutput{
					Call: ModuleCallInstance{
						Call: ModuleCall{
							Name: "foo",
						},
						Key: StringKey("baz"),
					},
					Name: "bar",
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 22, Byte: 21},
				},
			},
			``,
		},
		{
			`module.foo["baz"].bar.boop`,
			&Reference{
				Subject: ModuleCallInstanceOutput{
					Call: ModuleCallInstance{
						Call: ModuleCall{
							Name: "foo",
						},
						Key: StringKey("baz"),
					},
					Name: "bar",
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 22, Byte: 21},
				},
				Remaining: hcl.Traversal{
					hcl.TraverseAttr{
						Name: "boop",
						SrcRange: hcl.Range{
							Start: hcl.Pos{Line: 1, Column: 22, Byte: 21},
							End:   hcl.Pos{Line: 1, Column: 27, Byte: 26},
						},
					},
				},
			},
			``,
		},
		{
			`module`,
			nil,
			`The "module" object cannot be accessed directly. Instead, access one of its attributes.`,
		},
		{
			`module["foo"]`,
			nil,
			`The "module" object does not support this operation.`,
		},

		// path
		{
			`path.module`,
			&Reference{
				Subject: PathAttr{
					Name: "module",
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 12, Byte: 11},
				},
			},
			``,
		},
		{
			`path.module.blah`,
			&Reference{
				Subject: PathAttr{
					Name: "module",
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 12, Byte: 11},
				},
				Remaining: hcl.Traversal{
					hcl.TraverseAttr{
						Name: "blah",
						SrcRange: hcl.Range{
							Start: hcl.Pos{Line: 1, Column: 12, Byte: 11},
							End:   hcl.Pos{Line: 1, Column: 17, Byte: 16},
						},
					},
				},
			},
			``, // valid at this layer, but will fail during eval because "module" is a string
		},
		{
			`path`,
			nil,
			`The "path" object cannot be accessed directly. Instead, access one of its attributes.`,
		},
		{
			`path["module"]`,
			nil,
			`The "path" object does not support this operation.`,
		},

		// self
		{
			`self`,
			&Reference{
				Subject: Self,
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 5, Byte: 4},
				},
			},
			``,
		},
		{
			`self.blah`,
			&Reference{
				Subject: Self,
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 5, Byte: 4},
				},
				Remaining: hcl.Traversal{
					hcl.TraverseAttr{
						Name: "blah",
						SrcRange: hcl.Range{
							Start: hcl.Pos{Line: 1, Column: 5, Byte: 4},
							End:   hcl.Pos{Line: 1, Column: 10, Byte: 9},
						},
					},
				},
			},
			``,
		},

		// terraform
		{
			`terraform.workspace`,
			&Reference{
				Subject: TerraformAttr{
					Name: "workspace",
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 20, Byte: 19},
				},
			},
			``,
		},
		{
			`terraform.workspace.blah`,
			&Reference{
				Subject: TerraformAttr{
					Name: "workspace",
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 20, Byte: 19},
				},
				Remaining: hcl.Traversal{
					hcl.TraverseAttr{
						Name: "blah",
						SrcRange: hcl.Range{
							Start: hcl.Pos{Line: 1, Column: 20, Byte: 19},
							End:   hcl.Pos{Line: 1, Column: 25, Byte: 24},
						},
					},
				},
			},
			``, // valid at this layer, but will fail during eval because "workspace" is a string
		},
		{
			`terraform`,
			nil,
			`The "terraform" object cannot be accessed directly. Instead, access one of its attributes.`,
		},
		{
			`terraform["workspace"]`,
			nil,
			`The "terraform" object does not support this operation.`,
		},

		// var
		{
			`var.foo`,
			&Reference{
				Subject: InputVariable{
					Name: "foo",
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 8, Byte: 7},
				},
			},
			``,
		},
		{
			`var.foo.blah`,
			&Reference{
				Subject: InputVariable{
					Name: "foo",
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 8, Byte: 7},
				},
				Remaining: hcl.Traversal{
					hcl.TraverseAttr{
						Name: "blah",
						SrcRange: hcl.Range{
							Start: hcl.Pos{Line: 1, Column: 8, Byte: 7},
							End:   hcl.Pos{Line: 1, Column: 13, Byte: 12},
						},
					},
				},
			},
			``, // valid at this layer, but will fail during eval because "module" is a string
		},
		{
			`var`,
			nil,
			`The "var" object cannot be accessed directly. Instead, access one of its attributes.`,
		},
		{
			`var["foo"]`,
			nil,
			`The "var" object does not support this operation.`,
		},

		// the "resource" prefix forces interpreting the next name as a
		// resource type name. This is an alias for just using a resource
		// type name at the top level, to be used only if a later edition
		// of the Terraform language introduces a new reserved word that
		// overlaps with a resource type name.
		{
			`resource.boop_instance.foo`,
			&Reference{
				Subject: Resource{
					Mode: ManagedResourceMode,
					Type: "boop_instance",
					Name: "foo",
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 27, Byte: 26},
				},
			},
			``,
		},

		// We have some names reserved which might be used by a
		// still-under-discussion proposal for template values or lazy
		// expressions.
		{
			`template.foo`,
			nil,
			`The symbol name "template" is reserved for use in a future Terraform version. If you are using a provider that already uses this as a resource type name, add the prefix "resource." to force interpretation as a resource type name.`,
		},
		{
			`lazy.foo`,
			nil,
			`The symbol name "lazy" is reserved for use in a future Terraform version. If you are using a provider that already uses this as a resource type name, add the prefix "resource." to force interpretation as a resource type name.`,
		},
		{
			`arg.foo`,
			nil,
			`The symbol name "arg" is reserved for use in a future Terraform version. If you are using a provider that already uses this as a resource type name, add the prefix "resource." to force interpretation as a resource type name.`,
		},

		// anything else, interpreted as a managed resource reference
		{
			`boop_instance.foo`,
			&Reference{
				Subject: Resource{
					Mode: ManagedResourceMode,
					Type: "boop_instance",
					Name: "foo",
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 18, Byte: 17},
				},
			},
			``,
		},
		{
			`boop_instance.foo.bar`,
			&Reference{
				Subject: ResourceInstance{
					Resource: Resource{
						Mode: ManagedResourceMode,
						Type: "boop_instance",
						Name: "foo",
					},
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 18, Byte: 17},
				},
				Remaining: hcl.Traversal{
					hcl.TraverseAttr{
						Name: "bar",
						SrcRange: hcl.Range{
							Start: hcl.Pos{Line: 1, Column: 18, Byte: 17},
							End:   hcl.Pos{Line: 1, Column: 22, Byte: 21},
						},
					},
				},
			},
			``,
		},
		{
			`boop_instance.foo["baz"].bar`,
			&Reference{
				Subject: ResourceInstance{
					Resource: Resource{
						Mode: ManagedResourceMode,
						Type: "boop_instance",
						Name: "foo",
					},
					Key: StringKey("baz"),
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 25, Byte: 24},
				},
				Remaining: hcl.Traversal{
					hcl.TraverseAttr{
						Name: "bar",
						SrcRange: hcl.Range{
							Start: hcl.Pos{Line: 1, Column: 25, Byte: 24},
							End:   hcl.Pos{Line: 1, Column: 29, Byte: 28},
						},
					},
				},
			},
			``,
		},
		{
			`boop_instance.foo["baz"]`,
			&Reference{
				Subject: ResourceInstance{
					Resource: Resource{
						Mode: ManagedResourceMode,
						Type: "boop_instance",
						Name: "foo",
					},
					Key: StringKey("baz"),
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 25, Byte: 24},
				},
			},
			``,
		},
		{
			`boop_instance`,
			nil,
			`A reference to a resource type must be followed by at least one attribute access, specifying the resource name.`,
		},

		// Should interpret checks, outputs, and runs as resource types.
		{
			`output.value`,
			&Reference{
				Subject: Resource{
					Mode: ManagedResourceMode,
					Type: "output",
					Name: "value",
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 13, Byte: 12},
				},
			},
			``,
		},
		{
			`check.health`,
			&Reference{
				Subject: Resource{
					Mode: ManagedResourceMode,
					Type: "check",
					Name: "health",
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 13, Byte: 12},
				},
			},
			``,
		},
		{
			`run.zero`,
			&Reference{
				Subject: Resource{
					Mode: ManagedResourceMode,
					Type: "run",
					Name: "zero",
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 9, Byte: 8},
				},
			},
			``,
		},
	}

	for _, test := range tests {
		t.Run(test.Input, func(t *testing.T) {
			traversal, travDiags := hclsyntax.ParseTraversalAbs([]byte(test.Input), "", hcl.Pos{Line: 1, Column: 1})
			if travDiags.HasErrors() {
				t.Fatal(travDiags.Error())
			}

			got, diags := ParseRef(traversal)

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
