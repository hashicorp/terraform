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
