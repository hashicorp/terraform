// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/namedvals"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/deferring"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestEvaluatorGetTerraformAttr(t *testing.T) {
	evaluator := &Evaluator{
		Meta: &ContextMeta{
			Env: "foo",
		},
		NamedValues: namedvals.NewState(),
	}
	data := &evaluationStateData{
		evaluationData: &evaluationData{
			Evaluator: evaluator,
		},
	}
	scope := evaluator.Scope(data, nil, nil, lang.ExternalFuncs{})

	t.Run("workspace", func(t *testing.T) {
		want := cty.StringVal("foo")
		got, diags := scope.Data.GetTerraformAttr(addrs.TerraformAttr{
			Name: "workspace",
		}, tfdiags.SourceRange{})
		if len(diags) != 0 {
			t.Errorf("unexpected diagnostics %s", spew.Sdump(diags))
		}
		if !got.RawEquals(want) {
			t.Errorf("wrong result %q; want %q", got, want)
		}
	})
}

func TestEvaluatorGetPathAttr(t *testing.T) {
	evaluator := &Evaluator{
		Meta: &ContextMeta{
			Env: "foo",
		},
		Config: &configs.Config{
			Module: &configs.Module{
				SourceDir: "bar/baz",
			},
		},
		NamedValues: namedvals.NewState(),
	}
	data := &evaluationStateData{
		evaluationData: &evaluationData{
			Evaluator: evaluator,
		},
	}
	scope := evaluator.Scope(data, nil, nil, lang.ExternalFuncs{})

	t.Run("module", func(t *testing.T) {
		want := cty.StringVal("bar/baz")
		got, diags := scope.Data.GetPathAttr(addrs.PathAttr{
			Name: "module",
		}, tfdiags.SourceRange{})
		if len(diags) != 0 {
			t.Errorf("unexpected diagnostics %s", spew.Sdump(diags))
		}
		if !got.RawEquals(want) {
			t.Errorf("wrong result %#v; want %#v", got, want)
		}
	})

	t.Run("root", func(t *testing.T) {
		want := cty.StringVal("bar/baz")
		got, diags := scope.Data.GetPathAttr(addrs.PathAttr{
			Name: "root",
		}, tfdiags.SourceRange{})
		if len(diags) != 0 {
			t.Errorf("unexpected diagnostics %s", spew.Sdump(diags))
		}
		if !got.RawEquals(want) {
			t.Errorf("wrong result %#v; want %#v", got, want)
		}
	})
}

func TestEvaluatorGetOutputValue(t *testing.T) {
	nvs := namedvals.NewState()
	nvs.SetOutputValue(mustAbsOutputValue("output.some_output"), cty.StringVal("first"))
	nvs.SetOutputValue(mustAbsOutputValue("output.some_other_output"), cty.StringVal("second"))
	nvs.SetOutputValue(mustAbsOutputValue("output.some_third_output"), cty.StringVal("third").Mark(marks.Sensitive))

	evaluator := &Evaluator{
		Meta: &ContextMeta{
			Env: "foo",
		},
		Config: &configs.Config{
			Module: &configs.Module{
				Outputs: map[string]*configs.Output{
					"some_output": {
						Name:      "some_output",
						Sensitive: true,
					},
					"some_other_output": {
						Name: "some_other_output",
					},
					"some_third_output": {
						Name: "some_third_output",
					},
				},
			},
		},
		NamedValues: nvs,
	}

	data := &evaluationStateData{
		evaluationData: &evaluationData{
			Evaluator: evaluator,
		},
	}
	scope := evaluator.Scope(data, nil, nil, lang.ExternalFuncs{})

	want := cty.StringVal("first").Mark(marks.Sensitive)
	got, diags := scope.Data.GetOutput(addrs.OutputValue{
		Name: "some_output",
	}, tfdiags.SourceRange{})

	if len(diags) != 0 {
		t.Errorf("unexpected diagnostics %s", spew.Sdump(diags))
	}
	if !got.RawEquals(want) {
		t.Errorf("wrong result %#v; want %#v", got, want)
	}

	want = cty.StringVal("second")
	got, diags = scope.Data.GetOutput(addrs.OutputValue{
		Name: "some_other_output",
	}, tfdiags.SourceRange{})

	if len(diags) != 0 {
		t.Errorf("unexpected diagnostics %s", spew.Sdump(diags))
	}
	if !got.RawEquals(want) {
		t.Errorf("wrong result %#v; want %#v", got, want)
	}

	want = cty.StringVal("third").Mark(marks.Sensitive)
	got, diags = scope.Data.GetOutput(addrs.OutputValue{
		Name: "some_third_output",
	}, tfdiags.SourceRange{})

	if len(diags) != 0 {
		t.Errorf("unexpected diagnostics %s", spew.Sdump(diags))
	}
	if !got.RawEquals(want) {
		t.Errorf("wrong result %#v; want %#v", got, want)
	}
}

