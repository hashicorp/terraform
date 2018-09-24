package terraform

import (
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/config/hcl2shim"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/providers"
)

func TestContext2Refresh(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "refresh-basic")

	startingState := mustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.web": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
							Attributes: map[string]string{
								"id":  "foo",
								"foo": "bar",
							},
						},
					},
				},
			},
		},
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: startingState,
	})

	schema := p.GetSchemaReturn.ResourceTypes["aws_instance"]
	ty := schema.ImpliedType()
	readState, err := hcl2shim.HCL2ValueFromFlatmap(map[string]string{"id": "foo", "foo": "baz"}, ty)
	if err != nil {
		t.Fatal(err)
	}

	p.ReadResourceFn = nil
	p.ReadResourceResponse = providers.ReadResourceResponse{
		NewState: readState,
	}

	s, diags := ctx.Refresh()
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	if !p.ReadResourceCalled {
		t.Fatal("ReadResource should be called")
	}

	mod := s.RootModule()
	fromState, err := mod.Resources["aws_instance.web"].Instances[addrs.NoKey].Current.Decode(ty)
	if err != nil {
		t.Fatal(err)
	}

	newState, err := schema.CoerceValue(fromState.Value)
	if err != nil {
		t.Fatal(err)
	}

	if !cmp.Equal(readState, newState, valueComparer) {
		t.Fatal(cmp.Diff(readState, newState, valueComparer, equateEmpty))
	}
}

func TestContext2Refresh_dataComputedModuleVar(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "refresh-data-module-var")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	p.ReadResourceFn = nil
	p.ReadResourceResponse = providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal("foo"),
		}),
	}

	p.GetSchemaReturn = &ProviderSchema{
		Provider: &configschema.Block{},
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {
						Type:     cty.String,
						Optional: true,
					},
					"id": {
						Type:     cty.String,
						Computed: true,
					},
				},
			},
		},
		DataSources: map[string]*configschema.Block{
			"aws_data_source": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
		},
	}

	s, diags := ctx.Refresh()
	if diags.HasErrors() {
		t.Fatalf("refresh errors: %s", diags.Err())
	}

	checkStateString(t, s, `
<no state>
`)
}

func TestContext2Refresh_targeted(t *testing.T) {
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		Provider: &configschema.Block{},
		ResourceTypes: map[string]*configschema.Block{
			"aws_elb": {
				Attributes: map[string]*configschema.Attribute{
					"instances": {
						Type:     cty.Set(cty.String),
						Optional: true,
					},
				},
			},
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"vpc_id": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
			"aws_vpc": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
				},
			},
		},
	}

	m := testModule(t, "refresh-targeted")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: mustShimLegacyState(&State{
			Modules: []*ModuleState{
				&ModuleState{
					Path: rootModulePath,
					Resources: map[string]*ResourceState{
						"aws_vpc.metoo":      resourceState("aws_vpc", "vpc-abc123"),
						"aws_instance.notme": resourceState("aws_instance", "i-bcd345"),
						"aws_instance.me":    resourceState("aws_instance", "i-abc123"),
						"aws_elb.meneither":  resourceState("aws_elb", "lb-abc123"),
					},
				},
			},
		}),
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "me",
			),
		},
	})

	refreshedResources := make([]string, 0, 2)
	p.ReadResourceFn = func(req providers.ReadResourceRequest) providers.ReadResourceResponse {
		refreshedResources = append(refreshedResources, req.PriorState.GetAttr("id").AsString())
		return providers.ReadResourceResponse{
			NewState: req.PriorState,
		}
	}

	_, diags := ctx.Refresh()
	if diags.HasErrors() {
		t.Fatalf("refresh errors: %s", diags.Err())
	}

	expected := []string{"vpc-abc123", "i-abc123"}
	if !reflect.DeepEqual(refreshedResources, expected) {
		t.Fatalf("expected: %#v, got: %#v", expected, refreshedResources)
	}
}

