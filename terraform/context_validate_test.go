package terraform

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"
	"github.com/zclconf/go-cty/cty"
)

func TestContext2Validate_badCount(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "validate-bad-count")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := c.Validate()
	if !diags.HasErrors() {
		t.Fatalf("bad: %#v", diags)
	}
}

func TestContext2Validate_badVar(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "validate-bad-var")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := c.Validate()
	if !diags.HasErrors() {
		t.Fatalf("bad: %#v", diags)
	}
}

func TestContext2Validate_varMapOverrideOld(t *testing.T) {
	m := testModule(t, "validate-module-pc-vars")
	p := testProvider("aws")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Variables: InputValues{
			"foo.foo": &InputValue{
				Value:      cty.StringVal("bar"),
				SourceType: ValueFromCaller,
			},
		},
	})

	diags := c.Validate()
	if !diags.HasErrors() {
		t.Fatalf("bad: %#v", diags)
	}
}

func TestContext2Validate_varNoDefaultExplicitType(t *testing.T) {
	m := testModule(t, "validate-var-no-default-explicit-type")
	c := testContext2(t, &ContextOpts{
		Config: m,
	})

	diags := c.Validate()
	if !diags.HasErrors() {
		t.Fatalf("bad: %#v", diags)
	}
}

func TestContext2Validate_computedVar(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "validate-computed-var")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws":  testProviderFuncFixed(p),
				"test": testProviderFuncFixed(testProvider("test")),
			},
		),
	})

	p.ValidateFn = func(c *ResourceConfig) ([]string, []error) {
		if !c.IsComputed("value") {
			return nil, []error{fmt.Errorf("value isn't computed")}
		}

		return nil, c.CheckSet([]string{"value"})
	}

	p.ConfigureFn = func(c *ResourceConfig) error {
		return fmt.Errorf("Configure should not be called for provider")
	}

	diags := c.Validate()
	if diags.HasErrors() {
		t.Fatalf("bad: %#v", diags)
	}
}

// Test that validate allows through computed counts. We do this and allow
// them to fail during "plan" since we can't know if the computed values
// can be realized during a plan.
func TestContext2Validate_countComputed(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "validate-count-computed")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := c.Validate()
	if diags.HasErrors() {
		t.Fatalf("bad: %#v", diags)
	}
}

func TestContext2Validate_countNegative(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "validate-count-negative")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := c.Validate()
	if !diags.HasErrors() {
		t.Fatalf("bad: %#v", diags)
	}
}

func TestContext2Validate_countVariable(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "apply-count-variable")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := c.Validate()
	if diags.HasErrors() {
		t.Fatalf("bad: %#v", diags)
	}
}

func TestContext2Validate_countVariableNoDefault(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "validate-count-variable")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := c.Validate()
	if !diags.HasErrors() {
		t.Fatalf("bad: %#v", diags)
	}
}

/*
TODO: What should we do here?
func TestContext2Validate_cycle(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "validate-cycle")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := c.Validate()
	if !diags.HasErrors() {
		t.Fatalf("bad: %#v", diags)
	}
}
*/

func TestContext2Validate_moduleBadOutput(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "validate-bad-module-output")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := c.Validate()
	if !diags.HasErrors() {
		t.Fatalf("bad: %#v", diags)
	}
}

func TestContext2Validate_moduleGood(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "validate-good-module")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := c.Validate()
	if diags.HasErrors() {
		t.Fatalf("bad: %#v", diags)
	}
}

func TestContext2Validate_moduleBadResource(t *testing.T) {
	m := testModule(t, "validate-module-bad-rc")
	p := testProvider("aws")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	p.ValidateResourceReturnErrors = []error{fmt.Errorf("bad")}

	diags := c.Validate()
	if !diags.HasErrors() {
		t.Fatalf("bad: %#v", diags)
	}
}

func TestContext2Validate_moduleDepsShouldNotCycle(t *testing.T) {
	m := testModule(t, "validate-module-deps-cycle")
	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := ctx.Validate()
	if diags.HasErrors() {
		t.Fatalf("bad: %#v", diags)
	}
}

func TestContext2Validate_moduleProviderInheritOrphan(t *testing.T) {
	m := testModule(t, "validate-module-pc-inherit-orphan")
	p := testProvider("aws")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: &State{
			Modules: []*ModuleState{
				&ModuleState{
					Path: []string{"root", "child"},
					Resources: map[string]*ResourceState{
						"aws_instance.bar": &ResourceState{
							Type: "aws_instance",
							Primary: &InstanceState{
								ID: "bar",
							},
						},
					},
				},
			},
		},
	})

	p.ValidateFn = func(c *ResourceConfig) ([]string, []error) {
		v, ok := c.Get("set")
		if !ok {
			return nil, []error{fmt.Errorf("not set")}
		}
		if v != "bar" {
			return nil, []error{fmt.Errorf("bad: %#v", v)}
		}

		return nil, nil
	}

	diags := c.Validate()
	if diags.HasErrors() {
		t.Fatalf("bad: %#v", diags)
	}
}