// This particularly tests that a sensitive attribute in config
// results in a value that has a "sensitive" cty Mark
func TestEvaluatorGetInputVariable(t *testing.T) {
	namedValues := namedvals.NewState()
	namedValues.SetInputVariableValue(
		addrs.RootModuleInstance.InputVariable("some_var"), cty.StringVal("bar"),
	)
	namedValues.SetInputVariableValue(
		addrs.RootModuleInstance.InputVariable("some_other_var"), cty.StringVal("boop").Mark(marks.Sensitive),
	)

	evaluator := &Evaluator{
		Meta: &ContextMeta{
			Env: "foo",
		},
		Config: &configs.Config{
			Module: &configs.Module{
				Variables: map[string]*configs.Variable{
					"some_var": {
						Name:           "some_var",
						Sensitive:      true,
						Default:        cty.StringVal("foo"),
						Type:           cty.String,
						ConstraintType: cty.String,
					},
					// Avoid double marking a value
					"some_other_var": {
						Name:           "some_other_var",
						Sensitive:      true,
						Default:        cty.StringVal("bar"),
						Type:           cty.String,
						ConstraintType: cty.String,
					},
				},
			},
		},
		NamedValues: namedValues,
	}

	data := &evaluationStateData{
		evaluationData: &evaluationData{
			Evaluator: evaluator,
		},
	}
	scope := evaluator.Scope(data, nil, nil, lang.ExternalFuncs{})

	want := cty.StringVal("bar").Mark(marks.Sensitive)
	got, diags := scope.Data.GetInputVariable(addrs.InputVariable{
		Name: "some_var",
	}, tfdiags.SourceRange{})

	if len(diags) != 0 {
		t.Errorf("unexpected diagnostics %s", spew.Sdump(diags))
	}
	if !got.RawEquals(want) {
		t.Errorf("wrong result %#v; want %#v", got, want)
	}

	want = cty.StringVal("boop").Mark(marks.Sensitive)
	got, diags = scope.Data.GetInputVariable(addrs.InputVariable{
		Name: "some_other_var",
	}, tfdiags.SourceRange{})

	if len(diags) != 0 {
		t.Errorf("unexpected diagnostics %s", spew.Sdump(diags))
	}
	if !got.RawEquals(want) {
		t.Errorf("wrong result %#v; want %#v", got, want)
	}
}

func TestEvaluatorGetResource(t *testing.T) {
	stateSync := states.BuildState(func(ss *states.SyncState) {
		ss.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_resource",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				Status:    states.ObjectReady,
				AttrsJSON: []byte(`{"id":"foo", "nesting_list": [{"sensitive_value":"abc"}], "nesting_map": {"foo":{"foo":"x"}}, "nesting_set": [{"baz":"abc"}], "nesting_single": {"boop":"abc"}, "nesting_nesting": {"nesting_list":[{"sensitive_value":"abc"}]}, "value":"hello"}`),
				AttrSensitivePaths: []cty.Path{
					cty.GetAttrPath("nesting_list").IndexInt(0).GetAttr("sensitive_value"),
					cty.GetAttrPath("nesting_map").IndexString("foo").GetAttr("foo"),
					cty.GetAttrPath("nesting_nesting").GetAttr("nesting_list").IndexInt(0).GetAttr("sensitive_value"),
					cty.GetAttrPath("nesting_set"),
					cty.GetAttrPath("nesting_single").GetAttr("boop"),
					cty.GetAttrPath("value"),
				},
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	}).SyncWrapper()

	rc := &configs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test_resource",
		Name: "foo",
		Config: configs.SynthBody("", map[string]cty.Value{
			"id": cty.StringVal("foo"),
		}),
		Provider: addrs.Provider{
			Hostname:  addrs.DefaultProviderRegistryHost,
			Namespace: "hashicorp",
			Type:      "test",
		},
	}

	evaluator := &Evaluator{
		Meta: &ContextMeta{
			Env: "foo",
		},
		Changes: plans.NewChanges().SyncWrapper(),
		Config: &configs.Config{
			Module: &configs.Module{
				ManagedResources: map[string]*configs.Resource{
					"test_resource.foo": rc,
				},
			},
		},
		State:       stateSync,
		NamedValues: namedvals.NewState(),
		Deferrals:   deferring.NewDeferred(false),
		Plugins: schemaOnlyProvidersForTesting(map[addrs.Provider]providers.ProviderSchema{
			addrs.NewDefaultProvider("test"): {
				ResourceTypes: map[string]providers.Schema{
					"test_resource": {
						Body: &configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"id": {
									Type:     cty.String,
									Computed: true,
								},
								"value": {
									Type:      cty.String,
									Computed:  true,
									Sensitive: true,
								},
							},
							BlockTypes: map[string]*configschema.NestedBlock{
								"nesting_list": {
									Block: configschema.Block{
										Attributes: map[string]*configschema.Attribute{
											"value":           {Type: cty.String, Optional: true},
											"sensitive_value": {Type: cty.String, Optional: true, Sensitive: true},
										},
									},
									Nesting: configschema.NestingList,
								},
								"nesting_map": {
									Block: configschema.Block{
										Attributes: map[string]*configschema.Attribute{
											"foo": {Type: cty.String, Optional: true, Sensitive: true},
										},
									},
									Nesting: configschema.NestingMap,
								},
								"nesting_set": {
									Block: configschema.Block{
										Attributes: map[string]*configschema.Attribute{
											"baz": {Type: cty.String, Optional: true, Sensitive: true},
										},
									},
									Nesting: configschema.NestingSet,
								},
								"nesting_single": {
									Block: configschema.Block{
										Attributes: map[string]*configschema.Attribute{
											"boop": {Type: cty.String, Optional: true, Sensitive: true},
										},
									},
									Nesting: configschema.NestingSingle,
								},
								"nesting_nesting": {
									Block: configschema.Block{
										BlockTypes: map[string]*configschema.NestedBlock{
											"nesting_list": {
												Block: configschema.Block{
													Attributes: map[string]*configschema.Attribute{
														"value":           {Type: cty.String, Optional: true},
														"sensitive_value": {Type: cty.String, Optional: true, Sensitive: true},
													},
												},
												Nesting: configschema.NestingList,
											},
										},
									},
									Nesting: configschema.NestingSingle,
								},
							},
						},
					},
				},
			},
		}),
	}

	data := &evaluationStateData{
		evaluationData: &evaluationData{
			Evaluator: evaluator,
		},
	}
	scope := evaluator.Scope(data, nil, nil, lang.ExternalFuncs{})

	want := cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("foo"),
		"nesting_list": cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"sensitive_value": cty.StringVal("abc").Mark(marks.Sensitive),
				"value":           cty.NullVal(cty.String),
			}),
		}),
		"nesting_map": cty.MapVal(map[string]cty.Value{
			"foo": cty.ObjectVal(map[string]cty.Value{"foo": cty.StringVal("x").Mark(marks.Sensitive)}),
		}),
		"nesting_nesting": cty.ObjectVal(map[string]cty.Value{
			"nesting_list": cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"sensitive_value": cty.StringVal("abc").Mark(marks.Sensitive),
					"value":           cty.NullVal(cty.String),
				}),
			}),
		}),
		"nesting_set": cty.SetVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"baz": cty.StringVal("abc").Mark(marks.Sensitive),
			}),
		}),
		"nesting_single": cty.ObjectVal(map[string]cty.Value{
			"boop": cty.StringVal("abc").Mark(marks.Sensitive),
		}),
		"value": cty.StringVal("hello").Mark(marks.Sensitive),
	})

	addr := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test_resource",
		Name: "foo",
	}
	got, diags := scope.Data.GetResource(addr, tfdiags.SourceRange{})

	if len(diags) != 0 {
		t.Errorf("unexpected diagnostics %s", spew.Sdump(diags))
	}

	if !got.RawEquals(want) {
		t.Errorf("wrong result:\ngot: %#v\nwant: %#v", got, want)
	}
}

