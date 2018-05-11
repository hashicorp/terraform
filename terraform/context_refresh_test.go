package terraform

import (
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform/addrs"
)

func TestContext2Refresh(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "refresh-basic")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: &State{
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
		},
	})

	p.RefreshFn = nil
	p.RefreshReturn = &InstanceState{
		ID: "foo",
	}

	s, err := ctx.Refresh()
	mod := s.RootModule()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if !p.RefreshCalled {
		t.Fatal("refresh should be called")
	}
	if p.RefreshState.ID != "foo" {
		t.Fatalf("bad: %#v", p.RefreshState)
	}
	if !reflect.DeepEqual(mod.Resources["aws_instance.web"].Primary, p.RefreshReturn) {
		t.Fatalf("bad: %#v %#v", mod.Resources["aws_instance.web"], p.RefreshReturn)
	}

	for _, r := range mod.Resources {
		if r.Type == "" {
			t.Fatalf("no type: %#v", r)
		}
	}
}

func TestContext2Refresh_dataComputedModuleVar(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "refresh-data-module-var")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	p.RefreshFn = nil
	p.RefreshReturn = &InstanceState{
		ID: "foo",
	}

	s, err := ctx.Refresh()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	checkStateString(t, s, `
<no state>
module.child:
  <no state>`)
}

func TestContext2Refresh_targeted(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "refresh-targeted")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: &State{
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
		},
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "me",
			),
		},
	})

	refreshedResources := make([]string, 0, 2)
	p.RefreshFn = func(i *InstanceInfo, is *InstanceState) (*InstanceState, error) {
		refreshedResources = append(refreshedResources, i.Id)
		return is, nil
	}

	_, err := ctx.Refresh()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := []string{"aws_vpc.metoo", "aws_instance.me"}
	if !reflect.DeepEqual(refreshedResources, expected) {
		t.Fatalf("expected: %#v, got: %#v", expected, refreshedResources)
	}
}

func TestContext2Refresh_targetedCount(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "refresh-targeted-count")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: &State{
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
		},
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "me",
			),
		},
	})

	refreshedResources := make([]string, 0, 2)
	p.RefreshFn = func(i *InstanceInfo, is *InstanceState) (*InstanceState, error) {
		refreshedResources = append(refreshedResources, i.Id)
		return is, nil
	}

	_, err := ctx.Refresh()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Target didn't specify index, so we should get all our instances
	expected := []string{
		"aws_vpc.metoo",
		"aws_instance.me.0",
		"aws_instance.me.1",
		"aws_instance.me.2",
	}
	sort.Strings(expected)
	sort.Strings(refreshedResources)
	if !reflect.DeepEqual(refreshedResources, expected) {
		t.Fatalf("expected: %#v, got: %#v", expected, refreshedResources)
	}
}

func TestContext2Refresh_targetedCountIndex(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "refresh-targeted-count")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: &State{
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
		},
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.ResourceInstance(
				addrs.ManagedResourceMode, "aws_instance", "me", addrs.IntKey(0),
			),
		},
	})

	refreshedResources := make([]string, 0, 2)
	p.RefreshFn = func(i *InstanceInfo, is *InstanceState) (*InstanceState, error) {
		refreshedResources = append(refreshedResources, i.Id)
		return is, nil
	}

	_, err := ctx.Refresh()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := []string{"aws_vpc.metoo", "aws_instance.me.0"}
	if !reflect.DeepEqual(refreshedResources, expected) {
		t.Fatalf("expected: %#v, got: %#v", expected, refreshedResources)
	}
}

func TestContext2Refresh_moduleComputedVar(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "refresh-module-computed-var")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	// This was failing (see GH-2188) at some point, so this test just
	// verifies that the failure goes away.
	if _, err := ctx.Refresh(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestContext2Refresh_delete(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "refresh-basic")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: &State{
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
		},
	})

	p.RefreshFn = nil
	p.RefreshReturn = nil

	s, err := ctx.Refresh()
	if err != nil {
		t.Fatalf("err: %s", err)
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: nil,
	})

	p.RefreshFn = nil
	p.RefreshReturn = &InstanceState{
		ID: "foo",
	}

	_, err := ctx.Refresh()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if p.RefreshCalled {
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: &State{
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
		},
	})

	if _, err := ctx.Refresh(); err != nil {
		t.Fatalf("err: %s", err)
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
	state := &State{
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
	}
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: state,
	})

	p.RefreshFn = func(info *InstanceInfo, s *InstanceState) (*InstanceState, error) {
		if s.ID != "baz" {
			return s, nil
		}

		s.ID = "new"
		return s, nil
	}

	s, err := ctx.Refresh()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(s.String())
	expected := strings.TrimSpace(testContextRefreshModuleStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s\n\n%s", actual, expected)
	}
}