func TestContext2Refresh_targetedCount(t *testing.T) {
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		Provider: &configschema.Block{},
		ResourceTypes: map[string]*configschema.Block{
			"aws_elb": {
				Attributes: map[string]*configschema.Attribute{
					"instances": {
						Type:     cty.Set(cty.String),
						Optional: true,
					},
				},
			},
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"vpc_id": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
			"aws_vpc": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
				},
			},
		},
	}

	m := testModule(t, "refresh-targeted-count")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: mustShimLegacyState(&State{
			Modules: []*ModuleState{
				&ModuleState{
					Path: rootModulePath,
					Resources: map[string]*ResourceState{
						"aws_vpc.metoo":      resourceState("aws_vpc", "vpc-abc123"),
						"aws_instance.notme": resourceState("aws_instance", "i-bcd345"),
						"aws_instance.me.0":  resourceState("aws_instance", "i-abc123"),
						"aws_instance.me.1":  resourceState("aws_instance", "i-cde567"),
						"aws_instance.me.2":  resourceState("aws_instance", "i-cde789"),
						"aws_elb.meneither":  resourceState("aws_elb", "lb-abc123"),
					},
				},
			},
		}),
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "me",
			),
		},
	})

	refreshedResources := make([]string, 0, 2)
	p.ReadResourceFn = func(req providers.ReadResourceRequest) providers.ReadResourceResponse {
		refreshedResources = append(refreshedResources, req.PriorState.GetAttr("id").AsString())
		return providers.ReadResourceResponse{
			NewState: req.PriorState,
		}
	}

	_, diags := ctx.Refresh()
	if diags.HasErrors() {
		t.Fatalf("refresh errors: %s", diags.Err())
	}

	// Target didn't specify index, so we should get all our instances
	expected := []string{
		"vpc-abc123",
		"i-abc123",
		"i-cde567",
		"i-cde789",
	}
	sort.Strings(expected)
	sort.Strings(refreshedResources)
	if !reflect.DeepEqual(refreshedResources, expected) {
		t.Fatalf("wrong result\ngot:  %#v\nwant: %#v", refreshedResources, expected)
	}
}

func TestContext2Refresh_targetedCountIndex(t *testing.T) {
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		Provider: &configschema.Block{},
		ResourceTypes: map[string]*configschema.Block{
			"aws_elb": {
				Attributes: map[string]*configschema.Attribute{
					"instances": {
						Type:     cty.Set(cty.String),
						Optional: true,
					},
				},
			},
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"vpc_id": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
			"aws_vpc": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
				},
			},
		},
	}

	m := testModule(t, "refresh-targeted-count")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: mustShimLegacyState(&State{
			Modules: []*ModuleState{
				&ModuleState{
					Path: rootModulePath,
					Resources: map[string]*ResourceState{
						"aws_vpc.metoo":      resourceState("aws_vpc", "vpc-abc123"),
						"aws_instance.notme": resourceState("aws_instance", "i-bcd345"),
						"aws_instance.me.0":  resourceState("aws_instance", "i-abc123"),
						"aws_instance.me.1":  resourceState("aws_instance", "i-cde567"),
						"aws_instance.me.2":  resourceState("aws_instance", "i-cde789"),
						"aws_elb.meneither":  resourceState("aws_elb", "lb-abc123"),
					},
				},
			},
		}),
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.ResourceInstance(
				addrs.ManagedResourceMode, "aws_instance", "me", addrs.IntKey(0),
			),
		},
	})

	refreshedResources := make([]string, 0, 2)
	p.ReadResourceFn = func(req providers.ReadResourceRequest) providers.ReadResourceResponse {
		refreshedResources = append(refreshedResources, req.PriorState.GetAttr("id").AsString())
		return providers.ReadResourceResponse{
			NewState: req.PriorState,
		}
	}

	_, diags := ctx.Refresh()
	if diags.HasErrors() {
		t.Fatalf("refresh errors: %s", diags.Err())
	}

	expected := []string{"vpc-abc123", "i-abc123"}
	if !reflect.DeepEqual(refreshedResources, expected) {
		t.Fatalf("wrong result\ngot:  %#v\nwant: %#v", refreshedResources, expected)
	}
}

func TestContext2Refresh_moduleComputedVar(t *testing.T) {
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		Provider: &configschema.Block{},
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"value": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
		},
	}

	m := testModule(t, "refresh-module-computed-var")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	// This was failing (see GH-2188) at some point, so this test just
	// verifies that the failure goes away.
	if _, diags := ctx.Refresh(); diags.HasErrors() {
		t.Fatalf("refresh errs: %s", diags.Err())
	}
}