// GetResource will return a planned object's After value
// if there is a change for that resource instance.
func TestEvaluatorGetResource_changes(t *testing.T) {
	// Set up existing state
	stateSync := states.BuildState(func(ss *states.SyncState) {
		ss.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_resource",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				Status:    states.ObjectPlanned,
				AttrsJSON: []byte(`{"id":"foo", "to_mark_val":"tacos", "sensitive_value":"abc"}`),
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	}).SyncWrapper()

	// Create a change for the existing state resource,
	// to exercise retrieving the After value of the change
	changesSync := plans.NewChanges().SyncWrapper()
	change := &plans.ResourceInstanceChange{
		Addr: mustResourceInstanceAddr("test_resource.foo"),
		ProviderAddr: addrs.AbsProviderConfig{
			Module:   addrs.RootModule,
			Provider: addrs.NewDefaultProvider("test"),
		},
		Change: plans.Change{
			Action: plans.Update,
			// Provide an After value that contains a marked value
			After: cty.ObjectVal(map[string]cty.Value{
				"id":              cty.StringVal("foo"),
				"to_mark_val":     cty.StringVal("pizza").Mark(marks.Sensitive),
				"sensitive_value": cty.StringVal("abc").Mark(marks.Sensitive),
				"sensitive_collection": cty.MapVal(map[string]cty.Value{
					"boop": cty.StringVal("beep"),
				}).Mark(marks.Sensitive),
			}),
		},
	}

	// Set up our schemas
	schemas := &Schemas{
		Providers: map[addrs.Provider]providers.ProviderSchema{
			addrs.NewDefaultProvider("test"): {
				ResourceTypes: map[string]providers.Schema{
					"test_resource": {
						Body: &configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"id": {
									Type:     cty.String,
									Computed: true,
								},
								"to_mark_val": {
									Type:     cty.String,
									Computed: true,
								},
								"sensitive_value": {
									Type:      cty.String,
									Computed:  true,
									Sensitive: true,
								},
								"sensitive_collection": {
									Type:      cty.Map(cty.String),
									Computed:  true,
									Sensitive: true,
								},
							},
						},
					},
				},
			},
		},
	}

	// The resource we'll inspect
	addr := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test_resource",
		Name: "foo",
	}

	changesSync.AppendResourceInstanceChange(change)

	evaluator := &Evaluator{
		Meta: &ContextMeta{
			Env: "foo",
		},
		Changes: changesSync,
		Config: &configs.Config{
			Module: &configs.Module{
				ManagedResources: map[string]*configs.Resource{
					"test_resource.foo": {
						Mode: addrs.ManagedResourceMode,
						Type: "test_resource",
						Name: "foo",
						Provider: addrs.Provider{
							Hostname:  addrs.DefaultProviderRegistryHost,
							Namespace: "hashicorp",
							Type:      "test",
						},
					},
				},
			},
		},
		State:       stateSync,
		NamedValues: namedvals.NewState(),
		Deferrals:   deferring.NewDeferred(false),
		Plugins:     schemaOnlyProvidersForTesting(schemas.Providers),
	}

	data := &evaluationStateData{
		evaluationData: &evaluationData{
			Evaluator: evaluator,
		},
	}
	scope := evaluator.Scope(data, nil, nil, lang.ExternalFuncs{})

	want := cty.ObjectVal(map[string]cty.Value{
		"id":              cty.StringVal("foo"),
		"to_mark_val":     cty.StringVal("pizza").Mark(marks.Sensitive),
		"sensitive_value": cty.StringVal("abc").Mark(marks.Sensitive),
		"sensitive_collection": cty.MapVal(map[string]cty.Value{
			"boop": cty.StringVal("beep"),
		}).Mark(marks.Sensitive),
	})

	got, diags := scope.Data.GetResource(addr, tfdiags.SourceRange{})

	if len(diags) != 0 {
		t.Errorf("unexpected diagnostics %s", spew.Sdump(diags))
	}

	if !got.RawEquals(want) {
		t.Errorf("wrong result:\ngot: %#v\nwant: %#v", got, want)
	}
}

