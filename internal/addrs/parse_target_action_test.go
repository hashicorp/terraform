// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"testing"

	"github.com/go-test/deep"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseTargetAction(t *testing.T) {
	tcs := []struct {
		Input   string
		Want    *Target
		WantErr string
	}{
		{
			Input: "action.action_type.action_name",
			Want: &Target{
				Subject: AbsAction{
					Action: Action{
						Type: "action_type",
						Name: "action_name",
					},
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 31, Byte: 30},
				},
			},
		},
		{
			Input: "action.action_type.action_name[0]",
			Want: &Target{
				Subject: AbsActionInstance{
					Action: ActionInstance{
						Action: Action{
							Type: "action_type",
							Name: "action_name",
						},
						Key: IntKey(0),
					},
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 34, Byte: 33},
				},
			},
		},
		{
			Input: "module.module_name.action.action_type.action_name",
			Want: &Target{
				Subject: AbsAction{
					Module: ModuleInstance{
						{
							Name: "module_name",
						},
					},
					Action: Action{
						Type: "action_type",
						Name: "action_name",
					},
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 50, Byte: 49},
				},
			},
		},
		{
			Input: "module.module_name.action.action_type.action_name[0]",
			Want: &Target{
				Subject: AbsActionInstance{
					Module: ModuleInstance{
						{
							Name: "module_name",
						},
					},
					Action: ActionInstance{
						Action: Action{
							Type: "action_type",
							Name: "action_name",
						},
						Key: IntKey(0),
					},
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 53, Byte: 52},
				},
			},
		},
		{
			Input: "module.module_name[0].action.action_type.action_name",
			Want: &Target{
				Subject: AbsAction{
					Module: ModuleInstance{
						{
							Name:        "module_name",
							InstanceKey: IntKey(0),
						},
					},
					Action: Action{
						Type: "action_type",
						Name: "action_name",
					},
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 53, Byte: 52},
				},
			},
		},
		{
			Input: "module.module_name[0].action.action_type.action_name[0]",
			Want: &Target{
				Subject: AbsActionInstance{
					Module: ModuleInstance{
						{
							Name:        "module_name",
							InstanceKey: IntKey(0),
						},
					},
					Action: ActionInstance{
						Action: Action{
							Type: "action_type",
							Name: "action_name",
						},
						Key: IntKey(0),
					},
				},
				SourceRange: tfdiags.SourceRange{
					Start: tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					End:   tfdiags.SourcePos{Line: 1, Column: 56, Byte: 55},
				},
			},
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
			traversal, travDiags := hclsyntax.ParseTraversalAbs([]byte(test.Input), "", hcl.Pos{Line: 1, Column: 1})
			if travDiags.HasErrors() {
				t.Fatal(travDiags.Error())
			}

			got, diags := ParseTargetAction(traversal)

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
