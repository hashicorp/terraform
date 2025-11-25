// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package planfile

import (
	"bytes"
	"testing"

	"github.com/go-test/deep"
	version "github.com/hashicorp/go-version"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/globalref"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
)

// TestTFPlanRoundTrip writes a plan to a planfile, reads the contents of the planfile,
// and asserts that the read data matches the written data.
func TestTFPlanRoundTrip(t *testing.T) {
	cases := map[string]struct {
		plan *plans.Plan
	}{
		"round trip with backend": {
			plan: func() *plans.Plan {
				rawPlan := examplePlanForTest(t)
				return rawPlan
			}(),
		},
		"round trip with state store": {
			plan: func() *plans.Plan {
				rawPlan := examplePlanForTest(t)
				// remove backend data from example plan
				rawPlan.Backend = nil

				// add state store instead
				ver, err := version.NewVersion("9.9.9")
				if err != nil {
					t.Fatalf("error encountered during test setup: %s", err)
				}

				// add state store instead
				rawPlan.StateStore = &plans.StateStore{
					Type: "foo_bar",
					Provider: &plans.Provider{
						Version: ver,
						Source: &tfaddr.Provider{
							Hostname:  tfaddr.DefaultProviderRegistryHost,
							Namespace: "foobar",
							Type:      "foo",
						},
						// Imagining a provider that has nothing in its schema
						Config: mustNewDynamicValue(
							cty.EmptyObjectVal,
							cty.Object(nil),
						),
					},
					// Imagining a state store with a field called `foo` in its schema
					Config: mustNewDynamicValue(
						cty.ObjectVal(map[string]cty.Value{
							"foo": cty.StringVal("bar"),
						}),
						cty.Object(map[string]cty.Type{
							"foo": cty.String,
						}),
					),
					Workspace: "default",
				}
				return rawPlan
			}(),
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			var buf bytes.Buffer
			err := writeTfplan(tc.plan, &buf)
			if err != nil {
				t.Fatalf("unexpected err: %s", err)
			}

			newPlan, err := readTfplan(&buf)
			if err != nil {
				t.Fatal(err)
			}

			{
				oldDepth := deep.MaxDepth
				oldCompare := deep.CompareUnexportedFields
				deep.MaxDepth = 20
				deep.CompareUnexportedFields = true
				defer func() {
					deep.MaxDepth = oldDepth
					deep.CompareUnexportedFields = oldCompare
				}()
			}
			for _, problem := range deep.Equal(newPlan, tc.plan) {
				t.Error(problem)
			}
		})
	}
}

func Test_writeTfplan_validation(t *testing.T) {
	cases := map[string]struct {
		plan            *plans.Plan
		wantWriteErrMsg string
	}{
		"error when missing both backend and state store": {
			plan: func() *plans.Plan {
				rawPlan := examplePlanForTest(t)
				// remove backend from example plan
				rawPlan.Backend = nil
				return rawPlan
			}(),
			wantWriteErrMsg: "plan does not have a backend or state_store configuration",
		},
		"error when got both backend and state store": {
			plan: func() *plans.Plan {
				rawPlan := examplePlanForTest(t)
				// Backend is already set on example plan

				// Add state store in parallel
				ver, err := version.NewVersion("9.9.9")
				if err != nil {
					t.Fatalf("error encountered during test setup: %s", err)
				}
				rawPlan.StateStore = &plans.StateStore{
					Type: "foo_bar",
					Provider: &plans.Provider{
						Version: ver,
						Source: &tfaddr.Provider{
							Hostname:  tfaddr.DefaultProviderRegistryHost,
							Namespace: "foobar",
							Type:      "foo",
						},
						Config: mustNewDynamicValue(
							cty.ObjectVal(map[string]cty.Value{
								"foo": cty.StringVal("bar"),
							}),
							cty.Object(map[string]cty.Type{
								"foo": cty.String,
							}),
						),
					},
					Config: mustNewDynamicValue(
						cty.ObjectVal(map[string]cty.Value{
							"foo": cty.StringVal("bar"),
						}),
						cty.Object(map[string]cty.Type{
							"foo": cty.String,
						}),
					),
					Workspace: "default",
				}
				return rawPlan
			}(),
			wantWriteErrMsg: "plan contains both backend and state_store configurations, only one is expected",
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			var buf bytes.Buffer
			err := writeTfplan(tc.plan, &buf)
			if err == nil {
				t.Fatal("this test expects an error but got none")
			}
			if err.Error() != tc.wantWriteErrMsg {
				t.Fatalf("unexpected error message: wanted %q, got %q", tc.wantWriteErrMsg, err)
			}
		})
	}
}