func TestContext2Validate_moduleProviderVar(t *testing.T) {
	m := testModule(t, "validate-module-pc-vars")
	p := testProvider("aws")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Variables: InputValues{
			"provider_var": &InputValue{
				Value:      cty.StringVal("bar"),
				SourceType: ValueFromCaller,
			},
		},
	})

	p.ValidateFn = func(c *ResourceConfig) ([]string, []error) {
		return nil, c.CheckSet([]string{"foo"})
	}

	diags := c.Validate()
	if diags.HasErrors() {
		t.Fatalf("bad: %#v", diags)
	}
}

func TestContext2Validate_moduleProviderInheritUnused(t *testing.T) {
	m := testModule(t, "validate-module-pc-inherit-unused")
	p := testProvider("aws")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	p.ValidateFn = func(c *ResourceConfig) ([]string, []error) {
		return nil, c.CheckSet([]string{"foo"})
	}

	diags := c.Validate()
	if diags.HasErrors() {
		t.Fatalf("bad: %#v", diags)
	}
}

func TestContext2Validate_orphans(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "validate-good")
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
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: state,
	})

	p.ValidateResourceFn = func(
		t string, c *ResourceConfig) ([]string, []error) {
		return nil, c.CheckSet([]string{"foo"})
	}

	diags := c.Validate()
	if diags.HasErrors() {
		t.Fatalf("bad: %#v", diags)
	}
}

func TestContext2Validate_providerConfig_bad(t *testing.T) {
	m := testModule(t, "validate-bad-pc")
	p := testProvider("aws")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	p.ValidateReturnErrors = []error{fmt.Errorf("bad")}

	diags := c.Validate()
	if len(diags) != 1 {
		t.Fatalf("wrong number of diagnostics %d; want %d", len(diags), 1)
	}
	if !strings.Contains(diags.Err().Error(), "bad") {
		t.Fatalf("bad: %s", diags.Err().Error())
	}
}

func TestContext2Validate_providerConfig_badEmpty(t *testing.T) {
	m := testModule(t, "validate-bad-pc-empty")
	p := testProvider("aws")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	p.ValidateReturnErrors = []error{fmt.Errorf("bad")}

	diags := c.Validate()
	if !diags.HasErrors() {
		t.Fatalf("bad: %#v", diags)
	}
}

func TestContext2Validate_providerConfig_good(t *testing.T) {
	m := testModule(t, "validate-bad-pc")
	p := testProvider("aws")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := c.Validate()
	if diags.HasErrors() {
		t.Fatalf("bad: %#v", diags)
	}
}

func TestContext2Validate_provisionerConfig_bad(t *testing.T) {
	m := testModule(t, "validate-bad-prov-conf")
	p := testProvider("aws")
	pr := testProvisioner()
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ResourceProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	pr.ValidateReturnErrors = []error{fmt.Errorf("bad")}

	diags := c.Validate()
	if !diags.HasErrors() {
		t.Fatalf("bad: %#v", diags)
	}
}

func TestContext2Validate_provisionerConfig_good(t *testing.T) {
	m := testModule(t, "validate-bad-prov-conf")
	p := testProvider("aws")
	pr := testProvisioner()
	pr.ValidateFn = func(c *ResourceConfig) ([]string, []error) {
		if c == nil {
			t.Fatalf("missing resource config for provisioner")
		}
		return nil, c.CheckSet([]string{"command"})
	}
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ResourceProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	diags := c.Validate()
	if diags.HasErrors() {
		t.Fatalf("bad: %#v", diags)
	}
}

func TestContext2Validate_requiredVar(t *testing.T) {
	m := testModule(t, "validate-required-var")
	p := testProvider("aws")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := c.Validate()
	if !diags.HasErrors() {
		t.Fatalf("bad: %#v", diags)
	}
}

func TestContext2Validate_resourceConfig_bad(t *testing.T) {
	m := testModule(t, "validate-bad-rc")
	p := testProvider("aws")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	p.ValidateResourceReturnErrors = []error{fmt.Errorf("bad")}

	diags := c.Validate()
	if !diags.HasErrors() {
		t.Fatalf("bad: %#v", diags)
	}
}

func TestContext2Validate_resourceConfig_good(t *testing.T) {
	m := testModule(t, "validate-bad-rc")
	p := testProvider("aws")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := c.Validate()
	if diags.HasErrors() {
		t.Fatalf("bad: %#v", diags)
	}
}

func TestContext2Validate_resourceNameSymbol(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "validate-resource-name-symbol")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := c.Validate()
	if !diags.HasErrors() {
		t.Fatalf("bad: %#v", diags)
	}
}

func TestContext2Validate_selfRef(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "validate-self-ref")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := c.Validate()
	if !diags.HasErrors() {
		t.Fatalf("bad: %#v", diags)
	}
}

func TestContext2Validate_selfRefMulti(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "validate-self-ref-multi")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := c.Validate()
	if !diags.HasErrors() {
		t.Fatalf("bad: %#v", diags)
	}
}