func TestContext2Refresh_delete(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "refresh-basic")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: mustShimLegacyState(&State{
			Modules: []*ModuleState{
				&ModuleState{
					Path: rootModulePath,
					Resources: map[string]*ResourceState{
						"aws_instance.web": &ResourceState{
							Type: "aws_instance",
							Primary: &InstanceState{
								ID: "foo",
							},
						},
					},
				},
			},
		}),
	})

	p.ReadResourceFn = nil
	p.ReadResourceResponse = providers.ReadResourceResponse{
		NewState: cty.NullVal(p.GetSchemaReturn.ResourceTypes["aws_instance"].ImpliedType()),
	}

	s, diags := ctx.Refresh()
	if diags.HasErrors() {
		t.Fatalf("refresh errors: %s", diags.Err())
	}

	mod := s.RootModule()
	if len(mod.Resources) > 0 {
		t.Fatal("resources should be empty")
	}
}

func TestContext2Refresh_ignoreUncreated(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "refresh-basic")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: nil,
	})

	p.ReadResourceFn = nil
	p.ReadResourceResponse = providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal("foo"),
		}),
	}

	_, diags := ctx.Refresh()
	if diags.HasErrors() {
		t.Fatalf("refresh errors: %s", diags.Err())
	}
	if p.ReadResourceCalled {
		t.Fatal("refresh should not be called")
	}
}

func TestContext2Refresh_hook(t *testing.T) {
	h := new(MockHook)
	p := testProvider("aws")
	m := testModule(t, "refresh-basic")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Hooks:  []Hook{h},
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: mustShimLegacyState(&State{
			Modules: []*ModuleState{
				&ModuleState{
					Path: rootModulePath,
					Resources: map[string]*ResourceState{
						"aws_instance.web": &ResourceState{
							Type: "aws_instance",
							Primary: &InstanceState{
								ID: "foo",
							},
						},
					},
				},
			},
		}),
	})

	if _, diags := ctx.Refresh(); diags.HasErrors() {
		t.Fatalf("refresh errs: %s", diags.Err())
	}
	if !h.PreRefreshCalled {
		t.Fatal("should be called")
	}
	if !h.PostRefreshCalled {
		t.Fatal("should be called")
	}
}

func TestContext2Refresh_modules(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "refresh-modules")
	state := mustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.web": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID:      "bar",
							Tainted: true,
						},
					},
				},
			},

			&ModuleState{
				Path: []string{"root", "child"},
				Resources: map[string]*ResourceState{
					"aws_instance.web": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "baz",
						},
					},
				},
			},
		},
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: state,
	})

	p.ReadResourceFn = func(req providers.ReadResourceRequest) providers.ReadResourceResponse {
		if !req.PriorState.GetAttr("id").RawEquals(cty.StringVal("baz")) {
			return providers.ReadResourceResponse{
				NewState: req.PriorState,
			}
		}

		new, _ := cty.Transform(req.PriorState, func(path cty.Path, v cty.Value) (cty.Value, error) {
			if len(path) == 1 && path[0].(cty.GetAttrStep).Name == "id" {
				return cty.StringVal("new"), nil
			}
			return v, nil
		})
		return providers.ReadResourceResponse{
			NewState: new,
		}
	}

	s, diags := ctx.Refresh()
	if diags.HasErrors() {
		t.Fatalf("refresh errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(s.String())
	expected := strings.TrimSpace(testContextRefreshModuleStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Refresh_moduleInputComputedOutput(t *testing.T) {
	m := testModule(t, "refresh-module-input-computed-output")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	p.GetSchemaReturn = &ProviderSchema{
		Provider: &configschema.Block{},
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {
						Type:     cty.String,
						Optional: true,
					},
					"compute": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Refresh(); diags.HasErrors() {
		t.Fatalf("refresh errs: %s", diags.Err())
	}
}

func TestContext2Refresh_moduleVarModule(t *testing.T) {
	m := testModule(t, "refresh-module-var-module")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Refresh(); diags.HasErrors() {
		t.Fatalf("refresh errs: %s", diags.Err())
	}
}

// GH-70
func TestContext2Refresh_noState(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "refresh-no-state")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	p.ReadResourceFn = nil
	p.ReadResourceResponse = providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal("foo"),
		}),
	}

	if _, diags := ctx.Refresh(); diags.HasErrors() {
		t.Fatalf("refresh errs: %s", diags.Err())
	}
}

