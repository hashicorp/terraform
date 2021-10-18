package planfile

import (
	"bytes"
	"testing"

	"github.com/go-test/deep"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
)

func TestTFPlanRoundTrip(t *testing.T) {
	objTy := cty.Object(map[string]cty.Type{
		"id": cty.String,
	})

	plan := &plans.Plan{
		VariableValues: map[string]plans.DynamicValue{
			"foo": mustNewDynamicValueStr("foo value"),
		},
		Changes: &plans.Changes{
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
					ProviderAddr: addrs.AbsProviderConfig{
						Provider: addrs.NewDefaultProvider("test"),
						Module:   addrs.RootModule,
					},
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
						AfterValMarks: []cty.PathValueMarks{
							{
								Path:  cty.GetAttrPath("boop").IndexInt(1),
								Marks: cty.NewValueMarks(marks.Sensitive),
							},
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
				ProviderAddr: addrs.AbsProviderConfig{
					Provider: addrs.NewDefaultProvider("test"),
					Module:   addrs.RootModule,
				},
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
					AfterValMarks: []cty.PathValueMarks{
						{
							Path:  cty.GetAttrPath("boop").IndexInt(1),
							Marks: cty.NewValueMarks(marks.Sensitive),
						},
					},
				},
			},
		},
		TargetAddrs: []addrs.Targetable{
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_thing",
				Name: "woot",
			}.Absolute(addrs.RootModuleInstance),
		},
		Backend: plans.Backend{
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
	for _, problem := range deep.Equal(newPlan, plan) {
		t.Error(problem)
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

	plan := &plans.Plan{
		Changes: &plans.Changes{
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
		Backend: plans.Backend{
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
		ric, err := rics.Decode(objTy)
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
