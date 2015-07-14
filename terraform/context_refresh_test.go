package terraform

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestContext2Refresh(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "refresh-basic")
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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

func TestContext2Refresh_targeted(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "refresh-targeted")
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		Targets: []string{"aws_instance.me"},
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
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		Targets: []string{"aws_instance.me"},
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
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		Targets: []string{"aws_instance.me[0]"},
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
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		Module: m,
		Hooks:  []Hook{h},
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
						Tainted: []*InstanceState{
							&InstanceState{
								ID: "bar",
							},
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
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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

					Outputs: map[string]string{
						"foo": "foo",
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
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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

func TestContext2Refresh_state(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "refresh-basic")
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.web": &ResourceState{
						Primary: &InstanceState{
							ID: "bar",
						},
					},
				},
			},
		},
	}
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
						Tainted: []*InstanceState{
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
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
	ctx := testContext2(t, &ContextOpts{
		Module:    m,
		Providers: map[string]ResourceProviderFactory{},
	})

	if _, err := ctx.Refresh(); err == nil {
		t.Fatal("should error")
	}
}

func TestContext2Refresh_vars(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "refresh-vars")
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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

func TestContext2Validate(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "validate-good")
	c := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	w, e := c.Validate()
	if len(w) > 0 {
		t.Fatalf("bad: %#v", w)
	}
	if len(e) > 0 {
		t.Fatalf("bad: %s", e)
	}
}