func TestContext2Refresh_moduleInputComputedOutput(t *testing.T) {
	m := testModule(t, "refresh-module-input-computed-output")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, err := ctx.Refresh(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestContext2Refresh_moduleVarModule(t *testing.T) {
	m := testModule(t, "refresh-module-var-module")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, err := ctx.Refresh(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

// GH-70
func TestContext2Refresh_noState(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "refresh-no-state")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	p.RefreshFn = nil
	p.RefreshReturn = &InstanceState{
		ID: "foo",
	}

	if _, err := ctx.Refresh(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestContext2Refresh_output(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "refresh-output")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: &State{
			Modules: []*ModuleState{
				&ModuleState{
					Path: rootModulePath,
					Resources: map[string]*ResourceState{
						"aws_instance.web": &ResourceState{
							Type: "aws_instance",
							Primary: &InstanceState{
								ID: "foo",
								Attributes: map[string]string{
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
		},
	})

	p.RefreshFn = func(info *InstanceInfo, s *InstanceState) (*InstanceState, error) {
		return s, nil
	}

	s, err := ctx.Refresh()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(s.String())
	expected := strings.TrimSpace(testContextRefreshOutputStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s\n\n%s", actual, expected)
	}
}

func TestContext2Refresh_outputPartial(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "refresh-output-partial")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: &State{
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
		},
	})

	p.RefreshFn = nil
	p.RefreshReturn = nil

	s, err := ctx.Refresh()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(s.String())
	expected := strings.TrimSpace(testContextRefreshOutputPartialStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s\n\n%s", actual, expected)
	}
}

func TestContext2Refresh_stateBasic(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "refresh-basic")
	state := &State{
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
	}
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: state,
	})

	p.RefreshFn = nil
	p.RefreshReturn = &InstanceState{
		ID: "foo",
	}

	s, err := ctx.Refresh()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	originalMod := state.RootModule()
	mod := s.RootModule()
	if !p.RefreshCalled {
		t.Fatal("refresh should be called")
	}
	if !reflect.DeepEqual(p.RefreshState, originalMod.Resources["aws_instance.web"].Primary) {
		t.Fatalf(
			"bad:\n\n%#v\n\n%#v",
			p.RefreshState,
			originalMod.Resources["aws_instance.web"].Primary)
	}
	if !reflect.DeepEqual(mod.Resources["aws_instance.web"].Primary, p.RefreshReturn) {
		t.Fatalf("bad: %#v", mod.Resources)
	}
}

func TestContext2Refresh_dataOrphan(t *testing.T) {
	p := testProvider("null")
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"data.null_data_source.bar": &ResourceState{
						Type: "null_data_source",
						Primary: &InstanceState{
							ID: "foo",
						},
					},
				},
			},
		},
	}
	ctx := testContext2(t, &ContextOpts{
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"null": testProviderFuncFixed(p),
			},
		),
		State: state,
	})

	s, err := ctx.Refresh()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	checkStateString(t, s, `<no state>`)
}

func TestContext2Refresh_dataState(t *testing.T) {
	p := testProvider("null")
	m := testModule(t, "refresh-data-resource-basic")
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				// Intentionally no resources since data resources are
				// supposed to refresh themselves even if they didn't
				// already exist.
				Resources: map[string]*ResourceState{},
			},
		},
	}
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"null": testProviderFuncFixed(p),
			},
		),
		State: state,
	})

	p.ReadDataDiffFn = nil
	p.ReadDataDiffReturn = &InstanceDiff{
		Attributes: map[string]*ResourceAttrDiff{
			"inputs.#": {
				Old:  "0",
				New:  "1",
				Type: DiffAttrInput,
			},
			"inputs.test": {
				Old:  "",
				New:  "yes",
				Type: DiffAttrInput,
			},
			"outputs.#": {
				Old:         "",
				New:         "",
				NewComputed: true,
				Type:        DiffAttrOutput,
			},
		},
	}

	p.ReadDataApplyFn = nil
	p.ReadDataApplyReturn = &InstanceState{
		ID: "-",
	}

	s, err := ctx.Refresh()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !p.ReadDataDiffCalled {
		t.Fatal("ReadDataDiff should have been called")
	}
	if !p.ReadDataApplyCalled {
		t.Fatal("ReadDataApply should have been called")
	}

	mod := s.RootModule()
	if got := mod.Resources["data.null_data_source.testing"].Primary.ID; got != "-" {
		t.Fatalf("resource id is %q; want %s", got, "-")
	}
	if !reflect.DeepEqual(mod.Resources["data.null_data_source.testing"].Primary, p.ReadDataApplyReturn) {
		t.Fatalf("bad: %#v", mod.Resources)
	}
}