// examplePlanForTest returns a plans.Plan struct pointer that can be used
// when setting up tests. The returned plan can be mutated depending on the
// test case.
func examplePlanForTest(t *testing.T) *plans.Plan {
	t.Helper()
	objTy := cty.Object(map[string]cty.Type{
		"id": cty.String,
	})
	applyTimeVariables := collections.NewSetCmp[string]()
	applyTimeVariables.Add("bar")

	provider := addrs.AbsProviderConfig{
		Provider: addrs.NewDefaultProvider("test"),
		Module:   addrs.RootModule,
	}

	return &plans.Plan{
		Applyable: true,
		Complete:  true,
		VariableValues: map[string]plans.DynamicValue{
			"foo": mustNewDynamicValueStr("foo value"),
		},
		ApplyTimeVariables: applyTimeVariables,
		Changes: &plans.ChangesSrc{
			Outputs: []*plans.OutputChangeSrc{
				{
					Addr: addrs.OutputValue{Name: "bar"}.Absolute(addrs.RootModuleInstance),
					ChangeSrc: plans.ChangeSrc{
						Action: plans.Create,
						After:  mustDynamicOutputValue("bar value"),
					},
					Sensitive: false,
				},
				{
					Addr: addrs.OutputValue{Name: "baz"}.Absolute(addrs.RootModuleInstance),
					ChangeSrc: plans.ChangeSrc{
						Action: plans.NoOp,
						Before: mustDynamicOutputValue("baz value"),
						After:  mustDynamicOutputValue("baz value"),
					},
					Sensitive: false,
				},
				{
					Addr: addrs.OutputValue{Name: "secret"}.Absolute(addrs.RootModuleInstance),
					ChangeSrc: plans.ChangeSrc{
						Action: plans.Update,
						Before: mustDynamicOutputValue("old secret value"),
						After:  mustDynamicOutputValue("new secret value"),
					},
					Sensitive: true,
				},
			},
			Resources: []*plans.ResourceInstanceChangeSrc{
				{
					Addr: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_thing",
						Name: "woot",
					}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),
					PrevRunAddr: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_thing",
						Name: "woot",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					ProviderAddr: provider,
					ChangeSrc: plans.ChangeSrc{
						Action: plans.DeleteThenCreate,
						Before: mustNewDynamicValue(cty.ObjectVal(map[string]cty.Value{
							"id": cty.StringVal("foo-bar-baz"),
							"boop": cty.ListVal([]cty.Value{
								cty.StringVal("beep"),
							}),
						}), objTy),
						After: mustNewDynamicValue(cty.ObjectVal(map[string]cty.Value{
							"id": cty.UnknownVal(cty.String),
							"boop": cty.ListVal([]cty.Value{
								cty.StringVal("beep"),
								cty.StringVal("honk"),
							}),
						}), objTy),
						AfterSensitivePaths: []cty.Path{
							cty.GetAttrPath("boop").IndexInt(1),
						},
					},
					RequiredReplace: cty.NewPathSet(
						cty.GetAttrPath("boop"),
					),
					ActionReason: plans.ResourceInstanceReplaceBecauseCannotUpdate,
				},
				{
					Addr: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_thing",
						Name: "woot",
					}.Instance(addrs.IntKey(1)).Absolute(addrs.RootModuleInstance),
					PrevRunAddr: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_thing",
						Name: "woot",
					}.Instance(addrs.IntKey(1)).Absolute(addrs.RootModuleInstance),
					DeposedKey: "foodface",
					ProviderAddr: addrs.AbsProviderConfig{
						Provider: addrs.NewDefaultProvider("test"),
						Module:   addrs.RootModule,
					},
					ChangeSrc: plans.ChangeSrc{
						Action: plans.Delete,
						Before: mustNewDynamicValue(cty.ObjectVal(map[string]cty.Value{
							"id": cty.StringVal("bar-baz-foo"),
						}), objTy),
					},
				},
				{
					Addr: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_thing",
						Name: "importing",
					}.Instance(addrs.IntKey(1)).Absolute(addrs.RootModuleInstance),
					PrevRunAddr: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_thing",
						Name: "importing",
					}.Instance(addrs.IntKey(1)).Absolute(addrs.RootModuleInstance),
					ProviderAddr: provider,
					ChangeSrc: plans.ChangeSrc{
						Action: plans.NoOp,
						Before: mustNewDynamicValue(cty.ObjectVal(map[string]cty.Value{
							"id": cty.StringVal("testing"),
						}), objTy),
						After: mustNewDynamicValue(cty.ObjectVal(map[string]cty.Value{
							"id": cty.StringVal("testing"),
						}), objTy),
						Importing:       &plans.ImportingSrc{ID: "testing"},
						GeneratedConfig: "resource \\\"test_thing\\\" \\\"importing\\\" {}",
					},
				},
			},
			ActionInvocations: []*plans.ActionInvocationInstanceSrc{
				{
					Addr:         addrs.Action{Type: "example", Name: "foo"}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					ProviderAddr: provider,
					ActionTrigger: &plans.LifecycleActionTrigger{
						ActionTriggerEvent:      configs.BeforeCreate,
						ActionTriggerBlockIndex: 2,
						ActionsListIndex:        0,
						TriggeringResourceAddr: addrs.Resource{
							Mode: addrs.ManagedResourceMode,
							Type: "test_thing",
							Name: "woot",
						}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),
					},
				},
				{
					Addr:         addrs.Action{Type: "example", Name: "bar"}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					ProviderAddr: provider,
					ActionTrigger: &plans.LifecycleActionTrigger{
						ActionTriggerEvent:      configs.BeforeCreate,
						ActionTriggerBlockIndex: 2,
						ActionsListIndex:        1,
						TriggeringResourceAddr: addrs.Resource{
							Mode: addrs.ManagedResourceMode,
							Type: "test_thing",
							Name: "woot",
						}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),
					},
					ConfigValue: mustNewDynamicValue(cty.ObjectVal(map[string]cty.Value{
						"id": cty.StringVal("testing"),
					}), objTy),
				},
				{
					Addr:         addrs.Action{Type: "example", Name: "baz"}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					ProviderAddr: provider,
					ActionTrigger: &plans.LifecycleActionTrigger{
						ActionTriggerEvent:      configs.BeforeCreate,
						ActionTriggerBlockIndex: 2,
						ActionsListIndex:        1,
						TriggeringResourceAddr: addrs.Resource{
							Mode: addrs.ManagedResourceMode,
							Type: "test_thing",
							Name: "woot",
						}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),
					},
					ConfigValue: mustNewDynamicValue(cty.ObjectVal(map[string]cty.Value{
						"id": cty.StringVal("secret"),
					}), objTy),
					SensitiveConfigPaths: []cty.Path{cty.GetAttrPath("id")},
				},
			},
		},
		DriftedResources: []*plans.ResourceInstanceChangeSrc{
			{
				Addr: addrs.Resource{
					Mode: addrs.ManagedResourceMode,
					Type: "test_thing",
					Name: "woot",
				}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),
				PrevRunAddr: addrs.Resource{
					Mode: addrs.ManagedResourceMode,
					Type: "test_thing",
					Name: "woot",
				}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
				ProviderAddr: provider,
				ChangeSrc: plans.ChangeSrc{
					Action: plans.DeleteThenCreate,
					Before: mustNewDynamicValue(cty.ObjectVal(map[string]cty.Value{
						"id": cty.StringVal("foo-bar-baz"),
						"boop": cty.ListVal([]cty.Value{
							cty.StringVal("beep"),
						}),
					}), objTy),
					After: mustNewDynamicValue(cty.ObjectVal(map[string]cty.Value{
						"id": cty.UnknownVal(cty.String),
						"boop": cty.ListVal([]cty.Value{
							cty.StringVal("beep"),
							cty.StringVal("bonk"),
						}),
					}), objTy),
					AfterSensitivePaths: []cty.Path{
						cty.GetAttrPath("boop").IndexInt(1),
					},
				},
			},
		},
		DeferredResources: []*plans.DeferredResourceInstanceChangeSrc{
			{
				DeferredReason: providers.DeferredReasonInstanceCountUnknown,
				ChangeSrc: &plans.ResourceInstanceChangeSrc{
					Addr: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_thing",
						Name: "woot",
					}.Instance(addrs.WildcardKey).Absolute(addrs.RootModuleInstance),
					ProviderAddr: provider,
					ChangeSrc: plans.ChangeSrc{
						Action: plans.Create,
						After: mustNewDynamicValue(cty.ObjectVal(map[string]cty.Value{
							"id": cty.UnknownVal(cty.String),
							"boop": cty.ListVal([]cty.Value{
								cty.StringVal("beep"),
								cty.StringVal("bonk"),
							}),
						}), objTy),
					},
				},
			},
			{
				DeferredReason: providers.DeferredReasonInstanceCountUnknown,
				ChangeSrc: &plans.ResourceInstanceChangeSrc{
					Addr: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_thing",
						Name: "woot",
					}.Instance(addrs.WildcardKey).Absolute(addrs.ModuleInstance{
						addrs.ModuleInstanceStep{
							Name:        "mod",
							InstanceKey: addrs.WildcardKey,
						},
					}),
					ProviderAddr: provider,
					ChangeSrc: plans.ChangeSrc{
						Action: plans.Create,
						After: mustNewDynamicValue(cty.ObjectVal(map[string]cty.Value{
							"id": cty.UnknownVal(cty.String),
							"boop": cty.ListVal([]cty.Value{
								cty.StringVal("beep"),
								cty.StringVal("bonk"),
							}),
						}), objTy),
					},
				},
			},
		},
		DeferredActionInvocations: []*plans.DeferredActionInvocationSrc{
			{
				DeferredReason: providers.DeferredReasonDeferredPrereq,
				ActionInvocationInstanceSrc: &plans.ActionInvocationInstanceSrc{
					Addr: addrs.Action{Type: "test_action", Name: "example"}.Absolute(addrs.RootModuleInstance).Instance(addrs.NoKey),
					ActionTrigger: &plans.LifecycleActionTrigger{
						TriggeringResourceAddr: addrs.Resource{
							Mode: addrs.ManagedResourceMode,
							Type: "test_thing",
							Name: "woot",
						}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),
						ActionTriggerBlockIndex: 1,
						ActionsListIndex:        2,
						ActionTriggerEvent:      configs.AfterCreate,
					},
					ProviderAddr: provider,
					ConfigValue: mustNewDynamicValue(cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("value"),
					}), cty.Object(map[string]cty.Type{
						"attr": cty.String,
					})),
				},
			},
		},
		RelevantAttributes: []globalref.ResourceAttr{
			{
				Resource: addrs.Resource{
					Mode: addrs.ManagedResourceMode,
					Type: "test_thing",
					Name: "woot",
				}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),
				Attr: cty.GetAttrPath("boop").Index(cty.NumberIntVal(1)),
			},
		},
		Checks: &states.CheckResults{
			ConfigResults: addrs.MakeMap(
				addrs.MakeMapElem[addrs.ConfigCheckable](
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_thing",
						Name: "woot",
					}.InModule(addrs.RootModule),
					&states.CheckResultAggregate{
						Status: checks.StatusFail,
						ObjectResults: addrs.MakeMap(
							addrs.MakeMapElem[addrs.Checkable](
								addrs.Resource{
									Mode: addrs.ManagedResourceMode,
									Type: "test_thing",
									Name: "woot",
								}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),
								&states.CheckResultObject{
									Status:          checks.StatusFail,
									FailureMessages: []string{"Oh no!"},
								},
							),
						),
					},
				),
				addrs.MakeMapElem[addrs.ConfigCheckable](
					addrs.Check{
						Name: "check",
					}.InModule(addrs.RootModule),
					&states.CheckResultAggregate{
						Status: checks.StatusFail,
						ObjectResults: addrs.MakeMap(
							addrs.MakeMapElem[addrs.Checkable](
								addrs.Check{
									Name: "check",
								}.Absolute(addrs.RootModuleInstance),
								&states.CheckResultObject{
									Status:          checks.StatusFail,
									FailureMessages: []string{"check failed"},
								},
							),
						),
					},
				),
			),
		},
		TargetAddrs: []addrs.Targetable{
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_thing",
				Name: "woot",
			}.Absolute(addrs.RootModuleInstance),
		},
		Backend: &plans.Backend{
			Type: "local",
			Config: mustNewDynamicValue(
				cty.ObjectVal(map[string]cty.Value{
					"foo": cty.StringVal("bar"),
				}),
				cty.Object(map[string]cty.Type{
					"foo": cty.String,
				}),
			),
			Workspace: "default",
		},
	}
}