func TestContext2Refresh_output(t *testing.T) {
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		Provider: &configschema.Block{},
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"foo": {
						Type:     cty.String,
						Computed: true,
					},
				},
			},
		},
	}

	m := testModule(t, "refresh-output")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: mustShimLegacyState(&State{
			Modules: []*ModuleState{
				&ModuleState{
					Path: rootModulePath,
					Resources: map[string]*ResourceState{
						"aws_instance.web": &ResourceState{
							Type: "aws_instance",
							Primary: &InstanceState{
								ID: "foo",
								Attributes: map[string]string{
									"id":  "foo",
									"foo": "bar",
								},
							},
						},
					},

					Outputs: map[string]*OutputState{
						"foo": &OutputState{
							Type:      "string",
							Sensitive: false,
							Value:     "foo",
						},
					},
				},
			},
		}),
	})

	s, diags := ctx.Refresh()
	if diags.HasErrors() {
		t.Fatalf("refresh errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(s.String())
	expected := strings.TrimSpace(testContextRefreshOutputStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%q\n\nwant:\n%q", actual, expected)
	}
}

func TestContext2Refresh_outputPartial(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "refresh-output-partial")

	// Refresh creates a partial plan for any instances that don't have
	// remote objects yet, to get stub values for interpolation. Therefore
	// we need to make DiffFn available to let that complete.
	p.DiffFn = testDiffFn

	p.GetSchemaReturn = &ProviderSchema{
		Provider: &configschema.Block{},
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {
						Type:     cty.String,
						Computed: true,
					},
				},
			},
		},
	}

	p.ReadResourceFn = nil
	p.ReadResourceResponse = providers.ReadResourceResponse{
		NewState: cty.NullVal(p.GetSchemaReturn.ResourceTypes["aws_instance"].ImpliedType()),
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: mustShimLegacyState(&State{
			Modules: []*ModuleState{
				&ModuleState{
					Path: rootModulePath,
					Resources: map[string]*ResourceState{
						"aws_instance.foo": &ResourceState{
							Type: "aws_instance",
							Primary: &InstanceState{
								ID: "foo",
							},
						},
					},
					Outputs: map[string]*OutputState{},
				},
			},
		}),
	})

	s, diags := ctx.Refresh()
	if diags.HasErrors() {
		t.Fatalf("refresh errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(s.String())
	expected := strings.TrimSpace(testContextRefreshOutputPartialStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Refresh_stateBasic(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "refresh-basic")
	state := mustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.web": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
						},
					},
				},
			},
		},
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: state,
	})

	schema := p.GetSchemaReturn.ResourceTypes["aws_instance"]
	ty := schema.ImpliedType()

	readStateVal, err := schema.CoerceValue(cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("foo"),
	}))
	if err != nil {
		t.Fatal(err)
	}

	p.ReadResourceFn = nil
	p.ReadResourceResponse = providers.ReadResourceResponse{
		NewState: readStateVal,
	}

	s, diags := ctx.Refresh()
	if diags.HasErrors() {
		t.Fatalf("refresh errors: %s", diags.Err())
	}

	if !p.ReadResourceCalled {
		t.Fatal("read resource should be called")
	}

	mod := s.RootModule()
	newState, err := mod.Resources["aws_instance.web"].Instances[addrs.NoKey].Current.Decode(ty)
	if err != nil {
		t.Fatal(err)
	}

	if !cmp.Equal(readStateVal, newState.Value, valueComparer, equateEmpty) {
		t.Fatal(cmp.Diff(readStateVal, newState.Value, valueComparer, equateEmpty))
	}
}

func TestContext2Refresh_dataOrphan(t *testing.T) {
	p := testProvider("null")
	state := mustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"data.null_data_source.bar": &ResourceState{
						Type: "null_data_source",
						Primary: &InstanceState{
							ID: "foo",
						},
						Provider: "provider.null",
					},
				},
			},
		},
	})
	ctx := testContext2(t, &ContextOpts{
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"null": testProviderFuncFixed(p),
			},
		),
		State: state,
	})

	s, diags := ctx.Refresh()
	if diags.HasErrors() {
		t.Fatalf("refresh errors: %s", diags.Err())
	}

	checkStateString(t, s, `<no state>`)
}

