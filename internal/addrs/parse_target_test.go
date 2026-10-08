// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"testing"

	"github.com/go-test/deep"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func TestParseAbsTargetable(t *testing.T) {
	foo := RootModuleInstance.Child("foo", NoKey)
	fooBar := foo.Child("bar", NoKey)

	tests := []struct {
		Input   string
		Want    Targetable
		WantErr string
	}{
		{
			`module.foo`,
			foo,
			``,
		},
		{
			`module.foo[2]`,
			RootModuleInstance.Child("foo", IntKey(2)),
			``,
		},
		{
			`module.foo[2].module.bar`,
			RootModuleInstance.Child("foo", IntKey(2)).Child("bar", NoKey),
			``,
		},
		{
			`aws_instance.foo`,
			RootModuleInstance.Resource(ManagedResourceMode, "aws_instance", "foo"),
			``,
		},
		{
			`resource.aws_instance.foo`,
			RootModuleInstance.Resource(ManagedResourceMode, "aws_instance", "foo"),
			``,
		},
		{
			`aws_instance.foo[1]`,
			RootModuleInstance.ResourceInstance(ManagedResourceMode, "aws_instance", "foo", IntKey(1)),
			``,
		},
		{
			`data.aws_instance.foo`,
			RootModuleInstance.Resource(DataResourceMode, "aws_instance", "foo"),
			``,
		},
		{
			`data.aws_instance.foo[1]`,
			RootModuleInstance.ResourceInstance(DataResourceMode, "aws_instance", "foo", IntKey(1)),
			``,
		},
		{
			`ephemeral.aws_instance.foo`,
			RootModuleInstance.Resource(EphemeralResourceMode, "aws_instance", "foo"),
			``,
		},
		{
			`ephemeral.aws_instance.foo[1]`,
			RootModuleInstance.ResourceInstance(EphemeralResourceMode, "aws_instance", "foo", IntKey(1)),
			``,
		},
		{
			`module.foo.aws_instance.bar`,
			foo.Resource(ManagedResourceMode, "aws_instance", "bar"),
			``,
		},
		{
			`module.foo.module.bar.aws_instance.baz`,
			fooBar.Resource(ManagedResourceMode, "aws_instance", "baz"),
			``,
		},
		{
			`module.foo.module.bar.aws_instance.baz["hello"]`,
			fooBar.ResourceInstance(ManagedResourceMode, "aws_instance", "baz", StringKey("hello")),
			``,
		},
		{
			`module.foo.data.aws_instance.bar`,
			foo.Resource(DataResourceMode, "aws_instance", "bar"),
			``,
		},
		{
			`module.foo.module.bar.data.aws_instance.baz`,
			fooBar.Resource(DataResourceMode, "aws_instance", "baz"),
			``,
		},
		{
			`module.foo.module.bar.ephemeral.aws_instance.baz`,
			fooBar.Resource(EphemeralResourceMode, "aws_instance", "baz"),
			``,
		},
		{
			`module.foo.module.bar[0].data.aws_instance.baz`,
			foo.Child("bar", IntKey(0)).Resource(DataResourceMode, "aws_instance", "baz"),
			``,
		},
		{
			`module.foo.module.bar["a"].data.aws_instance.baz["hello"]`,
			foo.Child("bar", StringKey("a")).ResourceInstance(DataResourceMode, "aws_instance", "baz", StringKey("hello")),
			``,
		},
		{
			`module.foo.module.bar.data.aws_instance.baz["hello"]`,
			fooBar.ResourceInstance(DataResourceMode, "aws_instance", "baz", StringKey("hello")),
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

			got, diags := ParseAbsTargetable(traversal)

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

			// A concrete address doesn't accept traversal patterns.
			patternTraversal, _ := hclsyntax.ParseTraversalPartial([]byte(test.Input+"[*]"), "", hcl.Pos{Line: 1, Column: 1})
			if _, diags := ParseAbsTargetable(patternTraversal); !diags.HasErrors() {
				t.Errorf("ParseAbsTargetable accepted traversal pattern %q", test.Input+"[*]")
			}
		})
	}
}