// TestEvaluatorGetResourceInstance checks that GetResourceInstance returns
// the same value as indexing the value returned by GetResource, and declines
// to return a value whenever that can't be determined from the instance alone
// or the instance itself has a problem.
func TestEvaluatorGetResourceInstance(t *testing.T) {
	providerAddr := addrs.AbsProviderConfig{
		Provider: addrs.NewDefaultProvider("test"),
		Module:   addrs.RootModule,
	}
	resource := func(name string) addrs.Resource {
		return addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_resource",
			Name: name,
		}
	}
	instAddr := func(name string, key addrs.InstanceKey) addrs.AbsResourceInstance {
		return resource(name).Instance(key).Absolute(addrs.RootModuleInstance)
	}
	ready := func(id string) *states.ResourceInstanceObjectSrc {
		return &states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(fmt.Sprintf(`{"id":%q,"secret":"shh","old":"x"}`, id)),
		}
	}
	planned := &states.ResourceInstanceObjectSrc{
		Status:    states.ObjectPlanned,
		AttrsJSON: []byte(`{"id":"planned","secret":"shh","old":"x"}`),
	}

	stateSync := states.BuildState(func(ss *states.SyncState) {
		ss.SetResourceInstanceCurrent(instAddr("c", addrs.IntKey(0)), ready("c0"), providerAddr)
		ss.SetResourceInstanceCurrent(instAddr("c", addrs.IntKey(1)), planned, providerAddr)
		ss.SetResourceInstanceDeposed(instAddr("c", addrs.IntKey(2)), states.NewDeposedKey(), ready("c2-deposed"), providerAddr)
		ss.SetResourceInstanceCurrent(instAddr("c", addrs.IntKey(3)), ready("c3"), providerAddr)
		ss.SetResourceInstanceCurrent(instAddr("c", addrs.IntKey(4)), planned, providerAddr)
		ss.SetResourceInstanceCurrent(instAddr("f", addrs.StringKey("a")), ready("fa"), providerAddr)
		ss.SetResourceInstanceCurrent(instAddr("f", addrs.StringKey("b")), ready("fb"), providerAddr)
		ss.SetResourceInstanceCurrent(instAddr("s", addrs.NoKey), ready("s"), providerAddr)
		ss.SetResourceInstanceCurrent(instAddr("m", addrs.StringKey("a")), ready("ma"), providerAddr)
		// state which can't be decoded with the current schema, as if it
		// hadn't been upgraded yet
		ss.SetResourceInstanceCurrent(instAddr("m", addrs.StringKey("b")), &states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"mb","removed":"x"}`),
		}, providerAddr)
	}).SyncWrapper()

	objTy := cty.Object(map[string]cty.Type{
		"id":     cty.String,
		"secret": cty.String,
		"old":    cty.String,
	})
	changesSync := plans.NewChanges().SyncWrapper()
	changesSync.AppendResourceInstanceChange(&plans.ResourceInstanceChange{
		Addr:         instAddr("c", addrs.IntKey(1)),
		PrevRunAddr:  instAddr("c", addrs.IntKey(1)),
		ProviderAddr: providerAddr,
		Change: plans.Change{
			Action: plans.Update,
			Before: cty.NullVal(objTy),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":     cty.StringVal("c1"),
				"secret": cty.StringVal("new").Mark(marks.Sensitive),
				"old":    cty.UnknownVal(cty.String),
			}),
		},
	})
	changesSync.AppendResourceInstanceChange(&plans.ResourceInstanceChange{
		Addr:         instAddr("c", addrs.IntKey(3)),
		PrevRunAddr:  instAddr("c", addrs.IntKey(3)),
		ProviderAddr: providerAddr,
		Change: plans.Change{
			Action: plans.Delete,
			Before: cty.NullVal(objTy),
			After:  cty.NullVal(objTy),
		},
	})

	schemas := &Schemas{
		Providers: map[addrs.Provider]providers.ProviderSchema{
			addrs.NewDefaultProvider("test"): {
				ResourceTypes: map[string]providers.Schema{
					"test_resource": {
						Body: &configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"id":     {Type: cty.String, Computed: true},
								"secret": {Type: cty.String, Optional: true, Sensitive: true},
								"old":    {Type: cty.String, Optional: true, Deprecated: true},
							},
						},
					},
				},
			},
		},
	}

	resourceConfig := func(name string) *configs.Resource {
		return &configs.Resource{
			Mode:     addrs.ManagedResourceMode,
			Type:     "test_resource",
			Name:     name,
			Provider: addrs.NewDefaultProvider("test"),
		}
	}
	countConfig := resourceConfig("c")
	countConfig.Count = hcltest.MockExprLiteral(cty.NumberIntVal(5))
	forEachConfig := resourceConfig("f")
	forEachConfig.ForEach = hcltest.MockExprLiteral(cty.MapVal(map[string]cty.Value{
		"a": cty.StringVal("a"),
		"b": cty.StringVal("b"),
	}))
	invalidConfig := resourceConfig("m")
	invalidConfig.ForEach = forEachConfig.ForEach

	evaluator := &Evaluator{
		Meta:      &ContextMeta{Env: "foo"},
		Operation: walkPlan,
		Config: &configs.Config{
			Module: &configs.Module{
				ManagedResources: map[string]*configs.Resource{
					"test_resource.c": countConfig,
					"test_resource.f": forEachConfig,
					"test_resource.m": invalidConfig,
					"test_resource.s": resourceConfig("s"),
				},
			},
		},
		State:       stateSync,
		Changes:     changesSync,
		Instances:   instances.NewExpander(nil),
		NamedValues: namedvals.NewState(),
		Deferrals:   deferring.NewDeferred(false),
		Plugins:     schemaOnlyProvidersForTesting(schemas.Providers),
	}
	evaluator.Instances.SetResourceCount(addrs.RootModuleInstance, resource("c"), 5)
	evaluator.Instances.SetResourceForEach(addrs.RootModuleInstance, resource("f"), map[string]cty.Value{
		"a": cty.StringVal("a"),
		"b": cty.StringVal("b"),
	})
	evaluator.Instances.SetResourceForEach(addrs.RootModuleInstance, resource("m"), map[string]cty.Value{
		"a": cty.StringVal("a"),
		"b": cty.StringVal("b"),
	})
	evaluator.Instances.SetResourceSingle(addrs.RootModuleInstance, resource("s"))

	data := &evaluationStateData{
		evaluationData: &evaluationData{
			Evaluator: evaluator,
		},
		ModulePath: addrs.RootModuleInstance,
		Operation:  walkPlan,
	}

	tests := []struct {
		addr   addrs.ResourceInstance
		wantOk bool
	}{
		{resource("c").Instance(addrs.IntKey(0)), true},
		// planned, with the value taken from the change
		{resource("c").Instance(addrs.IntKey(1)), true},
		// no current object, so the value is unknown
		{resource("c").Instance(addrs.IntKey(2)), true},
		// pending deletion, so excluded from the resource's value
		{resource("c").Instance(addrs.IntKey(3)), false},
		// planned, but with no recorded change, so the value is unknown
		{resource("c").Instance(addrs.IntKey(4)), true},
		// not in the state at all
		{resource("c").Instance(addrs.IntKey(5)), false},
		// wrong kind of key for the resource
		{resource("c").Instance(addrs.StringKey("a")), false},
		{resource("f").Instance(addrs.StringKey("a")), true},
		{resource("f").Instance(addrs.StringKey("b")), true},
		{resource("f").Instance(addrs.StringKey("c")), false},
		{resource("s").Instance(addrs.NoKey), true},
		{resource("s").Instance(addrs.IntKey(0)), false},
		// undeclared
		{resource("missing").Instance(addrs.NoKey), false},
	}

	for _, test := range tests {
		t.Run(test.addr.String(), func(t *testing.T) {
			got, ok := data.GetResourceInstance(test.addr)
			if ok != test.wantOk {
				t.Fatalf("wrong ok result %t, want %t", ok, test.wantOk)
			}
			if !ok {
				return
			}

			all, diags := data.GetResource(test.addr.Resource, tfdiags.SourceRange{})
			if diags.HasErrors() {
				t.Fatal(diags.Err())
			}
			want := all
			switch key := test.addr.Key.(type) {
			case addrs.IntKey:
				want = all.Index(cty.NumberIntVal(int64(key)))
			case addrs.StringKey:
				want = all.GetAttr(string(key))
			}
			if !got.RawEquals(want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, want)
			}
		})
	}

	// Instances aren't ordered relative to one another, so another instance's
	// state may not have been upgraded yet. GetResource reports that problem,
	// but it must not affect the value of this instance.
	t.Run("invalid other instance", func(t *testing.T) {
		all, diags := data.GetResource(resource("m"), tfdiags.SourceRange{})
		if !diags.HasErrors() {
			t.Fatal("GetResource did not report the invalid instance")
		}

		got, ok := data.GetResourceInstance(resource("m").Instance(addrs.StringKey("a")))
		if !ok {
			t.Fatal("valid instance was not returned")
		}
		if want := all.GetAttr("a"); !got.RawEquals(want) {
			t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, want)
		}

		// The invalid instance's own problem is left to GetResource.
		if _, ok := data.GetResourceInstance(resource("m").Instance(addrs.StringKey("b"))); ok {
			t.Error("invalid instance was returned")
		}
	})
}

func TestEvaluatorGetModule(t *testing.T) {
	evaluator := evaluatorForModule(states.NewState().SyncWrapper(), plans.NewChanges().SyncWrapper(), nil, nil)
	evaluator.Instances.SetModuleSingle(addrs.RootModuleInstance, addrs.ModuleCall{Name: "mod"})
	evaluator.NamedValues.SetOutputValue(
		addrs.OutputValue{Name: "out"}.Absolute(addrs.ModuleInstance{addrs.ModuleInstanceStep{Name: "mod"}}),
		cty.StringVal("bar").Mark(marks.Sensitive),
	)
	data := &evaluationStateData{
		evaluationData: &evaluationData{
			Evaluator: evaluator,
		},
	}
	scope := evaluator.Scope(data, nil, nil, lang.ExternalFuncs{})
	want := cty.ObjectVal(map[string]cty.Value{"out": cty.StringVal("bar").Mark(marks.Sensitive)})
	got, diags := scope.Data.GetModule(addrs.ModuleCall{
		Name: "mod",
	}, tfdiags.SourceRange{})

	if len(diags) != 0 {
		t.Errorf("unexpected diagnostics %s", spew.Sdump(diags))
	}
	if !got.RawEquals(want) {
		t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestEvaluatorGetModule_cache(t *testing.T) {
	evaluator := evaluatorForModule(states.NewState().SyncWrapper(), plans.NewChanges().SyncWrapper(), nil, nil)
	evaluator.ModuleCallValues = newModuleCallValues()
	evaluator.Instances.SetModuleCount(addrs.RootModuleInstance, addrs.ModuleCall{Name: "mod"}, 2)
	data := &evaluationStateData{
		evaluationData: &evaluationData{
			Evaluator: evaluator,
		},
	}
	scope := evaluator.Scope(data, nil, nil, lang.ExternalFuncs{})
	callAddr := addrs.ModuleCall{Name: "mod"}
	absCallAddr := callAddr.Absolute(addrs.RootModuleInstance)
	outAddr := func(i int) addrs.AbsOutputValue {
		return addrs.OutputValue{Name: "out"}.Absolute(addrs.RootModuleInstance.Child("mod", addrs.IntKey(i)))
	}

	evaluator.NamedValues.SetOutputValue(outAddr(0), cty.StringVal("a"))

	// Only one of the instances has published its output value, so the
	// result must not be cached.
	want := cty.TupleVal([]cty.Value{
		cty.ObjectVal(map[string]cty.Value{"out": cty.StringVal("a").Mark(marks.Sensitive)}),
		cty.ObjectVal(map[string]cty.Value{"out": cty.DynamicVal}),
	})
	got, diags := scope.Data.GetModule(callAddr, tfdiags.SourceRange{})
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics %s", spew.Sdump(diags))
	}
	if !got.RawEquals(want) {
		t.Fatalf("wrong result\ngot:  %#v\nwant: %#v", got, want)
	}
	if _, ok := evaluator.ModuleCallValues.get(absCallAddr); ok {
		t.Fatal("incomplete module call value was cached")
	}

	evaluator.NamedValues.SetOutputValue(outAddr(1), cty.StringVal("b"))

	want = cty.TupleVal([]cty.Value{
		cty.ObjectVal(map[string]cty.Value{"out": cty.StringVal("a").Mark(marks.Sensitive)}),
		cty.ObjectVal(map[string]cty.Value{"out": cty.StringVal("b").Mark(marks.Sensitive)}),
	})
	got, diags = scope.Data.GetModule(callAddr, tfdiags.SourceRange{})
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics %s", spew.Sdump(diags))
	}
	if !got.RawEquals(want) {
		t.Fatalf("wrong result\ngot:  %#v\nwant: %#v", got, want)
	}
	cached, ok := evaluator.ModuleCallValues.get(absCallAddr)
	if !ok {
		t.Fatal("complete module call value was not cached")
	}
	if !cached.RawEquals(want) {
		t.Fatalf("wrong cached value\ngot:  %#v\nwant: %#v", cached, want)
	}

	// Later references must get the cached value.
	sentinel := cty.StringVal("cached")
	evaluator.ModuleCallValues.put(absCallAddr, sentinel)
	got, _ = scope.Data.GetModule(callAddr, tfdiags.SourceRange{})
	if !got.RawEquals(sentinel) {
		t.Fatalf("cached value not used\ngot: %#v", got)
	}
}

func TestEvaluatorGetModule_validateTypedOutputs(t *testing.T) {
	tests := map[string]struct {
		configureModuleCall func(*configs.ModuleCall)
		want                cty.Value
	}{
		"single": {
			want: cty.ObjectVal(map[string]cty.Value{
				"out": cty.UnknownVal(cty.String),
			}),
		},
		"count": {
			configureModuleCall: func(call *configs.ModuleCall) {
				call.Count = hcltest.MockExprLiteral(cty.NumberIntVal(1))
			},
			want: cty.UnknownVal(cty.List(cty.Object(map[string]cty.Type{
				"out": cty.String,
			}))),
		},
		"empty_count": {
			configureModuleCall: func(call *configs.ModuleCall) {
				call.Count = hcltest.MockExprLiteral(cty.NumberIntVal(0))
			},
			want: cty.UnknownVal(cty.List(cty.Object(map[string]cty.Type{
				"out": cty.String,
			}))),
		},
		"for_each": {
			configureModuleCall: func(call *configs.ModuleCall) {
				call.ForEach = hcltest.MockExprLiteral(cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("a"),
				}))
			},
			want: cty.UnknownVal(cty.Map(cty.Object(map[string]cty.Type{
				"out": cty.String,
			}))),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			evaluator := evaluatorForModule(states.NewState().SyncWrapper(), plans.NewChanges().SyncWrapper(), func(out *configs.Output) {
				out.ConstraintType = cty.String
				out.TypeSet = true
			}, test.configureModuleCall)

			data := &evaluationStateData{
				evaluationData: &evaluationData{
					Evaluator: evaluator,
				},
				Operation: walkValidate,
			}
			scope := evaluator.Scope(data, nil, nil, lang.ExternalFuncs{})

			got, diags := scope.Data.GetModule(addrs.ModuleCall{
				Name: "mod",
			}, tfdiags.SourceRange{})

			if len(diags) != 0 {
				t.Errorf("unexpected diagnostics %s", spew.Sdump(diags))
			}
			if !got.RawEquals(test.want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.want)
			}
		})
	}
}

func TestEvaluatorGetModule_validateTypedOutputsWithDynamicTypes(t *testing.T) {
	tests := map[string]struct {
		configureModuleCall func(*configs.ModuleCall)
	}{
		"count": {
			configureModuleCall: func(call *configs.ModuleCall) {
				call.Count = hcltest.MockExprLiteral(cty.NumberIntVal(1))
			},
		},
		"for_each": {
			configureModuleCall: func(call *configs.ModuleCall) {
				call.ForEach = hcltest.MockExprLiteral(cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("a"),
				}))
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			evaluator := evaluatorForModule(states.NewState().SyncWrapper(), plans.NewChanges().SyncWrapper(), func(out *configs.Output) {
				out.ConstraintType = cty.DynamicPseudoType
				out.TypeSet = true
			}, test.configureModuleCall)

			data := &evaluationStateData{
				evaluationData: &evaluationData{
					Evaluator: evaluator,
				},
				Operation: walkValidate,
			}
			scope := evaluator.Scope(data, nil, nil, lang.ExternalFuncs{})

			got, diags := scope.Data.GetModule(addrs.ModuleCall{
				Name: "mod",
			}, tfdiags.SourceRange{})

			if len(diags) != 0 {
				t.Errorf("unexpected diagnostics %s", spew.Sdump(diags))
			}
			if !got.RawEquals(cty.DynamicVal) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, cty.DynamicVal)
			}
		})
	}
}

func TestEvaluatorGetModule_planTypedOutputs(t *testing.T) {
	tests := map[string]struct {
		setupInstances func(*instances.Expander)
		setupOutputs   func(*namedvals.State)
		want           cty.Value
	}{
		"count": {
			setupInstances: func(expander *instances.Expander) {
				expander.SetModuleCount(addrs.RootModuleInstance, addrs.ModuleCall{Name: "mod"}, 2)
			},
			setupOutputs: func(namedValues *namedvals.State) {
				namedValues.SetOutputValue(
					addrs.OutputValue{Name: "out"}.Absolute(addrs.ModuleInstance{
						addrs.ModuleInstanceStep{Name: "mod", InstanceKey: addrs.IntKey(0)},
					}),
					cty.StringVal("first").Mark(marks.Sensitive),
				)
				namedValues.SetOutputValue(
					addrs.OutputValue{Name: "out"}.Absolute(addrs.ModuleInstance{
						addrs.ModuleInstanceStep{Name: "mod", InstanceKey: addrs.IntKey(1)},
					}),
					cty.StringVal("second").Mark(marks.Sensitive),
				)
			},
			want: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{"out": cty.StringVal("first").Mark(marks.Sensitive)}),
				cty.ObjectVal(map[string]cty.Value{"out": cty.StringVal("second").Mark(marks.Sensitive)}),
			}),
		},
		"empty_count": {
			setupInstances: func(expander *instances.Expander) {
				expander.SetModuleCount(addrs.RootModuleInstance, addrs.ModuleCall{Name: "mod"}, 0)
			},
			setupOutputs: func(namedValues *namedvals.State) {
				// explicitly setting no values
			},
			want: cty.ListValEmpty(cty.Object(map[string]cty.Type{"out": cty.String})),
		},
		"for_each": {
			setupInstances: func(expander *instances.Expander) {
				expander.SetModuleForEach(addrs.RootModuleInstance, addrs.ModuleCall{Name: "mod"}, map[string]cty.Value{
					"a": cty.StringVal("a"),
					"b": cty.StringVal("b"),
				})
			},
			setupOutputs: func(namedValues *namedvals.State) {
				namedValues.SetOutputValue(
					addrs.OutputValue{Name: "out"}.Absolute(addrs.ModuleInstance{
						addrs.ModuleInstanceStep{Name: "mod", InstanceKey: addrs.StringKey("a")},
					}),
					cty.StringVal("first").Mark(marks.Sensitive),
				)
				namedValues.SetOutputValue(
					addrs.OutputValue{Name: "out"}.Absolute(addrs.ModuleInstance{
						addrs.ModuleInstanceStep{Name: "mod", InstanceKey: addrs.StringKey("b")},
					}),
					cty.StringVal("second").Mark(marks.Sensitive),
				)
			},
			want: cty.MapVal(map[string]cty.Value{
				"a": cty.ObjectVal(map[string]cty.Value{"out": cty.StringVal("first").Mark(marks.Sensitive)}),
				"b": cty.ObjectVal(map[string]cty.Value{"out": cty.StringVal("second").Mark(marks.Sensitive)}),
			}),
		},
		"empty_for_each": {
			setupInstances: func(expander *instances.Expander) {
				expander.SetModuleForEach(addrs.RootModuleInstance, addrs.ModuleCall{Name: "mod"}, map[string]cty.Value{})
			},
			setupOutputs: func(namedValues *namedvals.State) {
				// no values for empty for_each
			},
			want: cty.MapValEmpty(cty.Object(map[string]cty.Type{"out": cty.String})),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			evaluator := evaluatorForModule(states.NewState().SyncWrapper(), plans.NewChanges().SyncWrapper(), func(out *configs.Output) {
				out.ConstraintType = cty.String
				out.TypeSet = true
			}, nil)

			test.setupInstances(evaluator.Instances)
			test.setupOutputs(evaluator.NamedValues)

			data := &evaluationStateData{
				evaluationData: &evaluationData{
					Evaluator: evaluator,
				},
				Operation: walkPlan,
			}
			scope := evaluator.Scope(data, nil, nil, lang.ExternalFuncs{})

			got, diags := scope.Data.GetModule(addrs.ModuleCall{
				Name: "mod",
			}, tfdiags.SourceRange{})

			if len(diags) != 0 {
				t.Errorf("unexpected diagnostics %s", spew.Sdump(diags))
			}
			if !got.RawEquals(test.want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.want)
			}
		})
	}
}

func TestEvaluatorGetModule_planUntypedOutputsRemainStructural(t *testing.T) {
	tests := map[string]struct {
		setupInstances func(*instances.Expander)
		setupOutputs   func(*namedvals.State)
		want           cty.Value
	}{
		"count": {
			setupInstances: func(expander *instances.Expander) {
				expander.SetModuleCount(addrs.RootModuleInstance, addrs.ModuleCall{Name: "mod"}, 2)
			},
			setupOutputs: func(namedValues *namedvals.State) {
				namedValues.SetOutputValue(
					addrs.OutputValue{Name: "out"}.Absolute(addrs.ModuleInstance{
						addrs.ModuleInstanceStep{Name: "mod", InstanceKey: addrs.IntKey(0)},
					}),
					cty.StringVal("first").Mark(marks.Sensitive),
				)
				namedValues.SetOutputValue(
					addrs.OutputValue{Name: "out"}.Absolute(addrs.ModuleInstance{
						addrs.ModuleInstanceStep{Name: "mod", InstanceKey: addrs.IntKey(1)},
					}),
					cty.StringVal("second").Mark(marks.Sensitive),
				)
			},
			want: cty.TupleVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{"out": cty.StringVal("first").Mark(marks.Sensitive)}),
				cty.ObjectVal(map[string]cty.Value{"out": cty.StringVal("second").Mark(marks.Sensitive)}),
			}),
		},
		"for_each": {
			setupInstances: func(expander *instances.Expander) {
				expander.SetModuleForEach(addrs.RootModuleInstance, addrs.ModuleCall{Name: "mod"}, map[string]cty.Value{
					"a": cty.StringVal("a"),
					"b": cty.StringVal("b"),
				})
			},
			setupOutputs: func(namedValues *namedvals.State) {
				namedValues.SetOutputValue(
					addrs.OutputValue{Name: "out"}.Absolute(addrs.ModuleInstance{
						addrs.ModuleInstanceStep{Name: "mod", InstanceKey: addrs.StringKey("a")},
					}),
					cty.StringVal("first").Mark(marks.Sensitive),
				)
				namedValues.SetOutputValue(
					addrs.OutputValue{Name: "out"}.Absolute(addrs.ModuleInstance{
						addrs.ModuleInstanceStep{Name: "mod", InstanceKey: addrs.StringKey("b")},
					}),
					cty.StringVal("second").Mark(marks.Sensitive),
				)
			},
			want: cty.ObjectVal(map[string]cty.Value{
				"a": cty.ObjectVal(map[string]cty.Value{"out": cty.StringVal("first").Mark(marks.Sensitive)}),
				"b": cty.ObjectVal(map[string]cty.Value{"out": cty.StringVal("second").Mark(marks.Sensitive)}),
			}),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			evaluator := evaluatorForModule(states.NewState().SyncWrapper(), plans.NewChanges().SyncWrapper(), nil, nil)

			test.setupInstances(evaluator.Instances)
			test.setupOutputs(evaluator.NamedValues)

			data := &evaluationStateData{
				evaluationData: &evaluationData{
					Evaluator: evaluator,
				},
				Operation: walkPlan,
			}
			scope := evaluator.Scope(data, nil, nil, lang.ExternalFuncs{})

			got, diags := scope.Data.GetModule(addrs.ModuleCall{
				Name: "mod",
			}, tfdiags.SourceRange{})

			if len(diags) != 0 {
				t.Errorf("unexpected diagnostics %s", spew.Sdump(diags))
			}
			if !got.RawEquals(test.want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.want)
			}
		})
	}
}

func evaluatorForModule(stateSync *states.SyncState, changesSync *plans.ChangesSync, configureOutput func(*configs.Output), configureModuleCall func(*configs.ModuleCall)) *Evaluator {
	moduleCall := &configs.ModuleCall{
		Name: "mod",
	}
	if configureModuleCall != nil {
		configureModuleCall(moduleCall)
	}

	output := &configs.Output{
		Name:           "out",
		Sensitive:      true,
		ConstraintType: cty.DynamicPseudoType,
	}
	if configureOutput != nil {
		configureOutput(output)
	}

	return &Evaluator{
		Meta: &ContextMeta{
			Env: "foo",
		},
		Config: &configs.Config{
			Module: &configs.Module{
				ModuleCalls: map[string]*configs.ModuleCall{
					"mod": moduleCall,
				},
			},
			Children: map[string]*configs.Config{
				"mod": {
					Path: addrs.Module{"module.mod"},
					Module: &configs.Module{
						Outputs: map[string]*configs.Output{
							"out": output,
						},
					},
				},
			},
		},
		State:       stateSync,
		Changes:     changesSync,
		Instances:   instances.NewExpander(nil),
		NamedValues: namedvals.NewState(),
	}
}

// fakeEvaluationData is an implementation of [lang.Data] that answers most
// questions just by returning data directly from the maps stored inside it.
type fakeEvaluationData struct {
	checkBlocks    map[addrs.Check]cty.Value
	countAttrs     map[addrs.CountAttr]cty.Value
	forEachAttrs   map[addrs.ForEachAttr]cty.Value
	inputVariables map[addrs.InputVariable]cty.Value
	localValues    map[addrs.LocalValue]cty.Value
	modules        map[addrs.ModuleCall]cty.Value
	outputValues   map[addrs.OutputValue]cty.Value
	pathAttrs      map[addrs.PathAttr]cty.Value
	resources      map[addrs.Resource]cty.Value
	runBlocks      map[addrs.Run]cty.Value
	terraformAttrs map[addrs.TerraformAttr]cty.Value

	// staticValidateRefs optionally implements [lang.Data.StaticValidateReferences],
	// but can be left as nil to just skip static validation altogether.
	staticValidateRefs func(refs []*addrs.Reference, self addrs.Referenceable, source addrs.Referenceable) tfdiags.Diagnostics
}

var _ lang.Data = (*fakeEvaluationData)(nil)

func fakeEvaluationDataLookup[Addr interface {
	comparable
	addrs.Referenceable
}](addr Addr, _ tfdiags.SourceRange, table map[Addr]cty.Value) (cty.Value, tfdiags.Diagnostics) {
	ret, ok := table[addr]
	if !ok {
		var diags tfdiags.Diagnostics
		diags = diags.Append(fmt.Errorf("fakeEvaluationData does not know about %s", addr))
		return cty.DynamicVal, diags
	}
	return ret, nil
}

// GetCheckBlock implements lang.Data.
func (d *fakeEvaluationData) GetCheckBlock(addr addrs.Check, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return fakeEvaluationDataLookup(addr, rng, d.checkBlocks)
}

// GetCountAttr implements lang.Data.
func (d *fakeEvaluationData) GetCountAttr(addr addrs.CountAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return fakeEvaluationDataLookup(addr, rng, d.countAttrs)
}

// GetForEachAttr implements lang.Data.
func (d *fakeEvaluationData) GetForEachAttr(addr addrs.ForEachAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return fakeEvaluationDataLookup(addr, rng, d.forEachAttrs)
}

// GetInputVariable implements lang.Data.
func (d *fakeEvaluationData) GetInputVariable(addr addrs.InputVariable, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return fakeEvaluationDataLookup(addr, rng, d.inputVariables)
}

// GetLocalValue implements lang.Data.
func (d *fakeEvaluationData) GetLocalValue(addr addrs.LocalValue, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return fakeEvaluationDataLookup(addr, rng, d.localValues)
}

// GetModule implements lang.Data.
func (d *fakeEvaluationData) GetModule(addr addrs.ModuleCall, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return fakeEvaluationDataLookup(addr, rng, d.modules)
}

// GetOutput implements lang.Data.
func (d *fakeEvaluationData) GetOutput(addr addrs.OutputValue, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return fakeEvaluationDataLookup(addr, rng, d.outputValues)
}

// GetPathAttr implements lang.Data.
func (d *fakeEvaluationData) GetPathAttr(addr addrs.PathAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return fakeEvaluationDataLookup(addr, rng, d.pathAttrs)
}

// GetResource implements lang.Data.
func (d *fakeEvaluationData) GetResource(addr addrs.Resource, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return fakeEvaluationDataLookup(addr, rng, d.resources)
}

// GetRunBlock implements lang.Data.
func (d *fakeEvaluationData) GetRunBlock(addr addrs.Run, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return fakeEvaluationDataLookup(addr, rng, d.runBlocks)
}

// GetTerraformAttr implements lang.Data.
func (d *fakeEvaluationData) GetTerraformAttr(addr addrs.TerraformAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return fakeEvaluationDataLookup(addr, rng, d.terraformAttrs)
}

// StaticValidateReferences implements lang.Data.
func (d *fakeEvaluationData) StaticValidateReferences(refs []*addrs.Reference, self addrs.Referenceable, source addrs.Referenceable) tfdiags.Diagnostics {
	if d.staticValidateRefs == nil {
		// By default we just skip static validation
		return nil
	}
	return d.staticValidateRefs(refs, self, source)
}