func TestContext2Refresh_dataState(t *testing.T) {
	m := testModule(t, "refresh-data-resource-basic")

	state := mustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				// Intentionally no resources since data resources are
				// supposed to refresh themselves even if they didn't
				// already exist.
				Resources: map[string]*ResourceState{},
			},
		},
	})

	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"inputs": {
				Type:     cty.Map(cty.String),
				Optional: true,
			},
		},
	}

	p := testProvider("null")
	p.GetSchemaReturn = &ProviderSchema{
		Provider: &configschema.Block{},
		DataSources: map[string]*configschema.Block{
			"null_data_source": schema,
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"null": testProviderFuncFixed(p),
			},
		),
		State: state,
	})

	var readStateVal cty.Value

	p.ReadDataSourceFn = func(req providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
		m := req.Config.AsValueMap()
		m["inputs"] = cty.MapVal(map[string]cty.Value{"test": cty.StringVal("yes")})
		readStateVal = cty.ObjectVal(m)

		return providers.ReadDataSourceResponse{
			State: readStateVal,
		}

		// FIXME: should the "outputs" value here be added to the reutnred state?
		// Attributes: map[string]*ResourceAttrDiff{
		// 	"inputs.#": {
		// 		Old:  "0",
		// 		New:  "1",
		// 		Type: DiffAttrInput,
		// 	},
		// 	"inputs.test": {
		// 		Old:  "",
		// 		New:  "yes",
		// 		Type: DiffAttrInput,
		// 	},
		// 	"outputs.#": {
		// 		Old:         "",
		// 		New:         "",
		// 		NewComputed: true,
		// 		Type:        DiffAttrOutput,
		// 	},
		// },
	}

	s, diags := ctx.Refresh()
	if diags.HasErrors() {
		t.Fatalf("refresh errors: %s", diags.Err())
	}

	if !p.ReadDataSourceCalled {
		t.Fatal("ReadDataSource should have been called")
	}

	// mod := s.RootModule()
	// if got := mod.Resources["data.null_data_source.testing"].Primary.ID; got != "-" {
	// 	t.Fatalf("resource id is %q; want %s", got, "-")
	// }
	// if !reflect.DeepEqual(mod.Resources["data.null_data_source.testing"].Primary, p.ReadDataApplyReturn) {
	// 	t.Fatalf("bad: %#v", mod.Resources)
	// }

	mod := s.RootModule()

	newState, err := mod.Resources["data.null_data_source.testing"].Instances[addrs.NoKey].Current.Decode(schema.ImpliedType())
	if err != nil {
		t.Fatal(err)
	}

	if !cmp.Equal(readStateVal, newState.Value, valueComparer, equateEmpty) {
		t.Fatal(cmp.Diff(readStateVal, newState.Value, valueComparer, equateEmpty))
	}
}

