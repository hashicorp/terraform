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

func TestContextValidate_providerConfig_bad(t *testing.T) {
	config := testConfig(t, "validate-bad-pc")
	p := testProvider("aws")
	c := testContext(t, &ContextOpts{
		Config: config,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	p.ValidateReturnErrors = []error{fmt.Errorf("bad")}

	w, e := c.Validate()
	if len(w) > 0 {
		t.Fatalf("bad: %#v", w)
	}
	if len(e) == 0 {
		t.Fatalf("bad: %#v", e)
	}
}

func TestContextValidate_providerConfig_good(t *testing.T) {
	config := testConfig(t, "validate-bad-pc")
	p := testProvider("aws")
	c := testContext(t, &ContextOpts{
		Config: config,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	w, e := c.Validate()
	if len(w) > 0 {
		t.Fatalf("bad: %#v", w)
	}
	if len(e) > 0 {
		t.Fatalf("bad: %#v", e)
	}
}

func TestContextValidate_resourceConfig_bad(t *testing.T) {
	config := testConfig(t, "validate-bad-rc")
	p := testProvider("aws")
	c := testContext(t, &ContextOpts{
		Config: config,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	p.ValidateResourceReturnErrors = []error{fmt.Errorf("bad")}

	w, e := c.Validate()
	if len(w) > 0 {
		t.Fatalf("bad: %#v", w)
	}
	if len(e) == 0 {
		t.Fatalf("bad: %#v", e)
	}
}

func TestContextValidate_resourceConfig_good(t *testing.T) {
	config := testConfig(t, "validate-bad-rc")
	p := testProvider("aws")
	c := testContext(t, &ContextOpts{
		Config: config,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	w, e := c.Validate()
	if len(w) > 0 {
		t.Fatalf("bad: %#v", w)
	}
	if len(e) > 0 {
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

func TestContextApply(t *testing.T) {
	c := testConfig(t, "apply-good")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext(t, &ContextOpts{
		Config: c,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	if _, err := ctx.Plan(nil); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if len(state.Resources) < 2 {
		t.Fatalf("bad: %#v", state.Resources)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextApply_Minimal(t *testing.T) {
	c := testConfig(t, "apply-minimal")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext(t, &ContextOpts{
		Config: c,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	if _, err := ctx.Plan(nil); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyMinimalStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextApply_cancel(t *testing.T) {
	stopped := false

	c := testConfig(t, "apply-cancel")
	p := testProvider("aws")
	ctx := testContext(t, &ContextOpts{
		Config: c,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	p.ApplyFn = func(*ResourceState, *ResourceDiff) (*ResourceState, error) {
		if !stopped {
			stopped = true
			go ctx.Stop()

			for {
				if ctx.sh.Stopped() {
					break
				}
			}
		}

		return &ResourceState{
			ID: "foo",
			Attributes: map[string]string{
				"num": "2",
			},
		}, nil
	}
	p.DiffFn = func(*ResourceState, *ResourceConfig) (*ResourceDiff, error) {
		return &ResourceDiff{
			Attributes: map[string]*ResourceAttrDiff{
				"num": &ResourceAttrDiff{
					New: "bar",
				},
			},
		}, nil
	}

	if _, err := ctx.Plan(nil); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Start the Apply in a goroutine
	stateCh := make(chan *State)
	go func() {
		state, err := ctx.Apply()
		if err != nil {
			panic(err)
		}

		stateCh <- state
	}()

	state := <-stateCh

	if len(state.Resources) != 1 {
		t.Fatalf("bad: %#v", state.Resources)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyCancelStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextApply_compute(t *testing.T) {
	c := testConfig(t, "apply-compute")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext(t, &ContextOpts{
		Config: c,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	if _, err := ctx.Plan(nil); err != nil {
		t.Fatalf("err: %s", err)
	}

	ctx.variables = map[string]string{"value": "1"}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyComputeStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextApply_destroy(t *testing.T) {
	c := testConfig(t, "apply-destroy")
	h := new(HookRecordApplyOrder)
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext(t, &ContextOpts{
		Config: c,
		Hooks:  []Hook{h},
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	// First plan and apply a create operation
	if _, err := ctx.Plan(nil); err != nil {
		t.Fatalf("err: %s", err)
	}

	if _, err := ctx.Apply(); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Next, plan and apply a destroy operation
	if _, err := ctx.Plan(&PlanOpts{Destroy: true}); err != nil {
		t.Fatalf("err: %s", err)
	}

	h.Active = true

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Test that things were destroyed
	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyDestroyStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}

	// Test that things were destroyed _in the right order_
	expected2 := []string{"aws_instance.bar", "aws_instance.foo"}
	actual2 := h.IDs
	if !reflect.DeepEqual(actual2, expected2) {
		t.Fatalf("bad: %#v", actual2)
	}
}

func TestContextApply_destroyOrphan(t *testing.T) {
	c := testConfig(t, "apply-error")
	p := testProvider("aws")
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

	p.ApplyFn = func(s *ResourceState, d *ResourceDiff) (*ResourceState, error) {
		if d.Destroy {
			return nil, nil
		}

		result := s.MergeDiff(d)
		result.ID = "foo"
		return result, nil
	}
	p.DiffFn = func(*ResourceState, *ResourceConfig) (*ResourceDiff, error) {
		return &ResourceDiff{
			Attributes: map[string]*ResourceAttrDiff{
				"num": &ResourceAttrDiff{
					New: "bar",
				},
			},
		}, nil
	}

	if _, err := ctx.Plan(nil); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if _, ok := state.Resources["aws_instance.baz"]; ok {
		t.Fatalf("bad: %#v", state.Resources)
	}
}

func TestContextApply_error(t *testing.T) {
	errored := false

	c := testConfig(t, "apply-error")
	p := testProvider("aws")
	ctx := testContext(t, &ContextOpts{
		Config: c,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	p.ApplyFn = func(*ResourceState, *ResourceDiff) (*ResourceState, error) {
		if errored {
			return nil, fmt.Errorf("error")
		}
		errored = true

		return &ResourceState{
			ID: "foo",
			Attributes: map[string]string{
				"num": "2",
			},
		}, nil
	}
	p.DiffFn = func(*ResourceState, *ResourceConfig) (*ResourceDiff, error) {
		return &ResourceDiff{
			Attributes: map[string]*ResourceAttrDiff{
				"num": &ResourceAttrDiff{
					New: "bar",
				},
			},
		}, nil
	}

	if _, err := ctx.Plan(nil); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err == nil {
		t.Fatal("should have error")
	}

	if len(state.Resources) != 1 {
		t.Fatalf("bad: %#v", state.Resources)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyErrorStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextApply_errorPartial(t *testing.T) {
	errored := false

	c := testConfig(t, "apply-error")
	p := testProvider("aws")
	s := &State{
		Resources: map[string]*ResourceState{
			"aws_instance.bar": &ResourceState{
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

	p.ApplyFn = func(*ResourceState, *ResourceDiff) (*ResourceState, error) {
		if errored {
			return nil, fmt.Errorf("error")
		}
		errored = true

		return &ResourceState{
			ID: "foo",
			Attributes: map[string]string{
				"num": "2",
			},
		}, nil
	}
	p.DiffFn = func(*ResourceState, *ResourceConfig) (*ResourceDiff, error) {
		return &ResourceDiff{
			Attributes: map[string]*ResourceAttrDiff{
				"num": &ResourceAttrDiff{
					New: "bar",
				},
			},
		}, nil
	}

	if _, err := ctx.Plan(nil); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err == nil {
		t.Fatal("should have error")
	}

	if len(state.Resources) != 2 {
		t.Fatalf("bad: %#v", state.Resources)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyErrorPartialStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextApply_hook(t *testing.T) {
	c := testConfig(t, "apply-good")
	h := new(MockHook)
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext(t, &ContextOpts{
		Config: c,
		Hooks:  []Hook{h},
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	if _, err := ctx.Plan(nil); err != nil {
		t.Fatalf("err: %s", err)
	}

	if _, err := ctx.Apply(); err != nil {
		t.Fatalf("err: %s", err)
	}

	if !h.PreApplyCalled {
		t.Fatal("should be called")
	}
	if !h.PostApplyCalled {
		t.Fatal("should be called")
	}
}

func TestContextApply_output(t *testing.T) {
	c := testConfig(t, "apply-output")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext(t, &ContextOpts{
		Config: c,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	if _, err := ctx.Plan(nil); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyOutputStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextApply_outputMulti(t *testing.T) {
	c := testConfig(t, "apply-output-multi")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext(t, &ContextOpts{
		Config: c,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	if _, err := ctx.Plan(nil); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyOutputMultiStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextApply_outputMultiIndex(t *testing.T) {
	c := testConfig(t, "apply-output-multi-index")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext(t, &ContextOpts{
		Config: c,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	if _, err := ctx.Plan(nil); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyOutputMultiIndexStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextApply_unknownAttribute(t *testing.T) {
	c := testConfig(t, "apply-unknown")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext(t, &ContextOpts{
		Config: c,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	if _, err := ctx.Plan(nil); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err == nil {
		t.Fatal("should error")
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyUnknownAttrStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextApply_vars(t *testing.T) {
	c := testConfig(t, "apply-vars")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext(t, &ContextOpts{
		Config: c,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		Variables: map[string]string{
			"foo": "bar",
		},
	})

	if _, err := ctx.Plan(nil); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyVarsStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
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

func TestContextPlan_minimal(t *testing.T) {
	c := testConfig(t, "plan-empty")
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

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanEmptyStr)
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
		State: &State{
			Resources: map[string]*ResourceState{
				"aws_instance.foo": &ResourceState{
					ID:   "bar",
					Type: "aws_instance",
				},
			},
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

func TestContextPlan_count(t *testing.T) {
	c := testConfig(t, "plan-count")
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

	if len(plan.Diff.Resources) < 6 {
		t.Fatalf("bad: %#v", plan.Diff.Resources)
	}

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanCountStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContextPlan_countDecreaseToOne(t *testing.T) {
	c := testConfig(t, "plan-count-dec")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	s := &State{
		Resources: map[string]*ResourceState{
			"aws_instance.foo.0": &ResourceState{
				ID:   "bar",
				Type: "aws_instance",
				Attributes: map[string]string{
					"foo":  "foo",
					"type": "aws_instance",
				},
			},
			"aws_instance.foo.1": &ResourceState{
				ID:   "bar",
				Type: "aws_instance",
			},
			"aws_instance.foo.2": &ResourceState{
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
	expected := strings.TrimSpace(testTerraformPlanCountDecreaseStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContextPlan_countIncreaseFromOne(t *testing.T) {
	c := testConfig(t, "plan-count-inc")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	s := &State{
		Resources: map[string]*ResourceState{
			"aws_instance.foo": &ResourceState{
				ID:   "bar",
				Type: "aws_instance",
				Attributes: map[string]string{
					"foo":  "foo",
					"type": "aws_instance",
				},
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
	expected := strings.TrimSpace(testTerraformPlanCountIncreaseStr)
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
		State: &State{
			Resources: map[string]*ResourceState{
				"aws_instance.web": &ResourceState{
					ID:   "foo",
					Type: "aws_instance",
				},
			},
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
	if p.RefreshState.ID != "foo" {
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

func TestContextRefresh_ignoreUncreated(t *testing.T) {
	p := testProvider("aws")
	c := testConfig(t, "refresh-basic")
	ctx := testContext(t, &ContextOpts{
		Config: c,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		State: nil,
	})

	p.RefreshFn = nil
	p.RefreshReturn = &ResourceState{
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
		State: &State{
			Resources: map[string]*ResourceState{
				"aws_instance.web": &ResourceState{
					ID:   "foo",
					Type: "aws_instance",
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

func TestContextRefresh_vars(t *testing.T) {
	p := testProvider("aws")
	c := testConfig(t, "refresh-vars")
	ctx := testContext(t, &ContextOpts{
		Config: c,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		State: &State{
			Resources: map[string]*ResourceState{
				"aws_instance.web": &ResourceState{
					ID:   "foo",
					Type: "aws_instance",
				},
			},
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
	if p.RefreshState.ID != "foo" {
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

func testContext(t *testing.T, opts *ContextOpts) *Context {
	return NewContext(opts)
}

func testApplyFn(
	s *ResourceState,
	d *ResourceDiff) (*ResourceState, error) {
	if d.Destroy {
		return nil, nil
	}

	id := "foo"
	if idAttr, ok := d.Attributes["id"]; ok && !idAttr.NewComputed {
		id = idAttr.New
	}

	result := &ResourceState{
		ID: id,
	}

	if d != nil {
		result = result.MergeDiff(d)
	}

	if depAttr, ok := d.Attributes["dep"]; ok {
		result.Dependencies = []ResourceDependency{
			ResourceDependency{
				ID: depAttr.New,
			},
		}
	}

	return result, nil
}

func testDiffFn(
	s *ResourceState,
	c *ResourceConfig) (*ResourceDiff, error) {
	var diff ResourceDiff
	diff.Attributes = make(map[string]*ResourceAttrDiff)

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

	for k, v := range diff.Attributes {
		if v.NewComputed {
			continue
		}

		old, ok := s.Attributes[k]
		if !ok {
			continue
		}
		if old == v.New {
			delete(diff.Attributes, k)
		}
	}

	if !diff.Empty() {
		diff.Attributes["type"] = &ResourceAttrDiff{
			Old: "",
			New: s.Type,
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
