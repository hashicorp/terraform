// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package jsonchecks

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/states"
)

func TestMarshalCheckStates(t *testing.T) {
	resourceAAddr := addrs.ConfigCheckable(addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test",
		Name: "a",
	}.InModule(addrs.RootModule))
	resourceAInstAddr := addrs.Checkable(addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test",
		Name: "a",
	}.Instance(addrs.StringKey("foo")).Absolute(addrs.RootModuleInstance))
	moduleChildAddr := addrs.RootModuleInstance.Child("child", addrs.IntKey(0))
	resourceBAddr := addrs.ConfigCheckable(addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test",
		Name: "b",
	}.InModule(moduleChildAddr.Module()))
	resourceBInstAddr := addrs.Checkable(addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test",
		Name: "b",
	}.Instance(addrs.NoKey).Absolute(moduleChildAddr))
	outputAAddr := addrs.ConfigCheckable(addrs.OutputValue{Name: "a"}.InModule(addrs.RootModule))
	outputAInstAddr := addrs.Checkable(addrs.OutputValue{Name: "a"}.Absolute(addrs.RootModuleInstance))
	outputBAddr := addrs.ConfigCheckable(addrs.OutputValue{Name: "b"}.InModule(moduleChildAddr.Module()))
	outputBInstAddr := addrs.Checkable(addrs.OutputValue{Name: "b"}.Absolute(moduleChildAddr))
	checkBlockAAddr := addrs.ConfigCheckable(addrs.Check{Name: "a"}.InModule(addrs.RootModule))
	checkBlockAInstAddr := addrs.Checkable(addrs.Check{Name: "a"}.Absolute(addrs.RootModuleInstance))

	tests := map[string]struct {
		Input *states.CheckResults
		Want  any
	}{
		"empty": {
			&states.CheckResults{},
			[]any{},
		},
		"failures": {
			&states.CheckResults{
				ConfigResults: addrs.MakeMap(
					addrs.MakeMapElem(resourceAAddr, &states.CheckResultAggregate{
						Status: checks.StatusFail,
						ObjectResults: addrs.MakeMap(
							addrs.MakeMapElem(resourceAInstAddr, &states.CheckResultObject{
								Status: checks.StatusFail,
								FailureMessages: []string{
									"Not enough boops.",
									"Too many beeps.",
								},
							}),
						),
					}),
					addrs.MakeMapElem(resourceBAddr, &states.CheckResultAggregate{
						Status: checks.StatusFail,
						ObjectResults: addrs.MakeMap(
							addrs.MakeMapElem(resourceBInstAddr, &states.CheckResultObject{
								Status: checks.StatusFail,
								FailureMessages: []string{
									"Splines are too pointy.",
								},
							}),
						),
					}),
					addrs.MakeMapElem(outputAAddr, &states.CheckResultAggregate{
						Status: checks.StatusFail,
						ObjectResults: addrs.MakeMap(
							addrs.MakeMapElem(outputAInstAddr, &states.CheckResultObject{
								Status: checks.StatusFail,
							}),
						),
					}),
					addrs.MakeMapElem(outputBAddr, &states.CheckResultAggregate{
						Status: checks.StatusFail,
						ObjectResults: addrs.MakeMap(
							addrs.MakeMapElem(outputBInstAddr, &states.CheckResultObject{
								Status: checks.StatusFail,
								FailureMessages: []string{
									"Not object-oriented enough.",
								},
							}),
						),
					}),
					addrs.MakeMapElem(checkBlockAAddr, &states.CheckResultAggregate{
						Status: checks.StatusFail,
						ObjectResults: addrs.MakeMap(
							addrs.MakeMapElem(checkBlockAInstAddr, &states.CheckResultObject{
								Status: checks.StatusFail,
								FailureMessages: []string{
									"Couldn't reverse the polarity.",
								},
							}),
						),
					}),
				),
			},
			[]any{
				map[string]any{
					"address": map[string]any{
						"kind":       "check",
						"to_display": "check.a",
						"name":       "a",
					},
					"instances": []any{
						map[string]any{
							"address": map[string]any{
								"to_display": `check.a`,
							},
							"problems": []any{
								map[string]any{
									"message": "Couldn't reverse the polarity.",
								},
							},
							"status": "fail",
						},
					},
					"status": "fail",
				},
				map[string]any{
					"address": map[string]any{
						"kind":       "output_value",
						"module":     "module.child",
						"name":       "b",
						"to_display": "module.child.output.b",
					},
					"instances": []any{
						map[string]any{
							"address": map[string]any{
								"module":     "module.child[0]",
								"to_display": "module.child[0].output.b",
							},
							"problems": []any{
								map[string]any{
									"message": "Not object-oriented enough.",
								},
							},
							"status": "fail",
						},
					},
					"status": "fail",
				},
				map[string]any{
					"address": map[string]any{
						"kind":       "resource",
						"mode":       "managed",
						"module":     "module.child",
						"name":       "b",
						"to_display": "module.child.test.b",
						"type":       "test",
					},
					"instances": []any{
						map[string]any{
							"address": map[string]any{
								"module":     "module.child[0]",
								"to_display": "module.child[0].test.b",
							},
							"problems": []any{
								map[string]any{
									"message": "Splines are too pointy.",
								},
							},
							"status": "fail",
						},
					},
					"status": "fail",
				},
				map[string]any{
					"address": map[string]any{
						"kind":       "output_value",
						"name":       "a",
						"to_display": "output.a",
					},
					"instances": []any{
						map[string]any{
							"address": map[string]any{
								"to_display": "output.a",
							},
							"status": "fail",
						},
					},
					"status": "fail",
				},
				map[string]any{
					"address": map[string]any{
						"kind":       "resource",
						"mode":       "managed",
						"name":       "a",
						"to_display": "test.a",
						"type":       "test",
					},
					"instances": []any{
						map[string]any{
							"address": map[string]any{
								"to_display":   `test.a["foo"]`,
								"instance_key": "foo",
							},
							"problems": []any{
								map[string]any{
									"message": "Not enough boops.",
								},
								map[string]any{
									"message": "Too many beeps.",
								},
							},
							"status": "fail",
						},
					},
					"status": "fail",
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			gotBytes := MarshalCheckStates(test.Input)

			var got any
			err := json.Unmarshal(gotBytes, &got)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(test.Want, got); diff != "" {
				t.Errorf("wrong result\n%s", diff)
			}
		})
	}
}