func TestContext2Refresh_dataStateRefData(t *testing.T) {
	p := testProvider("null")
	p.GetSchemaReturn = &ProviderSchema{
		Provider: &configschema.Block{},
		DataSources: map[string]*configschema.Block{
			"null_data_source": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"foo": {
						Type:     cty.String,
						Optional: true,
					},
					"bar": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
		},
	}

	m := testModule(t, "refresh-data-ref-data")
	state := mustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				// Intentionally no resources since data resources are
				// supposed to refresh themselves even if they didn't
				// already exist.
				Resources: map[string]*ResourceState{},
			},
		},
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"null": testProviderFuncFixed(p),
			},
		),
		State: state,
	})

	p.ReadDataSourceFn = func(req providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
		// add the required id
		m := req.Config.AsValueMap()
		m["id"] = cty.StringVal("foo")

		return providers.ReadDataSourceResponse{
			State: cty.ObjectVal(m),
		}
	}

	s, diags := ctx.Refresh()
	if diags.HasErrors() {
		t.Fatalf("refresh errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(s.String())
	expected := strings.TrimSpace(testTerraformRefreshDataRefDataStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Refresh_tainted(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "refresh-basic")
	state := mustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.web": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID:      "bar",
							Tainted: true,
						},
					},
				},
			},
		},
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: state,
	})
	p.ReadResourceFn = func(req providers.ReadResourceRequest) providers.ReadResourceResponse {
		// add the required id
		m := req.PriorState.AsValueMap()
		m["id"] = cty.StringVal("foo")

		return providers.ReadResourceResponse{
			NewState: cty.ObjectVal(m),
		}
	}

	s, diags := ctx.Refresh()
	if diags.HasErrors() {
		t.Fatalf("refresh errors: %s", diags.Err())
	}
	if !p.ReadResourceCalled {
		t.Fatal("ReadResource was not called; should have been")
	}

	actual := strings.TrimSpace(s.String())
	expected := strings.TrimSpace(testContextRefreshTaintedStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

// Doing a Refresh (or any operation really, but Refresh usually
// happens first) with a config with an unknown provider should result in
// an error. The key bug this found was that this wasn't happening if
// Providers was _empty_.
func TestContext2Refresh_unknownProvider(t *testing.T) {
	m := testModule(t, "refresh-unknown-provider")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	_, diags := NewContext(&ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{},
		),
		State: mustShimLegacyState(&State{
			Modules: []*ModuleState{
				&ModuleState{
					Path: rootModulePath,
					Resources: map[string]*ResourceState{
						"aws_instance.web": &ResourceState{
							Type: "aws_instance",
							Primary: &InstanceState{
								ID: "foo",
							},
						},
					},
				},
			},
		}),
	})

	if !diags.HasErrors() {
		t.Fatal("successfully created context; want error")
	}

	if !regexp.MustCompile(`provider ".+" is not available`).MatchString(diags.Err().Error()) {
		t.Fatalf("wrong error: %s", diags.Err())
	}
}

func TestContext2Refresh_vars(t *testing.T) {
	p := testProvider("aws")

	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"ami": {
				Type:     cty.String,
				Optional: true,
			},
			"id": {
				Type:     cty.String,
				Computed: true,
			},
		},
	}

	p.GetSchemaReturn = &ProviderSchema{
		Provider:      &configschema.Block{},
		ResourceTypes: map[string]*configschema.Block{"aws_instance": schema},
	}

	m := testModule(t, "refresh-vars")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: mustShimLegacyState(&State{

			Modules: []*ModuleState{
				&ModuleState{
					Path: rootModulePath,
					Resources: map[string]*ResourceState{
						"aws_instance.web": &ResourceState{
							Type: "aws_instance",
							Primary: &InstanceState{
								ID: "foo",
							},
						},
					},
				},
			},
		}),
	})

	readStateVal, err := schema.CoerceValue(cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("foo"),
	}))
	if err != nil {
		t.Fatal(err)
	}

	p.ReadResourceFn = nil
	p.ReadResourceResponse = providers.ReadResourceResponse{
		NewState: readStateVal,
	}

	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		return providers.PlanResourceChangeResponse{
			PlannedState: req.ProposedNewState,
		}
	}

	s, diags := ctx.Refresh()
	if diags.HasErrors() {
		t.Fatalf("refresh errors: %s", diags.Err())
	}

	if !p.ReadResourceCalled {
		t.Fatal("read resource should be called")
	}

	mod := s.RootModule()

	newState, err := mod.Resources["aws_instance.web"].Instances[addrs.NoKey].Current.Decode(schema.ImpliedType())
	if err != nil {
		t.Fatal(err)
	}

	if !cmp.Equal(readStateVal, newState.Value, valueComparer, equateEmpty) {
		t.Fatal(cmp.Diff(readStateVal, newState.Value, valueComparer, equateEmpty))
	}

	for _, r := range mod.Resources {
		if r.Addr.Type == "" {
			t.Fatalf("no type: %#v", r)
		}
	}
}