func TestContext2Refresh_dataStateRefData(t *testing.T) {
	p := testProvider("null")
	m := testModule(t, "refresh-data-ref-data")
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				// Intentionally no resources since data resources are
				// supposed to refresh themselves even if they didn't
				// already exist.
				Resources: map[string]*ResourceState{},
			},
		},
	}
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"null": testProviderFuncFixed(p),
			},
		),
		State: state,
	})

	p.ReadDataDiffFn = testDataDiffFn
	p.ReadDataApplyFn = testDataApplyFn

	s, err := ctx.Refresh()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(s.String())
	expected := strings.TrimSpace(testTerraformRefreshDataRefDataStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s\n\n%s", actual, expected)
	}
}

func TestContext2Refresh_tainted(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "refresh-basic")
	state := &State{
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
	}
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: state,
	})

	p.RefreshFn = nil
	p.RefreshReturn = &InstanceState{
		ID:      "foo",
		Tainted: true,
	}

	s, err := ctx.Refresh()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if !p.RefreshCalled {
		t.Fatal("refresh should be called")
	}

	actual := strings.TrimSpace(s.String())
	expected := strings.TrimSpace(testContextRefreshTaintedStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s\n\n%s", actual, expected)
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{},
		),
		Shadow: true,
		State: &State{
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
		},
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
	m := testModule(t, "refresh-vars")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: &State{

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
		},
	})

	p.RefreshFn = nil
	p.RefreshReturn = &InstanceState{
		ID: "foo",
	}

	s, err := ctx.Refresh()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	mod := s.RootModule()
	if !p.RefreshCalled {
		t.Fatal("refresh should be called")
	}
	if p.RefreshState.ID != "foo" {
		t.Fatalf("bad: %#v", p.RefreshState)
	}
	if !reflect.DeepEqual(mod.Resources["aws_instance.web"].Primary, p.RefreshReturn) {
		t.Fatalf("bad: %#v", mod.Resources["aws_instance.web"])
	}

	for _, r := range mod.Resources {
		if r.Type == "" {
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
	p.RefreshFn = func(
		info *InstanceInfo,
		is *InstanceState) (*InstanceState, error) {
		orderLock.Lock()
		defer orderLock.Unlock()

		order = append(order, is.ID)
		return is, nil
	}

	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.foo": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "i-abc123",
							Attributes: map[string]string{
								"childid":      "i-bcd234",
								"grandchildid": "i-cde345",
							},
						},
						Dependencies: []string{
							"module.child",
							"module.child",
						},
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
								"grandchildid": "i-cde345",
							},
						},
						Dependencies: []string{
							"module.grandchild",
						},
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
						},
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
	}
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
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
	m := testModule(t, "validate-good")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
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
	p.RefreshFn = nil

	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.foo.0": &ResourceState{
						Type: "aws_instance",
						Deposed: []*InstanceState{
							&InstanceState{
								ID: "foo",
							},
						},
					},
					"aws_instance.foo.1": &ResourceState{
						Type: "aws_instance",
						Deposed: []*InstanceState{
							&InstanceState{
								ID: "bar",
							},
						},
					},
				},
			},
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Hooks:  []Hook{h},
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: state,
	})

	_, err := ctx.Refresh()
	if err != nil {
		t.Fatalf("bad: %s", err)
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

	s := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.bar": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
						Provider: "provider.aws.baz",
					},
				},
			},
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	expected := strings.TrimSpace(`
aws_instance.bar:
  ID = foo
  provider = provider.aws.foo`)

	state, err := ctx.Refresh()
	if err != nil {
		t.Fatal(err)
	}

	actual := state.String()
	if actual != expected {
		t.Fatalf("expected:\n%s\n\ngot:\n%s", expected, actual)
	}
}
