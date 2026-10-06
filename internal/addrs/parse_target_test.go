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
			patternTraversal, _ := hclsyntax.ParseTraversalAbsPattern([]byte(test.Input+"[*]"), "", hcl.Pos{Line: 1, Column: 1})
			if _, diags := ParseAbsTargetable(patternTraversal); !diags.HasErrors() {
				t.Errorf("ParseAbsTargetable accepted traversal pattern %q", test.Input+"[*]")
			}
		})
	}
}

func TestParseTarget(t *testing.T) {
	tcs := []struct {
		Input string
		// Normalized is the pattern with all wildcards written explicitly
		Normalized string
		Config     string
		Key        InstanceKey
		WantErr    string
	}{
		{
			Input:      "module.foo",
			Normalized: "module.foo[*]",
			Config:     "module.foo",
		},
		{
			Input:      "module.foo[*]",
			Normalized: "module.foo[*]",
			Config:     "module.foo",
		},
		{
			Input:      "module.foo[*].module.bar",
			Normalized: "module.foo[*].module.bar[*]",
			Config:     "module.foo.module.bar",
		},
		{
			Input:      `module.foo["a"].module.bar[*]`,
			Normalized: `module.foo["a"].module.bar[*]`,
			Config:     "module.foo.module.bar",
		},
		{
			Input:      "module.foo.test.x",
			Normalized: "module.foo[*].test.x[*]",
			Config:     "module.foo.test.x",
			Key:        WildcardKey,
		},
		{
			Input:      "module.foo[*].test.x[0]",
			Normalized: "module.foo[*].test.x[0]",
			Config:     "module.foo.test.x",
			Key:        IntKey(0),
		},
		{
			Input:      "test.x[*]",
			Normalized: "test.x[*]",
			Config:     "test.x",
			Key:        WildcardKey,
		},
		{
			Input:      "module.foo[0].data.test.x",
			Normalized: "module.foo[0].data.test.x[*]",
			Config:     "module.foo.data.test.x",
			Key:        WildcardKey,
		},
		{
			Input:      `ephemeral.test.x["a"]`,
			Normalized: `ephemeral.test.x["a"]`,
			Config:     "ephemeral.test.x",
			Key:        StringKey("a"),
		},
		{
			Input:   "module[*].test.x",
			WantErr: `Prefix "module." must be followed by a module name.`,
		},
		{
			Input:   "aws_instance",
			WantErr: "Resource specification must include a resource type and name.",
		},
		{
			Input:   "aws_instance.foo[1].baz",
			WantErr: "Unexpected extra operators after address.",
		},
		{
			Input:   "each.key",
			WantErr: `The keyword "each" is reserved and cannot be used to target a resource address. If you are targeting a resource type that uses a reserved keyword, please prefix your address with "resource.".`,
		},
	}

	for _, test := range tcs {
		t.Run(test.Input, func(t *testing.T) {
			traversal, travDiags := hclsyntax.ParseTraversalAbsPattern([]byte(test.Input), "", hcl.InitialPos)
			if travDiags.HasErrors() {
				t.Fatal(travDiags.Error())
			}

			got, diags := ParseTarget(traversal)
			if test.WantErr != "" {
				if !diags.HasErrors() {
					t.Fatalf("succeeded; want error: %s", test.WantErr)
				}
				if got, want := diags[0].Description().Detail, test.WantErr; got != want {
					t.Fatalf("wrong error\ngot:  %s\nwant: %s", got, want)
				}
				return
			}
			if diags.HasErrors() {
				t.Fatalf("unexpected diagnostics: %s", diags.Err())
			}

			pattern := got
			if got, want := pattern.String(), test.Input; got != want {
				t.Errorf("wrong string\ngot:  %s\nwant: %s", got, want)
			}
			if got, want := pattern.format(true), test.Normalized; got != want {
				t.Errorf("wrong normalized string\ngot:  %s\nwant: %s", got, want)
			}
			if got, want := pattern.ConfigAddr().String(), test.Config; got != want {
				t.Errorf("wrong config address\ngot:  %s\nwant: %s", got, want)
			}
			if got, want := pattern.InstanceKey(), test.Key; got != want {
				t.Errorf("wrong instance key\ngot:  %#v\nwant: %#v", got, want)
			}

			// The string form must parse back to the same pattern, since that
			// is how targets are saved in plan files.
			roundTrip, diags := ParseTargetStr(pattern.String())
			if diags.HasErrors() {
				t.Fatalf("failed to parse %q: %s", pattern, diags.Err())
			}
			if !pattern.Equal(roundTrip) || roundTrip.String() != test.Input {
				t.Errorf("round trip produced %s", roundTrip)
			}
		})
	}
}
