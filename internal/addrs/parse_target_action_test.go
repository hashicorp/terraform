// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"testing"

	"github.com/go-test/deep"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func TestParseAbsActionTarget(t *testing.T) {
	tcs := []struct {
		Input   string
		Want    Targetable
		WantErr string
	}{
		{
			Input: "action.action_type.action_name",
			Want:  RootModuleInstance.Action("action_type", "action_name"),
		},
		{
			Input: "action.action_type.action_name[0]",
			Want:  RootModuleInstance.ActionInstance("action_type", "action_name", IntKey(0)),
		},
		{
			Input: "module.module_name.action.action_type.action_name",
			Want:  RootModuleInstance.Child("module_name", NoKey).Action("action_type", "action_name"),
		},
		{
			Input: "module.module_name[0].action.action_type.action_name",
			Want:  RootModuleInstance.Child("module_name", IntKey(0)).Action("action_type", "action_name"),
		},
		{
			Input: "module.module_name[0].action.action_type.action_name[0]",
			Want:  RootModuleInstance.Child("module_name", IntKey(0)).ActionInstance("action_type", "action_name", IntKey(0)),
		},
		{
			Input:   "module.module_name",
			WantErr: "Action addresses must contain an action reference after the module reference.",
		},
		{
			Input:   "module.module_name.resource_type.resource_name",
			WantErr: "Action specification must start with `action`.",
		},
	}

	for _, test := range tcs {
		t.Run(test.Input, func(t *testing.T) {
			traversal, travDiags := hclsyntax.ParseTraversalAbs([]byte(test.Input), "", hcl.InitialPos)
			if travDiags.HasErrors() {
				t.Fatal(travDiags.Error())
			}

			got, diags := parseAbsActionTarget(traversal)

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

func TestParseTargetAction(t *testing.T) {
	tcs := []struct {
		Input      string
		Normalized string
		Config     string
		Key        InstanceKey
		WantErr    string
	}{
		{
			Input:      "action.act.a",
			Normalized: "action.act.a[*]",
			Config:     "action.act.a",
			Key:        WildcardKey,
		},
		{
			Input:      "action.act.a[*]",
			Normalized: "action.act.a[*]",
			Config:     "action.act.a",
			Key:        WildcardKey,
		},
		{
			Input:      "module.foo[*].action.act.a",
			Normalized: "module.foo[*].action.act.a[*]",
			Config:     "module.foo.action.act.a",
			Key:        WildcardKey,
		},
		{
			Input:      "module.foo.action.act.a[1]",
			Normalized: "module.foo[*].action.act.a[1]",
			Config:     "module.foo.action.act.a",
			Key:        IntKey(1),
		},
		{
			Input:   "module.foo",
			WantErr: "Action addresses must contain an action reference after the module reference.",
		},
		{
			Input:   "module.foo.test.x",
			WantErr: "Action specification must start with `action`.",
		},
	}

	for _, test := range tcs {
		t.Run(test.Input, func(t *testing.T) {
			traversal, travDiags := hclsyntax.ParseTraversalAbsPattern([]byte(test.Input), "", hcl.InitialPos)
			if travDiags.HasErrors() {
				t.Fatal(travDiags.Error())
			}

			got, diags := ParseTargetAction(traversal)
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

			roundTrip, diags := ParseTargetActionStr(pattern.String())
			if diags.HasErrors() {
				t.Fatalf("failed to parse %q: %s", pattern, diags.Err())
			}
			if !pattern.Equal(roundTrip) || roundTrip.String() != test.Input {
				t.Errorf("round trip produced %s", roundTrip)
			}
		})
	}
}