func TestContext2Refresh_orphanModule(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "refresh-module-orphan")

	// Create a custom refresh function to track the order they were visited
	var order []string
	var orderLock sync.Mutex
	p.ReadResourceFn = func(req providers.ReadResourceRequest) providers.ReadResourceResponse {
		orderLock.Lock()
		defer orderLock.Unlock()

		order = append(order, req.PriorState.GetAttr("id").AsString())
		return providers.ReadResourceResponse{
			NewState: req.PriorState,
		}
	}

	state := mustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.foo": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "i-abc123",
							Attributes: map[string]string{
								"id":           "i-abc123",
								"childid":      "i-bcd234",
								"grandchildid": "i-cde345",
							},
						},
						Dependencies: []string{
							"module.child",
							"module.child",
						},
						Provider: "provider.aws",
					},
				},
			},
			&ModuleState{
				Path: append(rootModulePath, "child"),
				Resources: map[string]*ResourceState{
					"aws_instance.bar": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "i-bcd234",
							Attributes: map[string]string{
								"id":           "i-bcd234",
								"grandchildid": "i-cde345",
							},
						},
						Dependencies: []string{
							"module.grandchild",
						},
						Provider: "provider.aws",
					},
				},
				Outputs: map[string]*OutputState{
					"id": &OutputState{
						Value: "i-bcd234",
						Type:  "string",
					},
					"grandchild_id": &OutputState{
						Value: "i-cde345",
						Type:  "string",
					},
				},
			},
			&ModuleState{
				Path: append(rootModulePath, "child", "grandchild"),
				Resources: map[string]*ResourceState{
					"aws_instance.baz": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "i-cde345",
							Attributes: map[string]string{
								"id": "i-cde345",
							},
						},
						Provider: "provider.aws",
					},
				},
				Outputs: map[string]*OutputState{
					"id": &OutputState{
						Value: "i-cde345",
						Type:  "string",
					},
				},
			},
		},
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: state,
	})

	testCheckDeadlock(t, func() {
		_, err := ctx.Refresh()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		// TODO: handle order properly for orphaned modules / resources
		// expected := []string{"i-abc123", "i-bcd234", "i-cde345"}
		// if !reflect.DeepEqual(order, expected) {
		// 	t.Fatalf("expected: %#v, got: %#v", expected, order)
		// }
	})
}

func TestContext2Validate(t *testing.T) {
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		Provider: &configschema.Block{},
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {
						Type:     cty.String,
						Optional: true,
					},
					"num": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
		},
	}

	m := testModule(t, "validate-good")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := c.Validate()
	if len(diags) != 0 {
		t.Fatalf("unexpected error: %#v", diags.ErrWithWarnings())
	}
}

// TestContext2Refresh_noDiffHookOnScaleOut tests to make sure that
// pre/post-diff hooks are not called when running EvalDiff on scale-out nodes
// (nodes with no state). The effect here is to make sure that the diffs -
// which only exist for interpolation of parallel resources or data sources -
// do not end up being counted in the UI.
func TestContext2Refresh_noDiffHookOnScaleOut(t *testing.T) {
	h := new(MockHook)
	p := testProvider("aws")
	m := testModule(t, "refresh-resource-scale-inout")

	// Refresh creates a partial plan for any instances that don't have
	// remote objects yet, to get stub values for interpolation. Therefore
	// we need to make DiffFn available to let that complete.
	p.DiffFn = testDiffFn

	state := mustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.foo.0": &ResourceState{
						Type: "aws_instance",
						Deposed: []*InstanceState{
							&InstanceState{
								ID: "foo",
								Attributes: map[string]string{
									"id": "foo",
								},
							},
						},
					},
					"aws_instance.foo.1": &ResourceState{
						Type: "aws_instance",
						Deposed: []*InstanceState{
							&InstanceState{
								ID: "bar",
								Attributes: map[string]string{
									"id": "foo",
								},
							},
						},
					},
				},
			},
		},
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Hooks:  []Hook{h},
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: state,
	})

	_, diags := ctx.Refresh()
	if diags.HasErrors() {
		t.Fatalf("refresh errors: %s", diags.Err())
	}
	if h.PreDiffCalled {
		t.Fatal("PreDiff should not have been called")
	}
	if h.PostDiffCalled {
		t.Fatal("PostDiff should not have been called")
	}
}

func TestContext2Refresh_updateProviderInState(t *testing.T) {
	m := testModule(t, "update-resource-provider")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	p.ApplyFn = testApplyFn

	s := mustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.bar": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
							Attributes: map[string]string{
								"id": "foo",
							},
						},
						Provider: "provider.aws.baz",
					},
				},
			},
		},
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	expected := strings.TrimSpace(`
aws_instance.bar:
  ID = foo
  provider = provider.aws.foo`)

	state, diags := ctx.Refresh()
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	actual := state.String()
	if actual != expected {
		t.Fatalf("expected:\n%s\n\ngot:\n%s", expected, actual)
	}
}