func mustDynamicOutputValue(val string) plans.DynamicValue {
	ret, err := plans.NewDynamicValue(cty.StringVal(val), cty.DynamicPseudoType)
	if err != nil {
		panic(err)
	}
	return ret
}

func mustNewDynamicValue(val cty.Value, ty cty.Type) plans.DynamicValue {
	ret, err := plans.NewDynamicValue(val, ty)
	if err != nil {
		panic(err)
	}
	return ret
}

func mustNewDynamicValueStr(val string) plans.DynamicValue {
	realVal := cty.StringVal(val)
	ret, err := plans.NewDynamicValue(realVal, cty.String)
	if err != nil {
		panic(err)
	}
	return ret
}

// TestTFPlanRoundTripDestroy ensures that encoding and decoding null values for
// destroy doesn't leave us with any nil values.
func TestTFPlanRoundTripDestroy(t *testing.T) {
	objTy := cty.Object(map[string]cty.Type{
		"id": cty.String,
	})

	objSchema := providers.Schema{
		Body: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"id": {
					Type:     cty.String,
					Required: true,
				},
			},
		},
	}

	plan := &plans.Plan{
		Changes: &plans.ChangesSrc{
			Outputs: []*plans.OutputChangeSrc{
				{
					Addr: addrs.OutputValue{Name: "bar"}.Absolute(addrs.RootModuleInstance),
					ChangeSrc: plans.ChangeSrc{
						Action: plans.Delete,
						Before: mustDynamicOutputValue("output"),
						After:  mustNewDynamicValue(cty.NullVal(cty.String), cty.String),
					},
				},
			},
			Resources: []*plans.ResourceInstanceChangeSrc{
				{
					Addr: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_thing",
						Name: "woot",
					}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),
					PrevRunAddr: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_thing",
						Name: "woot",
					}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),
					ProviderAddr: addrs.AbsProviderConfig{
						Provider: addrs.NewDefaultProvider("test"),
						Module:   addrs.RootModule,
					},
					ChangeSrc: plans.ChangeSrc{
						Action: plans.Delete,
						Before: mustNewDynamicValue(cty.ObjectVal(map[string]cty.Value{
							"id": cty.StringVal("foo-bar-baz"),
						}), objTy),
						After: mustNewDynamicValue(cty.NullVal(objTy), objTy),
					},
				},
			},
		},
		DriftedResources: []*plans.ResourceInstanceChangeSrc{},
		TargetAddrs: []addrs.Targetable{
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_thing",
				Name: "woot",
			}.Absolute(addrs.RootModuleInstance),
		},
		Backend: &plans.Backend{
			Type: "local",
			Config: mustNewDynamicValue(
				cty.ObjectVal(map[string]cty.Value{
					"foo": cty.StringVal("bar"),
				}),
				cty.Object(map[string]cty.Type{
					"foo": cty.String,
				}),
			),
			Workspace: "default",
		},
	}

	var buf bytes.Buffer
	err := writeTfplan(plan, &buf)
	if err != nil {
		t.Fatal(err)
	}

	newPlan, err := readTfplan(&buf)
	if err != nil {
		t.Fatal(err)
	}

	for _, rics := range newPlan.Changes.Resources {
		ric, err := rics.Decode(objSchema)
		if err != nil {
			t.Fatal(err)
		}

		if ric.After == cty.NilVal {
			t.Fatalf("unexpected nil After value: %#v\n", ric)
		}
	}
	for _, ocs := range newPlan.Changes.Outputs {
		oc, err := ocs.Decode()
		if err != nil {
			t.Fatal(err)
		}

		if oc.After == cty.NilVal {
			t.Fatalf("unexpected nil After value: %#v\n", ocs)
		}
	}
}