func TestContext2Validate_selfRefMultiAll(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "validate-self-ref-multi-all")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := c.Validate()
	if !diags.HasErrors() {
		t.Fatalf("bad: %#v", diags)
	}
}

func TestContext2Validate_tainted(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "validate-good")
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.foo": &ResourceState{
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
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: state,
	})

	p.ValidateResourceFn = func(
		t string, c *ResourceConfig) ([]string, []error) {
		return nil, c.CheckSet([]string{"foo"})
	}

	diags := c.Validate()
	if diags.HasErrors() {
		t.Fatalf("bad: %#v", diags)
	}
}

func TestContext2Validate_targetedDestroy(t *testing.T) {
	m := testModule(t, "validate-targeted")
	p := testProvider("aws")
	pr := testProvisioner()
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ResourceProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
		State: &State{
			Modules: []*ModuleState{
				&ModuleState{
					Path: rootModulePath,
					Resources: map[string]*ResourceState{
						"aws_instance.foo": resourceState("aws_instance", "i-bcd345"),
						"aws_instance.bar": resourceState("aws_instance", "i-abc123"),
					},
				},
			},
		},
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
		},
		Destroy: true,
	})

	diags := ctx.Validate()
	if diags.HasErrors() {
		t.Fatalf("bad: %#v", diags)
	}
}

func TestContext2Validate_varRefFilled(t *testing.T) {
	m := testModule(t, "validate-variable-ref")
	p := testProvider("aws")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Variables: InputValues{
			"foo": &InputValue{
				Value:      cty.StringVal("bar"),
				SourceType: ValueFromCaller,
			},
		},
	})

	var value interface{}
	p.ValidateResourceFn = func(t string, c *ResourceConfig) ([]string, []error) {
		value, _ = c.Get("foo")
		return nil, nil
	}

	c.Validate()
	if value != "bar" {
		t.Fatalf("bad: %#v", value)
	}
}

// Module variables weren't being interpolated during Validate phase.
// related to https://github.com/hashicorp/terraform/issues/5322
func TestContext2Validate_interpolateVar(t *testing.T) {
	input := new(MockUIInput)

	m := testModule(t, "input-interpolate-var")
	p := testProvider("null")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"template": testProviderFuncFixed(p),
			},
		),
		UIInput: input,
	})

	diags := ctx.Validate()
	if diags.HasErrors() {
		t.Fatalf("bad: %#v", diags)
	}
}

// When module vars reference something that is actually computed, this
// shouldn't cause validation to fail.
func TestContext2Validate_interpolateComputedModuleVarDef(t *testing.T) {
	input := new(MockUIInput)

	m := testModule(t, "validate-computed-module-var-ref")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		UIInput: input,
	})

	diags := ctx.Validate()
	if diags.HasErrors() {
		t.Fatalf("bad: %#v", diags)
	}
}

// Computed values are lost when a map is output from a module
func TestContext2Validate_interpolateMap(t *testing.T) {
	input := new(MockUIInput)

	m := testModule(t, "issue-9549")
	p := testProvider("null")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"template": testProviderFuncFixed(p),
			},
		),
		UIInput: input,
	})

	diags := ctx.Validate()
	if diags.HasErrors() {
		t.Fatalf("bad: %#v", diags)
	}
}

// Manually validate using the new PlanGraphBuilder
func TestContext2Validate_PlanGraphBuilder(t *testing.T) {
	m := testModule(t, "apply-vars")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Variables: InputValues{
			"foo": &InputValue{
				Value:      cty.StringVal("us-west-2"),
				SourceType: ValueFromCaller,
			},
			"test_list": &InputValue{
				Value: cty.ListVal([]cty.Value{
					cty.StringVal("Hello"),
					cty.StringVal("World"),
				}),
				SourceType: ValueFromCaller,
			},
			"test_map": &InputValue{
				Value: cty.MapVal(map[string]cty.Value{
					"Hello": cty.StringVal("World"),
					"Foo":   cty.StringVal("Bar"),
					"Baz":   cty.StringVal("Foo"),
				}),
				SourceType: ValueFromCaller,
			},
			"amis": &InputValue{
				Value: cty.ListVal([]cty.Value{
					cty.MapVal(map[string]cty.Value{
						"us-east-1": cty.StringVal("override"),
					}),
				}),
				SourceType: ValueFromCaller,
			},
		},
	})

	graph, diags := (&PlanGraphBuilder{
		Config:     c.config,
		State:      NewState(),
		Components: c.components,
		Targets:    c.targets,
	}).Build(addrs.RootModuleInstance)
	if diags.HasErrors() {
		t.Fatalf("errors from PlanGraphBuilder: %s", diags.Err())
	}
	defer c.acquireRun("validate-test")()
	walker, diags := c.walk(graph, walkValidate)
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}
	if len(walker.NonFatalDiagnostics) > 0 {
		t.Fatal(walker.NonFatalDiagnostics.Err())
	}
}
