package terraform

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestContextValidate(t *testing.T) {
	config := testConfig(t, "validate-good")
	c := testContext(t, &ContextOpts{
		Config: config,
	})

	w, e := c.Validate()
	if len(w) > 0 {
		t.Fatalf("bad: %#v", w)
	}
	if len(e) > 0 {
		t.Fatalf("bad: %#v", e)
	}
}

func TestContextValidate_badVar(t *testing.T) {
	config := testConfig(t, "validate-bad-var")
	c := testContext(t, &ContextOpts{
		Config: config,
	})

	w, e := c.Validate()
	if len(w) > 0 {
		t.Fatalf("bad: %#v", w)
	}
	if len(e) == 0 {
		t.Fatalf("bad: %#v", e)
	}
}

func TestContextValidate_requiredVar(t *testing.T) {
	config := testConfig(t, "validate-required-var")
	c := testContext(t, &ContextOpts{
		Config: config,
	})

	w, e := c.Validate()
	if len(w) > 0 {
		t.Fatalf("bad: %#v", w)
	}
	if len(e) == 0 {
		t.Fatalf("bad: %#v", e)
	}
}

func TestContextPlan(t *testing.T) {
	c := testConfig(t, "plan-good")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext(t, &ContextOpts{
		Config: c,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	plan, err := ctx.Plan(nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if len(plan.Diff.Resources) < 2 {
		t.Fatalf("bad: %#v", plan.Diff.Resources)
	}

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContextPlan_nil(t *testing.T) {
	c := testConfig(t, "plan-nil")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext(t, &ContextOpts{
		Config: c,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	plan, err := ctx.Plan(nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if len(plan.Diff.Resources) != 0 {
		t.Fatalf("bad: %#v", plan.Diff.Resources)
	}
}

func TestContextPlan_computed(t *testing.T) {
	c := testConfig(t, "plan-computed")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext(t, &ContextOpts{
		Config: c,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	plan, err := ctx.Plan(nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if len(plan.Diff.Resources) < 2 {
		t.Fatalf("bad: %#v", plan.Diff.Resources)
	}

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanComputedStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContextPlan_destroy(t *testing.T) {
	c := testConfig(t, "plan-destroy")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	s := &State{
		Resources: map[string]*ResourceState{
			"aws_instance.one": &ResourceState{
				ID:   "bar",
				Type: "aws_instance",
			},
			"aws_instance.two": &ResourceState{
				ID:   "baz",
				Type: "aws_instance",
			},
		},
	}
	ctx := testContext(t, &ContextOpts{
		Config: c,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		State: s,
	})

	plan, err := ctx.Plan(&PlanOpts{Destroy: true})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if len(plan.Diff.Resources) != 2 {
		t.Fatalf("bad: %#v", plan.Diff.Resources)
	}

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanDestroyStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContextPlan_hook(t *testing.T) {
	c := testConfig(t, "plan-good")
	h := new(MockHook)
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext(t, &ContextOpts{
		Config: c,
		Hooks:  []Hook{h},
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	_, err := ctx.Plan(nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !h.PreDiffCalled {
		t.Fatal("should be called")
	}
	if !h.PostDiffCalled {
		t.Fatal("should be called")
	}
}

func TestContextPlan_orphan(t *testing.T) {
	c := testConfig(t, "plan-orphan")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	s := &State{
		Resources: map[string]*ResourceState{
			"aws_instance.baz": &ResourceState{
				ID:   "bar",
				Type: "aws_instance",
			},
		},
	}
	ctx := testContext(t, &ContextOpts{
		Config: c,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		State: s,
	})

	plan, err := ctx.Plan(nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanOrphanStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContextPlan_state(t *testing.T) {
	c := testConfig(t, "plan-good")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	s := &State{
		Resources: map[string]*ResourceState{
			"aws_instance.foo": &ResourceState{
				ID: "bar",
			},
		},
	}
	ctx := testContext(t, &ContextOpts{
		Config: c,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		State: s,
	})

	plan, err := ctx.Plan(nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if len(plan.Diff.Resources) < 2 {
		t.Fatalf("bad: %#v", plan.Diff.Resources)
	}

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanStateStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContextRefresh(t *testing.T) {
	p := testProvider("aws")
	c := testConfig(t, "refresh-basic")
	ctx := testContext(t, &ContextOpts{
		Config: c,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	p.RefreshFn = nil
	p.RefreshReturn = &ResourceState{
		ID: "foo",
	}

	s, err := ctx.Refresh()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if !p.RefreshCalled {
		t.Fatal("refresh should be called")
	}
	if p.RefreshState.ID != "" {
		t.Fatalf("bad: %#v", p.RefreshState)
	}
	if !reflect.DeepEqual(s.Resources["aws_instance.web"], p.RefreshReturn) {
		t.Fatalf("bad: %#v", s.Resources["aws_instance.web"])
	}

	for _, r := range s.Resources {
		if r.Type == "" {
			t.Fatalf("no type: %#v", r)
		}
	}
}

func TestContextRefresh_hook(t *testing.T) {
	h := new(MockHook)
	p := testProvider("aws")
	c := testConfig(t, "refresh-basic")
	ctx := testContext(t, &ContextOpts{
		Config: c,
		Hooks:  []Hook{h},
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	if _, err := ctx.Refresh(); err != nil {
		t.Fatalf("err: %s", err)
	}
	if !h.PreRefreshCalled {
		t.Fatal("should be called")
	}
	if h.PreRefreshState.Type != "aws_instance" {
		t.Fatalf("bad: %#v", h.PreRefreshState)
	}
	if !h.PostRefreshCalled {
		t.Fatal("should be called")
	}
	if h.PostRefreshState.Type != "aws_instance" {
		t.Fatalf("bad: %#v", h.PostRefreshState)
	}
}

func TestContextRefresh_state(t *testing.T) {
	p := testProvider("aws")
	c := testConfig(t, "refresh-basic")
	state := &State{
		Resources: map[string]*ResourceState{
			"aws_instance.web": &ResourceState{
				ID: "bar",
			},
		},
	}
	ctx := testContext(t, &ContextOpts{
		Config: c,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		State: state,
	})

	p.RefreshFn = nil
	p.RefreshReturn = &ResourceState{
		ID: "foo",
	}

	s, err := ctx.Refresh()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if !p.RefreshCalled {
		t.Fatal("refresh should be called")
	}
	if !reflect.DeepEqual(p.RefreshState, state.Resources["aws_instance.web"]) {
		t.Fatalf("bad: %#v", p.RefreshState)
	}
	if !reflect.DeepEqual(s.Resources["aws_instance.web"], p.RefreshReturn) {
		t.Fatalf("bad: %#v", s.Resources)
	}
}

func testContext(t *testing.T, opts *ContextOpts) *Context {
	return NewContext(opts)
}

func testDiffFn(
	s *ResourceState,
	c *ResourceConfig) (*ResourceDiff, error) {
	var diff ResourceDiff
	diff.Attributes = make(map[string]*ResourceAttrDiff)
	diff.Attributes["type"] = &ResourceAttrDiff{
		Old: "",
		New: s.Type,
	}

	for k, v := range c.Raw {
		if _, ok := v.(string); !ok {
			continue
		}

		if k == "nil" {
			return nil, nil
		}

		// This key is used for other purposes
		if k == "compute_value" {
			continue
		}

		if k == "compute" {
			attrDiff := &ResourceAttrDiff{
				Old:         "",
				New:         "",
				NewComputed: true,
			}

			if cv, ok := c.Config["compute_value"]; ok {
				if cv.(string) == "1" {
					attrDiff.NewComputed = false
					attrDiff.New = fmt.Sprintf("computed_%s", v.(string))
				}
			}

			diff.Attributes[v.(string)] = attrDiff
			continue
		}

		// If this key is not computed, then look it up in the
		// cleaned config.
		found := false
		for _, ck := range c.ComputedKeys {
			if ck == k {
				found = true
				break
			}
		}
		if !found {
			v = c.Config[k]
		}

		attrDiff := &ResourceAttrDiff{
			Old: "",
			New: v.(string),
		}

		diff.Attributes[k] = attrDiff
	}

	for _, k := range c.ComputedKeys {
		diff.Attributes[k] = &ResourceAttrDiff{
			Old:         "",
			NewComputed: true,
		}
	}

	return &diff, nil
}

func testProvider(prefix string) *MockResourceProvider {
	p := new(MockResourceProvider)
	p.RefreshFn = func(s *ResourceState) (*ResourceState, error) {
		return s, nil
	}
	p.ResourcesReturn = []ResourceType{
		ResourceType{
			Name: fmt.Sprintf("%s_instance", prefix),
		},
	}

	return p
}
