package terraform

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configschema"
)

func TestContext2Plan_basic(t *testing.T) {
	m := testModule(t, "plan-good")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		ProviderSHA256s: map[string][]byte{
			"aws": []byte("placeholder"),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if l := len(plan.Changes.Resources); l < 2 {
		t.Fatalf("wrong number of resources %d; want fewer than two\n%s", l, spew.Sdump(plan.Changes.Resources))
	}

	if !reflect.DeepEqual(plan.ProviderSHA256s, ctx.providerSHA256s) {
		t.Errorf("wrong ProviderSHA256s %#v; want %#v", plan.ProviderSHA256s, ctx.providerSHA256s)
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Plan_createBefore_deposed(t *testing.T) {
	m := testModule(t, "plan-cbd")
	p := testProvider("aws")
	p.DiffFn = testDiffFn

	s := mustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: []string{"root"},
				Resources: map[string]*ResourceState{
					"aws_instance.foo": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "baz",
						},
						Deposed: []*InstanceState{
							&InstanceState{ID: "foo"},
						},
					},
				},
			},
		},
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(`
DIFF:

DESTROY: aws_instance.foo (deposed only)

STATE:

aws_instance.foo: (1 deposed)
  ID = baz
  Deposed ID 1 = foo
		`)
	if actual != expected {
		t.Fatalf("expected:\n%s, got:\n%s", expected, actual)
	}
}

func TestContext2Plan_createBefore_maintainRoot(t *testing.T) {
	m := testModule(t, "plan-cbd-maintain-root")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Variables: InputValues{
			"in": &InputValue{
				Value:      cty.StringVal("a,b,c"),
				SourceType: ValueFromCaller,
			},
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(`
DIFF:

CREATE: aws_instance.bar.0
CREATE: aws_instance.bar.1
CREATE: aws_instance.foo.0
CREATE: aws_instance.foo.1

STATE:

<no state>
		`)
	if actual != expected {
		t.Fatalf("expected:\n%s, got:\n%s", expected, actual)
	}
}

func TestContext2Plan_emptyDiff(t *testing.T) {
	m := testModule(t, "plan-empty")
	p := testProvider("aws")
	p.DiffFn = func(
		info *InstanceInfo,
		s *InstanceState,
		c *ResourceConfig) (*InstanceDiff, error) {
		return nil, nil
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanEmptyStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Plan_escapedVar(t *testing.T) {
	m := testModule(t, "plan-escaped-var")
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

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanEscapedVarStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Plan_minimal(t *testing.T) {
	m := testModule(t, "plan-empty")
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

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanEmptyStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Plan_modules(t *testing.T) {
	m := testModule(t, "plan-modules")
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

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanModulesStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

// GH-1475
func TestContext2Plan_moduleCycle(t *testing.T) {
	m := testModule(t, "plan-module-cycle")
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Computed: true},
					"some_input": {Type: cty.String, Optional: true},
				},
			},
		},
	}
	p.DiffFn = testDiffFn

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanModuleCycleStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Plan_moduleDeadlock(t *testing.T) {
	testCheckDeadlock(t, func() {
		m := testModule(t, "plan-module-deadlock")
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

		plan, err := ctx.Plan()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
		expected := strings.TrimSpace(`
DIFF:

module.child:
  CREATE: aws_instance.foo.0
  CREATE: aws_instance.foo.1
  CREATE: aws_instance.foo.2

STATE:

<no state>
		`)
		if actual != expected {
			t.Fatalf("expected:\n%sgot:\n%s", expected, actual)
		}
	})
}

func TestContext2Plan_moduleInput(t *testing.T) {
	m := testModule(t, "plan-module-input")
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

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanModuleInputStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Plan_moduleInputComputed(t *testing.T) {
	m := testModule(t, "plan-module-input-computed")
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

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanModuleInputComputedStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Plan_moduleInputFromVar(t *testing.T) {
	m := testModule(t, "plan-module-input-var")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Variables: InputValues{
			"foo": &InputValue{
				Value:      cty.StringVal("52"),
				SourceType: ValueFromCaller,
			},
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanModuleInputVarStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Plan_moduleMultiVar(t *testing.T) {
	m := testModule(t, "plan-module-multi-var")
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"id":  {Type: cty.String, Computed: true},
					"foo": {Type: cty.String, Optional: true},
					"baz": {Type: cty.String, Optional: true},
				},
			},
		},
	}
	p.DiffFn = testDiffFn

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanModuleMultiVarStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Plan_moduleOrphans(t *testing.T) {
	m := testModule(t, "plan-modules-remove")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	s := mustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: []string{"root", "child"},
				Resources: map[string]*ResourceState{
					"aws_instance.foo": &ResourceState{
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanModuleOrphansStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

// https://github.com/hashicorp/terraform/issues/3114
func TestContext2Plan_moduleOrphansWithProvisioner(t *testing.T) {
	m := testModule(t, "plan-modules-remove-provisioners")
	p := testProvider("aws")
	pr := testProvisioner()
	p.DiffFn = testDiffFn
	s := mustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: []string{"root"},
				Resources: map[string]*ResourceState{
					"aws_instance.top": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "top",
						},
					},
				},
			},
			&ModuleState{
				Path: []string{"root", "parent", "childone"},
				Resources: map[string]*ResourceState{
					"aws_instance.foo": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "baz",
						},
					},
				},
			},
			&ModuleState{
				Path: []string{"root", "parent", "childtwo"},
				Resources: map[string]*ResourceState{
					"aws_instance.foo": &ResourceState{
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ResourceProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
		State: s,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(`
DIFF:

module.parent.module.childone:
  DESTROY: aws_instance.foo
module.parent.module.childtwo:
  DESTROY: aws_instance.foo

STATE:

aws_instance.top:
  ID = top

module.parent.childone:
  aws_instance.foo:
    ID = baz
module.parent.childtwo:
  aws_instance.foo:
    ID = baz
	`)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Plan_moduleProviderInherit(t *testing.T) {
	var l sync.Mutex
	var calls []string

	m := testModule(t, "plan-module-provider-inherit")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": func() (ResourceProvider, error) {
					l.Lock()
					defer l.Unlock()

					p := testProvider("aws")
					p.GetSchemaReturn = &ProviderSchema{
						Provider: &configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"from": {Type: cty.String, Optional: true},
							},
						},
						ResourceTypes: map[string]*configschema.Block{
							"aws_instance": {
								Attributes: map[string]*configschema.Attribute{
									"from": {Type: cty.String, Optional: true},
								},
							},
						},
					}
					p.ConfigureFn = func(c *ResourceConfig) error {
						if v, ok := c.Get("from"); !ok || v.(string) != "root" {
							return fmt.Errorf("bad")
						}

						return nil
					}
					p.DiffFn = func(
						info *InstanceInfo,
						state *InstanceState,
						c *ResourceConfig) (*InstanceDiff, error) {
						v, _ := c.Get("from")

						l.Lock()
						defer l.Unlock()
						calls = append(calls, v.(string))
						return testDiffFn(info, state, c)
					}
					return p, nil
				},
			},
		),
	})

	_, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := calls
	sort.Strings(actual)
	expected := []string{"child", "root"}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

// This tests (for GH-11282) that deeply nested modules properly inherit
// configuration.
func TestContext2Plan_moduleProviderInheritDeep(t *testing.T) {
	var l sync.Mutex

	m := testModule(t, "plan-module-provider-inherit-deep")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": func() (ResourceProvider, error) {
					l.Lock()
					defer l.Unlock()

					var from string
					p := testProvider("aws")

					p.GetSchemaReturn = &ProviderSchema{
						Provider: &configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"from": {Type: cty.String, Optional: true},
							},
						},
						ResourceTypes: map[string]*configschema.Block{
							"aws_instance": {
								Attributes: map[string]*configschema.Attribute{},
							},
						},
					}

					p.ConfigureFn = func(c *ResourceConfig) error {
						v, ok := c.Get("from")
						if !ok || v.(string) != "root" {
							return fmt.Errorf("bad")
						}

						from = v.(string)
						return nil
					}

					p.DiffFn = func(
						info *InstanceInfo,
						state *InstanceState,
						c *ResourceConfig) (*InstanceDiff, error) {
						if from != "root" {
							return nil, fmt.Errorf("bad resource")
						}

						return testDiffFn(info, state, c)
					}
					return p, nil
				},
			},
		),
	})

	_, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestContext2Plan_moduleProviderDefaultsVar(t *testing.T) {
	var l sync.Mutex
	var calls []string

	m := testModule(t, "plan-module-provider-defaults-var")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": func() (ResourceProvider, error) {
					l.Lock()
					defer l.Unlock()

					p := testProvider("aws")
					p.GetSchemaReturn = &ProviderSchema{
						Provider: &configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"to":   {Type: cty.String, Optional: true},
								"from": {Type: cty.String, Optional: true},
							},
						},
						ResourceTypes: map[string]*configschema.Block{
							"aws_instance": {
								Attributes: map[string]*configschema.Attribute{
									"from": {Type: cty.String, Optional: true},
								},
							},
						},
					}
					p.ConfigureFn = func(c *ResourceConfig) error {
						var buf bytes.Buffer
						if v, ok := c.Get("from"); ok {
							buf.WriteString(v.(string) + "\n")
						}
						if v, ok := c.Get("to"); ok {
							buf.WriteString(v.(string) + "\n")
						}

						l.Lock()
						defer l.Unlock()
						calls = append(calls, buf.String())
						return nil
					}
					p.DiffFn = testDiffFn
					return p, nil
				},
			},
		),
		Variables: InputValues{
			"foo": &InputValue{
				Value:      cty.StringVal("root"),
				SourceType: ValueFromCaller,
			},
		},
	})

	_, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := []string{
		"child\nchild\n",
		"root\n",
	}
	sort.Strings(calls)
	if !reflect.DeepEqual(calls, expected) {
		t.Fatalf("expected:\n%#v\ngot:\n%#v\n", expected, calls)
	}
}

func TestContext2Plan_moduleProviderVar(t *testing.T) {
	m := testModule(t, "plan-module-provider-var")
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		Provider: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"value": {Type: cty.String, Optional: true},
			},
		},
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"value": {Type: cty.String, Optional: true},
				},
			},
		},
	}
	p.DiffFn = testDiffFn

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanModuleProviderVarStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Plan_moduleVar(t *testing.T) {
	m := testModule(t, "plan-module-var")
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

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanModuleVarStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Plan_moduleVarWrongTypeBasic(t *testing.T) {
	m := testModule(t, "plan-module-wrong-var-type")
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

	_, diags := ctx.Plan()
	if !diags.HasErrors() {
		t.Fatalf("succeeded; want errors")
	}
}

func TestContext2Plan_moduleVarWrongTypeNested(t *testing.T) {
	m := testModule(t, "plan-module-wrong-var-type-nested")
	p := testProvider("null")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"null": testProviderFuncFixed(p),
			},
		),
	})

	_, diags := ctx.Plan()
	if !diags.HasErrors() {
		t.Fatalf("succeeded; want errors")
	}
}

func TestContext2Plan_moduleVarWithDefaultValue(t *testing.T) {
	m := testModule(t, "plan-module-var-with-default-value")
	p := testProvider("null")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"null": testProviderFuncFixed(p),
			},
		),
	})

	_, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}
}

func TestContext2Plan_moduleVarComputed(t *testing.T) {
	m := testModule(t, "plan-module-var-computed")
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

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanModuleVarComputedStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Plan_nil(t *testing.T) {
	m := testModule(t, "plan-nil")
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"nil": {Type: cty.String, Optional: true},
				},
			},
		},
	}
	p.DiffFn = testDiffFn

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
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
								ID: "bar",
							},
						},
					},
				},
			},
		}),
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if len(plan.Changes.Resources) != 0 {
		t.Fatalf("bad: %#v", plan.Changes.Resources)
	}
}

func TestContext2Plan_preventDestroy_bad(t *testing.T) {
	m := testModule(t, "plan-prevent-destroy-bad")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
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
								ID: "i-abc123",
							},
						},
					},
				},
			},
		}),
	})

	plan, err := ctx.Plan()

	expectedErr := "aws_instance.foo has lifecycle.prevent_destroy"
	if !strings.Contains(fmt.Sprintf("%s", err), expectedErr) {
		if plan != nil {
			t.Logf(legacyDiffComparisonString(plan.Changes))
		}
		t.Fatalf("expected err would contain %q\nerr: %s", expectedErr, err)
	}
}

func TestContext2Plan_preventDestroy_good(t *testing.T) {
	m := testModule(t, "plan-prevent-destroy-good")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
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
								ID: "i-abc123",
							},
						},
					},
				},
			},
		}),
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if !plan.Changes.Empty() {
		t.Fatalf("Expected empty plan, got %s", legacyDiffComparisonString(plan.Changes))
	}
}

func TestContext2Plan_preventDestroy_countBad(t *testing.T) {
	m := testModule(t, "plan-prevent-destroy-count-bad")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: mustShimLegacyState(&State{
			Modules: []*ModuleState{
				&ModuleState{
					Path: rootModulePath,
					Resources: map[string]*ResourceState{
						"aws_instance.foo.0": &ResourceState{
							Type: "aws_instance",
							Primary: &InstanceState{
								ID: "i-abc123",
							},
						},
						"aws_instance.foo.1": &ResourceState{
							Type: "aws_instance",
							Primary: &InstanceState{
								ID: "i-abc345",
							},
						},
					},
				},
			},
		}),
	})

	plan, err := ctx.Plan()

	expectedErr := "aws_instance.foo[1] has lifecycle.prevent_destroy"
	if !strings.Contains(fmt.Sprintf("%s", err), expectedErr) {
		if plan != nil {
			t.Logf(legacyDiffComparisonString(plan.Changes))
		}
		t.Fatalf("expected err would contain %q\nerr: %s", expectedErr, err)
	}
}

func TestContext2Plan_preventDestroy_countGood(t *testing.T) {
	m := testModule(t, "plan-prevent-destroy-count-good")
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"current": {Type: cty.String, Optional: true},
				},
			},
		},
	}
	p.DiffFn = testDiffFn

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: mustShimLegacyState(&State{
			Modules: []*ModuleState{
				&ModuleState{
					Path: rootModulePath,
					Resources: map[string]*ResourceState{
						"aws_instance.foo.0": &ResourceState{
							Type: "aws_instance",
							Primary: &InstanceState{
								ID: "i-abc123",
							},
						},
						"aws_instance.foo.1": &ResourceState{
							Type: "aws_instance",
							Primary: &InstanceState{
								ID: "i-abc345",
							},
						},
					},
				},
			},
		}),
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if plan.Changes.Empty() {
		t.Fatalf("Expected non-empty plan, got %s", legacyDiffComparisonString(plan.Changes))
	}
}

func TestContext2Plan_preventDestroy_countGoodNoChange(t *testing.T) {
	m := testModule(t, "plan-prevent-destroy-count-good")
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"current": {Type: cty.String, Optional: true},
				},
			},
		},
	}
	p.DiffFn = testDiffFn

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: mustShimLegacyState(&State{
			Modules: []*ModuleState{
				&ModuleState{
					Path: rootModulePath,
					Resources: map[string]*ResourceState{
						"aws_instance.foo.0": &ResourceState{
							Type: "aws_instance",
							Primary: &InstanceState{
								ID: "i-abc123",
								Attributes: map[string]string{
									"current": "0",
									"type":    "aws_instance",
								},
							},
						},
					},
				},
			},
		}),
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if !plan.Changes.Empty() {
		t.Fatalf("Expected empty plan, got %s", legacyDiffComparisonString(plan.Changes))
	}
}

func TestContext2Plan_preventDestroy_destroyPlan(t *testing.T) {
	m := testModule(t, "plan-prevent-destroy-good")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
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
								ID: "i-abc123",
							},
						},
					},
				},
			},
		}),
		Destroy: true,
	})

	plan, diags := ctx.Plan()

	expectedErr := "aws_instance.foo has lifecycle.prevent_destroy"
	if !strings.Contains(fmt.Sprintf("%s", diags.Err()), expectedErr) {
		if plan != nil {
			t.Logf(legacyDiffComparisonString(plan.Changes))
		}
		t.Fatalf("expected err would contain %q\nerr: %s", expectedErr, diags.Err())
	}
}

func TestContext2Plan_provisionerCycle(t *testing.T) {
	m := testModule(t, "plan-provisioner-cycle")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	pr := testProvisioner()
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ResourceProvisionerFactory{
			"local-exec": testProvisionerFuncFixed(pr),
		},
	})

	_, diags := ctx.Plan()
	if !diags.HasErrors() {
		t.Fatalf("succeeded; want errors")
	}
}

func TestContext2Plan_computed(t *testing.T) {
	m := testModule(t, "plan-computed")
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

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanComputedStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Plan_computedDataResource(t *testing.T) {
	m := testModule(t, "plan-computed-data-resource")
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"num":     {Type: cty.String, Optional: true},
					"compute": {Type: cty.String, Optional: true},
					"foo":     {Type: cty.String, Computed: true},
				},
			},
		},
		DataSources: map[string]*configschema.Block{
			"aws_vpc": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
	}
	p.DiffFn = testDiffFn

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if rc := plan.Changes.ResourceInstance(addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "aws_instance", Name: "foo"}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)); rc == nil {
		t.Fatalf("missing diff for aws_instance.foo")
	}
	rcs := plan.Changes.ResourceInstance(addrs.Resource{
		Mode: addrs.DataResourceMode,
		Type: "aws_vpc",
		Name: "bar",
	}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance))
	if rcs == nil {
		t.Fatalf("missing diff for data.aws_vpc.bar")
	}

	rc, err := rcs.Decode(cty.Object(map[string]cty.Type{"id": cty.String}))
	if err != nil {
		t.Fatal(err)
	}
	want := cty.ObjectVal(map[string]cty.Value{
		"id": cty.UnknownVal(cty.String),
	})
	if got := rc.After; !got.RawEquals(want) {
		t.Fatalf(
			"incorrect new value for data.aws_vpc.bar\ngot:  %#v\nwant: %#v",
			got, want,
		)
	}
}

func TestContext2Plan_computedDataCountResource(t *testing.T) {
	m := testModule(t, "plan-computed-data-count")
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"num":     {Type: cty.String, Optional: true},
					"compute": {Type: cty.String, Optional: true},
					"foo":     {Type: cty.String, Computed: true},
				},
			},
		},
		DataSources: map[string]*configschema.Block{
			"aws_vpc": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
	}
	p.DiffFn = testDiffFn

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	// make sure we created 3 "bar"s
	for i := 0; i < 3; i++ {
		addr := addrs.Resource{
			Mode: addrs.DataResourceMode,
			Type: "aws_vpc",
			Name: "bar",
		}.Instance(addrs.IntKey(i)).Absolute(addrs.RootModuleInstance)

		if rcs := plan.Changes.ResourceInstance(addr); rcs == nil {
			t.Fatalf("missing changes for %s", addr)
		}
	}
}

func TestContext2Plan_localValueCount(t *testing.T) {
	m := testModule(t, "plan-local-value-count")
	p := testProvider("test")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"test": testProviderFuncFixed(p),
			},
		),
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	// make sure we created 3 "foo"s
	for i := 0; i < 3; i++ {
		addr := addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_resource",
			Name: "foo",
		}.Instance(addrs.IntKey(i)).Absolute(addrs.RootModuleInstance)

		if rcs := plan.Changes.ResourceInstance(addr); rcs == nil {
			t.Fatalf("missing changes for %s", addr)
		}
	}
}

// Higher level test at TestResource_dataSourceListPlanPanic
func TestContext2Plan_dataSourceTypeMismatch(t *testing.T) {
	m := testModule(t, "plan-data-source-type-mismatch")
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"ami": {Type: cty.String, Optional: true},
				},
			},
		},
		DataSources: map[string]*configschema.Block{
			"aws_availability_zones": {
				Attributes: map[string]*configschema.Attribute{
					"names": {Type: cty.List(cty.String), Computed: true},
				},
			},
		},
	}
	p.ValidateResourceFn = func(t string, c *ResourceConfig) (ws []string, es []error) {
		// Emulate the type checking behavior of helper/schema based validation
		if t == "aws_instance" {
			ami, _ := c.Get("ami")
			switch a := ami.(type) {
			case string:
				// ok
			default:
				es = append(es, fmt.Errorf("Expected ami to be string, got %T", a))
			}
		}
		return
	}
	p.DiffFn = func(
		info *InstanceInfo,
		state *InstanceState,
		c *ResourceConfig) (*InstanceDiff, error) {
		if info.Type == "aws_instance" {
			// If we get to the diff, we should be able to assume types
			ami, _ := c.Get("ami")
			_ = ami.(string)
		}
		return nil, nil
	}
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		// Pretend like we ran a Refresh and the AZs data source was populated.
		State: mustShimLegacyState(&State{
			Modules: []*ModuleState{
				&ModuleState{
					Path: rootModulePath,
					Resources: map[string]*ResourceState{
						"data.aws_availability_zones.azs": &ResourceState{
							Type: "aws_availability_zones",
							Primary: &InstanceState{
								ID: "i-abc123",
								Attributes: map[string]string{
									"names.#": "2",
									"names.0": "us-east-1a",
									"names.1": "us-east-1b",
								},
							},
						},
					},
				},
			},
		}),
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	_, diags := ctx.Plan()

	if !diags.HasErrors() {
		t.Fatalf("Expected err, got none!")
	}
	expected := `Inappropriate value for attribute "ami": incorrect type; string required`
	if errStr := diags.Err().Error(); !strings.Contains(errStr, expected) {
		t.Fatalf("expected:\n\n%s\n\nto contain:\n\n%s", errStr, expected)
	}
}

func TestContext2Plan_dataResourceBecomesComputed(t *testing.T) {
	t.Fatal("not yet updated for new provider interface")
	/*
		m := testModule(t, "plan-data-resource-becomes-computed")
		p := testProvider("aws")

		p.GetSchemaReturn = &ProviderSchema{
			ResourceTypes: map[string]*configschema.Block{
				"aws_instance": {
					Attributes: map[string]*configschema.Attribute{
						"foo":      {Type: cty.String, Optional: true},
						"computed": {Type: cty.String, Computed: true},
					},
				},
			},
			DataSources: map[string]*configschema.Block{
				"aws_data_resource": {
					Attributes: map[string]*configschema.Attribute{
						"foo": {Type: cty.String, Optional: true},
					},
				},
			},
		}
		p.DiffFn = func(info *InstanceInfo, state *InstanceState, config *ResourceConfig) (*InstanceDiff, error) {
			if info.Type != "aws_instance" {
				t.Fatalf("don't know how to diff %s", info.Id)
				return nil, nil
			}

			return &InstanceDiff{
				Attributes: map[string]*ResourceAttrDiff{
					"computed": &ResourceAttrDiff{
						Old:         "",
						New:         "",
						NewComputed: true,
					},
				},
			}, nil
		}
		p.ReadDataDiffReturn = &InstanceDiff{
			Attributes: map[string]*ResourceAttrDiff{
				"foo": &ResourceAttrDiff{
					Old:         "",
					New:         "",
					NewComputed: true,
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
			State: mustShimLegacyState(&State{
				Modules: []*ModuleState{
					&ModuleState{
						Path: rootModulePath,
						Resources: map[string]*ResourceState{
							"data.aws_data_resource.foo": &ResourceState{
								Type: "aws_data_resource",
								Primary: &InstanceState{
									ID: "i-abc123",
									Attributes: map[string]string{
										"id":    "i-abc123",
										"value": "baz",
									},
								},
							},
						},
					},
				},
			}),
		})

		plan, diags := ctx.Plan()
		if diags.HasErrors() {
			t.Fatalf("unexpected errors: %s", diags.Err())
		}

		if !p.ReadDataDiffCalled {
			t.Fatal("ReadDataDiff wasn't called, but should've been")
		}
		if got, want := p.ReadDataDiffInfo.Id, "data.aws_data_resource.foo"; got != want {
			t.Fatalf("ReadDataDiff info id is %s; want %s", got, want)
		}

		rcs := plan.Changes.ResourceInstance(addrs.Resource{
			Mode: addrs.DataResourceMode,
			Type: "aws_data_resource",
			Name: "foo",
		}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance))
		if rcs == nil {
			t.Fatalf("missing diff for data.aws_data_resource.foo")
		}

		schema := p.GetSchemaReturn
		rtSchema := schema.DataSources["aws_data_resource"]

		rc, err := rcs.Decode(rtSchema.ImpliedType())
		if err != nil {
			t.Fatal(err)
		}

		// FIXME: Update this once p.ReadDataDiffReturn is replaced with its
		// new equivalent in the new provider interface, and then compare
		// the cty.Values directly.
		if same, _ := p.ReadDataDiffReturn.Same(iDiff); !same {
			t.Fatalf(
				"incorrect diff for data.data_resource.foo\ngot:  %#v\nwant: %#v",
				iDiff, p.ReadDataDiffReturn,
			)
		}
	*/
}

func TestContext2Plan_computedList(t *testing.T) {
	m := testModule(t, "plan-computed-list")
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"compute": {Type: cty.String, Optional: true},
					"foo":     {Type: cty.String, Optional: true},
					"num":     {Type: cty.String, Optional: true},
					"list":    {Type: cty.List(cty.String), Computed: true},
				},
			},
		},
	}
	p.DiffFn = testDiffFn

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanComputedListStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

// GH-8695. This tests that you can index into a computed list on a
// splatted resource.
func TestContext2Plan_computedMultiIndex(t *testing.T) {
	m := testModule(t, "plan-computed-multi-index")
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

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanComputedMultiIndexStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Plan_count(t *testing.T) {
	m := testModule(t, "plan-count")
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

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if len(plan.Changes.Resources) < 6 {
		t.Fatalf("bad: %#v", plan.Changes.Resources)
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanCountStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Plan_countComputed(t *testing.T) {
	m := testModule(t, "plan-count-computed")
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

	_, err := ctx.Plan()
	if err == nil {
		t.Fatal("should error")
	}
}

func TestContext2Plan_countComputedModule(t *testing.T) {
	m := testModule(t, "plan-count-computed-module")
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

	_, err := ctx.Plan()

	expectedErr := `The "count" value depends on resource attributes`
	if !strings.Contains(fmt.Sprintf("%s", err), expectedErr) {
		t.Fatalf("expected err would contain %q\nerr: %s\n",
			expectedErr, err)
	}
}

func TestContext2Plan_countModuleStatic(t *testing.T) {
	m := testModule(t, "plan-count-module-static")
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

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(`
DIFF:

module.child:
  CREATE: aws_instance.foo.0
  CREATE: aws_instance.foo.1
  CREATE: aws_instance.foo.2

STATE:

<no state>
`)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Plan_countModuleStaticGrandchild(t *testing.T) {
	m := testModule(t, "plan-count-module-static-grandchild")
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

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(`
DIFF:

module.child.module.child:
  CREATE: aws_instance.foo.0
  CREATE: aws_instance.foo.1
  CREATE: aws_instance.foo.2

STATE:

<no state>
`)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Plan_countIndex(t *testing.T) {
	m := testModule(t, "plan-count-index")
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

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanCountIndexStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Plan_countVar(t *testing.T) {
	m := testModule(t, "plan-count-var")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Variables: InputValues{
			"instance_count": &InputValue{
				Value:      cty.StringVal("3"),
				SourceType: ValueFromCaller,
			},
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanCountVarStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Plan_countZero(t *testing.T) {
	m := testModule(t, "plan-count-zero")
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.DynamicPseudoType, Optional: true},
				},
			},
		},
	}
	p.DiffFn = testDiffFn

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanCountZeroStr)
	if actual != expected {
		t.Logf("expected:\n%s", expected)
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContext2Plan_countOneIndex(t *testing.T) {
	m := testModule(t, "plan-count-one-index")
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

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanCountOneIndexStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Plan_countDecreaseToOne(t *testing.T) {
	m := testModule(t, "plan-count-dec")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	s := mustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.foo.0": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
							Attributes: map[string]string{
								"foo":  "foo",
								"type": "aws_instance",
							},
						},
					},
					"aws_instance.foo.1": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
						},
					},
					"aws_instance.foo.2": &ResourceState{
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanCountDecreaseStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Plan_countIncreaseFromNotSet(t *testing.T) {
	m := testModule(t, "plan-count-inc")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	s := mustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.foo": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
							Attributes: map[string]string{
								"foo":  "foo",
								"type": "aws_instance",
							},
						},
					},
				},
			},
		},
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanCountIncreaseStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Plan_countIncreaseFromOne(t *testing.T) {
	m := testModule(t, "plan-count-inc")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	s := mustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.foo.0": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
							Attributes: map[string]string{
								"foo":  "foo",
								"type": "aws_instance",
							},
						},
					},
				},
			},
		},
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanCountIncreaseFromOneStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

// https://github.com/PeoplePerHour/terraform/pull/11
//
// This tests a case where both a "resource" and "resource.0" are in
// the state file, which apparently is a reasonable backwards compatibility
// concern found in the above 3rd party repo.
func TestContext2Plan_countIncreaseFromOneCorrupted(t *testing.T) {
	m := testModule(t, "plan-count-inc")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	s := mustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.foo": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
							Attributes: map[string]string{
								"foo":  "foo",
								"type": "aws_instance",
							},
						},
					},
					"aws_instance.foo.0": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
							Attributes: map[string]string{
								"foo":  "foo",
								"type": "aws_instance",
							},
						},
					},
				},
			},
		},
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanCountIncreaseFromOneCorruptedStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

// A common pattern in TF configs is to have a set of resources with the same
// count and to use count.index to create correspondences between them:
//
//    foo_id = "${foo.bar.*.id[count.index]}"
//
// This test is for the situation where some instances already exist and the
// count is increased. In that case, we should see only the create diffs
// for the new instances and not any update diffs for the existing ones.
func TestContext2Plan_countIncreaseWithSplatReference(t *testing.T) {
	m := testModule(t, "plan-count-splat-reference")
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"name":     {Type: cty.String, Optional: true},
					"foo_name": {Type: cty.String, Optional: true},
				},
			},
		},
	}
	p.DiffFn = testDiffFn

	s := mustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.foo.0": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
							Attributes: map[string]string{
								"name": "foo 0",
							},
						},
					},
					"aws_instance.foo.1": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
							Attributes: map[string]string{
								"name": "foo 1",
							},
						},
					},
					"aws_instance.bar.0": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
							Attributes: map[string]string{
								"foo_name": "foo 0",
							},
						},
					},
					"aws_instance.bar.1": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
							Attributes: map[string]string{
								"foo_name": "foo 1",
							},
						},
					},
				},
			},
		},
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(`
DIFF:

CREATE: aws_instance.bar.2
  foo_name: "" => "foo 2"
  type:     "" => "aws_instance"
CREATE: aws_instance.foo.2
  name: "" => "foo 2"
  type: "" => "aws_instance"

STATE:

aws_instance.bar.0:
  ID = bar
  foo_name = foo 0
aws_instance.bar.1:
  ID = bar
  foo_name = foo 1
aws_instance.foo.0:
  ID = bar
  name = foo 0
aws_instance.foo.1:
  ID = bar
  name = foo 1
`)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Plan_destroy(t *testing.T) {
	m := testModule(t, "plan-destroy")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	s := mustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.one": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
						},
					},
					"aws_instance.two": &ResourceState{
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State:   s,
		Destroy: true,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if len(plan.Changes.Resources) != 2 {
		t.Fatalf("bad: %#v", plan.Changes.Resources)
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanDestroyStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Plan_moduleDestroy(t *testing.T) {
	m := testModule(t, "plan-module-destroy")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	s := mustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.foo": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
						},
					},
				},
			},
			&ModuleState{
				Path: []string{"root", "child"},
				Resources: map[string]*ResourceState{
					"aws_instance.foo": &ResourceState{
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State:   s,
		Destroy: true,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanModuleDestroyStr)
	if actual != expected {
		t.Fatalf("bad:\n%s\n\nexpected:\n\n%s", actual, expected)
	}
}

// GH-1835
func TestContext2Plan_moduleDestroyCycle(t *testing.T) {
	m := testModule(t, "plan-module-destroy-gh-1835")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	s := mustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: []string{"root", "a_module"},
				Resources: map[string]*ResourceState{
					"aws_instance.a": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "a",
						},
					},
				},
			},
			&ModuleState{
				Path: []string{"root", "b_module"},
				Resources: map[string]*ResourceState{
					"aws_instance.b": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "b",
						},
					},
				},
			},
		},
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State:   s,
		Destroy: true,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanModuleDestroyCycleStr)
	if actual != expected {
		t.Fatalf("bad:\n%s\n\nexpected:\n\n%s", actual, expected)
	}
}

func TestContext2Plan_moduleDestroyMultivar(t *testing.T) {
	m := testModule(t, "plan-module-destroy-multivar")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	s := mustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path:      rootModulePath,
				Resources: map[string]*ResourceState{},
			},
			&ModuleState{
				Path: []string{"root", "child"},
				Resources: map[string]*ResourceState{
					"aws_instance.foo.0": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar0",
						},
					},
					"aws_instance.foo.1": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar1",
						},
					},
				},
			},
		},
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State:   s,
		Destroy: true,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanModuleDestroyMultivarStr)
	if actual != expected {
		t.Fatalf("bad:\n%s\n\nexpected:\n\n%s", actual, expected)
	}
}

func TestContext2Plan_pathVar(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	m := testModule(t, "plan-path-var")
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"cwd":    {Type: cty.String, Optional: true},
					"module": {Type: cty.String, Optional: true},
					"root":   {Type: cty.String, Optional: true},
				},
			},
		},
	}
	p.DiffFn = testDiffFn

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("err: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanPathVarStr)

	// Warning: this ordering REALLY matters for this test. The
	// order is: cwd, module, root.
	expected = fmt.Sprintf(
		expected,
		cwd,
		m.Module.SourceDir,
		m.Module.SourceDir,
	)

	if actual != expected {
		t.Fatalf("bad:\n%s\n\nexpected:\n\n%s", actual, expected)
	}
}

func TestContext2Plan_diffVar(t *testing.T) {
	m := testModule(t, "plan-diffvar")
	p := testProvider("aws")
	s := mustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.foo": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
							Attributes: map[string]string{
								"num": "2",
							},
						},
					},
				},
			},
		},
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	p.DiffFn = func(
		info *InstanceInfo,
		s *InstanceState,
		c *ResourceConfig) (*InstanceDiff, error) {
		if s.ID != "bar" {
			return testDiffFn(info, s, c)
		}

		return &InstanceDiff{
			Attributes: map[string]*ResourceAttrDiff{
				"num": &ResourceAttrDiff{
					Old: "2",
					New: "3",
				},
			},
		}, nil
	}

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanDiffVarStr)
	if actual != expected {
		t.Fatalf("actual:\n%s\n\nexpected:\n%s", actual, expected)
	}
}

func TestContext2Plan_hook(t *testing.T) {
	m := testModule(t, "plan-good")
	h := new(MockHook)
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Hooks:  []Hook{h},
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	_, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if !h.PreDiffCalled {
		t.Fatal("should be called")
	}
	if !h.PostDiffCalled {
		t.Fatal("should be called")
	}
}

func TestContext2Plan_closeProvider(t *testing.T) {
	// this fixture only has an aliased provider located in the module, to make
	// sure that the provier name contains a path more complex than
	// "provider.aws".
	m := testModule(t, "plan-close-module-provider")
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

	_, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if !p.CloseCalled {
		t.Fatal("provider not closed")
	}
}

func TestContext2Plan_orphan(t *testing.T) {
	m := testModule(t, "plan-orphan")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	s := mustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.baz": &ResourceState{
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanOrphanStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

// This tests that configurations with UUIDs don't produce errors.
// For shadows, this would produce errors since a UUID changes every time.
func TestContext2Plan_shadowUuid(t *testing.T) {
	m := testModule(t, "plan-shadow-uuid")
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

	_, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}
}

func TestContext2Plan_state(t *testing.T) {
	m := testModule(t, "plan-good")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	s := mustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.foo": &ResourceState{
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if len(plan.Changes.Resources) < 2 {
		t.Fatalf("bad: %#v", plan.Changes.Resources)
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanStateStr)
	if actual != expected {
		t.Fatalf("bad:\n%s\n\nexpected:\n\n%s", actual, expected)
	}
}

func TestContext2Plan_taint(t *testing.T) {
	m := testModule(t, "plan-taint")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	s := mustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.foo": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID:         "bar",
							Attributes: map[string]string{"num": "2"},
						},
					},
					"aws_instance.bar": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID:      "baz",
							Tainted: true,
						},
					},
				},
			},
		},
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanTaintStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Plan_taintIgnoreChanges(t *testing.T) {
	m := testModule(t, "plan-taint-ignore-changes")
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"vars": {Type: cty.String, Optional: true},
				},
			},
		},
	}
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	s := mustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.foo": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
							Attributes: map[string]string{
								"vars": "foo",
								"type": "aws_instance",
							},
							Tainted: true,
						},
					},
				},
			},
		},
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanTaintIgnoreChangesStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

// Fails about 50% of the time before the fix for GH-4982, covers the fix.
func TestContext2Plan_taintDestroyInterpolatedCountRace(t *testing.T) {
	m := testModule(t, "plan-taint-interpolated-count")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	s := mustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.foo.0": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID:      "bar",
							Tainted: true,
						},
					},
					"aws_instance.foo.1": &ResourceState{
						Type:    "aws_instance",
						Primary: &InstanceState{ID: "bar"},
					},
					"aws_instance.foo.2": &ResourceState{
						Type:    "aws_instance",
						Primary: &InstanceState{ID: "bar"},
					},
				},
			},
		},
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	for i := 0; i < 100; i++ {
		plan, diags := ctx.Plan()
		if diags.HasErrors() {
			t.Fatalf("unexpected errors: %s", diags.Err())
		}

		actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
		expected := strings.TrimSpace(`
DIFF:

DESTROY/CREATE: aws_instance.foo.0
  type: "" => "aws_instance"

STATE:

aws_instance.foo.0: (tainted)
  ID = bar
aws_instance.foo.1:
  ID = bar
aws_instance.foo.2:
  ID = bar
		`)
		if actual != expected {
			t.Fatalf("[%d] bad:\n%s\nexpected:\n%s\n", i, actual, expected)
		}
	}
}

func TestContext2Plan_targeted(t *testing.T) {
	m := testModule(t, "plan-targeted")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(`
DIFF:

CREATE: aws_instance.foo
  num:  "" => "2"
  type: "" => "aws_instance"

STATE:

<no state>
	`)
	if actual != expected {
		t.Fatalf("expected:\n%s\n\ngot:\n%s", expected, actual)
	}
}

// Test that targeting a module properly plans any inputs that depend
// on another module.
func TestContext2Plan_targetedCrossModule(t *testing.T) {
	m := testModule(t, "plan-targeted-cross-module")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Child("B", addrs.NoKey),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(`
DIFF:

module.A:
  CREATE: aws_instance.foo
    foo:  "" => "bar"
    type: "" => "aws_instance"
module.B:
  CREATE: aws_instance.bar
    foo:  "" => "<computed>"
    type: "" => "aws_instance"

STATE:

<no state>
	`)
	if actual != expected {
		t.Fatalf("expected:\n%s\n\ngot:\n%s", expected, actual)
	}
}

func TestContext2Plan_targetedModuleWithProvider(t *testing.T) {
	m := testModule(t, "plan-targeted-module-with-provider")
	p := testProvider("null")
	p.GetSchemaReturn = &ProviderSchema{
		Provider: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"key": {Type: cty.String, Optional: true},
			},
		},
		ResourceTypes: map[string]*configschema.Block{
			"null_resource": {
				Attributes: map[string]*configschema.Attribute{},
			},
		},
	}
	p.DiffFn = testDiffFn

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"null": testProviderFuncFixed(p),
			},
		),
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Child("child2", addrs.NoKey),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(`
DIFF:

module.child2:
  CREATE: null_resource.foo

STATE:

<no state>
	`)
	if actual != expected {
		t.Fatalf("expected:\n%s\n\ngot:\n%s", expected, actual)
	}
}

func TestContext2Plan_targetedOrphan(t *testing.T) {
	m := testModule(t, "plan-targeted-orphan")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: mustShimLegacyState(&State{
			Modules: []*ModuleState{
				&ModuleState{
					Path: rootModulePath,
					Resources: map[string]*ResourceState{
						"aws_instance.orphan": &ResourceState{
							Type: "aws_instance",
							Primary: &InstanceState{
								ID: "i-789xyz",
							},
						},
						"aws_instance.nottargeted": &ResourceState{
							Type: "aws_instance",
							Primary: &InstanceState{
								ID: "i-abc123",
							},
						},
					},
				},
			},
		}),
		Destroy: true,
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "orphan",
			),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(`DIFF:

DESTROY: aws_instance.orphan

STATE:

aws_instance.nottargeted:
  ID = i-abc123
aws_instance.orphan:
  ID = i-789xyz
`)
	if actual != expected {
		t.Fatalf("expected:\n%s\n\ngot:\n%s", expected, actual)
	}
}

// https://github.com/hashicorp/terraform/issues/2538
func TestContext2Plan_targetedModuleOrphan(t *testing.T) {
	m := testModule(t, "plan-targeted-module-orphan")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: mustShimLegacyState(&State{
			Modules: []*ModuleState{
				&ModuleState{
					Path: []string{"root", "child"},
					Resources: map[string]*ResourceState{
						"aws_instance.orphan": &ResourceState{
							Type: "aws_instance",
							Primary: &InstanceState{
								ID: "i-789xyz",
							},
						},
						"aws_instance.nottargeted": &ResourceState{
							Type: "aws_instance",
							Primary: &InstanceState{
								ID: "i-abc123",
							},
						},
					},
				},
			},
		}),
		Destroy: true,
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Child("child", addrs.NoKey).Resource(
				addrs.ManagedResourceMode, "aws_instance", "orphan",
			),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(`DIFF:

module.child:
  DESTROY: aws_instance.orphan

STATE:

module.child:
  aws_instance.nottargeted:
    ID = i-abc123
  aws_instance.orphan:
    ID = i-789xyz
`)
	if actual != expected {
		t.Fatalf("expected:\n%s\n\ngot:\n%s", expected, actual)
	}
}

func TestContext2Plan_targetedModuleUntargetedVariable(t *testing.T) {
	m := testModule(t, "plan-targeted-module-untargeted-variable")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "blue",
			),
			addrs.RootModuleInstance.Child("blue_mod", addrs.NoKey),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(`
DIFF:

CREATE: aws_instance.blue

module.blue_mod:
  CREATE: aws_instance.mod
    type:  "" => "aws_instance"
    value: "" => "<computed>"

STATE:

<no state>
`)
	if actual != expected {
		t.Fatalf("expected:\n%s\n\ngot:\n%s", expected, actual)
	}
}

// ensure that outputs missing references due to targetting are removed from
// the graph.
func TestContext2Plan_outputContainsTargetedResource(t *testing.T) {
	m := testModule(t, "plan-untargeted-resource-output")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Child("mod", addrs.NoKey).Resource(
				addrs.ManagedResourceMode, "aws_instance", "a",
			),
		},
	})

	_, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
}

// https://github.com/hashicorp/terraform/issues/4515
func TestContext2Plan_targetedOverTen(t *testing.T) {
	m := testModule(t, "plan-targeted-over-ten")
	p := testProvider("aws")
	p.DiffFn = testDiffFn

	resources := make(map[string]*ResourceState)
	var expectedState []string
	for i := 0; i < 13; i++ {
		key := fmt.Sprintf("aws_instance.foo.%d", i)
		id := fmt.Sprintf("i-abc%d", i)
		resources[key] = &ResourceState{
			Type:    "aws_instance",
			Primary: &InstanceState{ID: id},
		}
		expectedState = append(expectedState,
			fmt.Sprintf("%s:\n  ID = %s\n", key, id))
	}
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: mustShimLegacyState(&State{
			Modules: []*ModuleState{
				&ModuleState{
					Path:      rootModulePath,
					Resources: resources,
				},
			},
		}),
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.ResourceInstance(
				addrs.ManagedResourceMode, "aws_instance", "foo", addrs.IntKey(1),
			),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	sort.Strings(expectedState)
	expected := strings.TrimSpace(`
DIFF:



STATE:

aws_instance.foo.0:
  ID = i-abc0
aws_instance.foo.1:
  ID = i-abc1
aws_instance.foo.2:
  ID = i-abc2
aws_instance.foo.3:
  ID = i-abc3
aws_instance.foo.4:
  ID = i-abc4
aws_instance.foo.5:
  ID = i-abc5
aws_instance.foo.6:
  ID = i-abc6
aws_instance.foo.7:
  ID = i-abc7
aws_instance.foo.8:
  ID = i-abc8
aws_instance.foo.9:
  ID = i-abc9
aws_instance.foo.10:
  ID = i-abc10
aws_instance.foo.11:
  ID = i-abc11
aws_instance.foo.12:
  ID = i-abc12
	`)
	if actual != expected {
		t.Fatalf("expected:\n%s\n\ngot:\n%s", expected, actual)
	}
}

func TestContext2Plan_provider(t *testing.T) {
	m := testModule(t, "plan-provider")
	p := testProvider("aws")
	p.DiffFn = testDiffFn

	var value interface{}
	p.ConfigureFn = func(c *ResourceConfig) error {
		value, _ = c.Get("foo")
		return nil
	}

	ctx := testContext2(t, &ContextOpts{
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

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	if value != "bar" {
		t.Fatalf("bad: %#v", value)
	}
}

func TestContext2Plan_varListErr(t *testing.T) {
	m := testModule(t, "plan-var-list-err")
	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	_, err := ctx.Plan()

	if err == nil {
		t.Fatal("should error")
	}
}

func TestContext2Plan_ignoreChanges(t *testing.T) {
	m := testModule(t, "plan-ignore-changes")
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"ami": {Type: cty.String, Optional: true},
				},
			},
		},
	}
	p.DiffFn = testDiffFn

	s := mustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.foo": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID:         "bar",
							Attributes: map[string]string{"ami": "ami-abcd1234"},
						},
					},
				},
			},
		},
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Variables: InputValues{
			"foo": &InputValue{
				Value:      cty.StringVal("ami-1234abcd"),
				SourceType: ValueFromCaller,
			},
		},
		State: s,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if len(plan.Changes.Resources) < 1 {
		t.Fatalf("bad: %#v", plan.Changes.Resources)
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanIgnoreChangesStr)
	if actual != expected {
		t.Fatalf("bad:\n%s\n\nexpected\n\n%s", actual, expected)
	}
}

func TestContext2Plan_ignoreChangesWildcard(t *testing.T) {
	m := testModule(t, "plan-ignore-changes-wildcard")
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"ami":           {Type: cty.String, Optional: true},
					"instance_type": {Type: cty.String, Optional: true},
				},
			},
		},
	}
	p.DiffFn = testDiffFn

	s := mustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.foo": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
							Attributes: map[string]string{
								"ami":           "ami-abcd1234",
								"instance_type": "t2.micro",
							},
						},
					},
				},
			},
		},
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Variables: InputValues{
			"foo": &InputValue{
				Value:      cty.StringVal("ami-1234abcd"),
				SourceType: ValueFromCaller,
			},
			"bar": &InputValue{
				Value:      cty.StringVal("t2.small"),
				SourceType: ValueFromCaller,
			},
		},
		State: s,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if len(plan.Changes.Resources) > 0 {
		t.Fatalf("unexpected resource diffs in root module: %s", spew.Sdump(plan.Changes.Resources))
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanIgnoreChangesWildcardStr)
	if actual != expected {
		t.Fatalf("bad:\n%s\n\nexpected\n\n%s", actual, expected)
	}
}

func TestContext2Plan_moduleMapLiteral(t *testing.T) {
	m := testModule(t, "plan-module-map-literal")
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"meta": {Type: cty.Map(cty.String), Optional: true},
					"tags": {Type: cty.Map(cty.String), Optional: true},
				},
			},
		},
	}
	p.ApplyFn = testApplyFn
	p.DiffFn = func(i *InstanceInfo, s *InstanceState, c *ResourceConfig) (*InstanceDiff, error) {
		// Here we verify that both the populated and empty map literals made it
		// through to the resource attributes
		val, _ := c.Get("tags")
		m, ok := val.(map[string]interface{})
		if !ok {
			t.Fatalf("Tags attr not map: %#v", val)
		}
		if m["foo"] != "bar" {
			t.Fatalf("Bad value in tags attr: %#v", m)
		}
		{
			val, _ := c.Get("meta")
			m, ok := val.(map[string]interface{})
			if !ok {
				t.Fatalf("Meta attr not map: %#v", val)
			}
			if len(m) != 0 {
				t.Fatalf("Meta attr not empty: %#v", val)
			}
		}
		return nil, nil
	}
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	_, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}
}

func TestContext2Plan_computedValueInMap(t *testing.T) {
	m := testModule(t, "plan-computed-value-in-map")
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"looked_up": {Type: cty.String, Optional: true},
				},
			},
			"aws_computed_source": {
				Attributes: map[string]*configschema.Attribute{
					"computed_read_only": {Type: cty.String, Computed: true},
				},
			},
		},
	}
	p.DiffFn = func(info *InstanceInfo, state *InstanceState, c *ResourceConfig) (*InstanceDiff, error) {
		switch info.Type {
		case "aws_computed_source":
			return &InstanceDiff{
				Attributes: map[string]*ResourceAttrDiff{
					"computed_read_only": &ResourceAttrDiff{
						NewComputed: true,
					},
				},
			}, nil
		}

		return testDiffFn(info, state, c)
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	// (not sure why this is repeated here; I updated some earlier code that
	// called ctx.Plan twice here, so retaining that in case it's somehow
	// important.)
	plan, diags = ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanComputedValueInMap)
	if actual != expected {
		t.Fatalf("bad:\n%s\n\nexpected\n\n%s", actual, expected)
	}
}

func TestContext2Plan_moduleVariableFromSplat(t *testing.T) {
	m := testModule(t, "plan-module-variable-from-splat")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"thing": {Type: cty.String, Optional: true},
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
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	// (not sure why this is repeated here; I updated some earlier code that
	// called ctx.Plan twice here, so retaining that in case it's somehow
	// important.)
	plan, diags = ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(testTerraformPlanModuleVariableFromSplat)
	if actual != expected {
		t.Fatalf("bad:\n%s\n\nexpected\n\n%s", actual, expected)
	}
}

func TestContext2Plan_createBeforeDestroy_depends_datasource(t *testing.T) {
	m := testModule(t, "plan-cbd-depends-datasource")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"num":     {Type: cty.String, Optional: true},
					"compute": {Type: cty.String, Optional: true, Computed: true},
				},
			},
		},
		DataSources: map[string]*configschema.Block{
			"aws_vpc": {
				Attributes: map[string]*configschema.Attribute{
					"id":  {Type: cty.String, Computed: true},
					"foo": {Type: cty.Number, Optional: true},
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
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	for i := 0; i < 2; i++ {
		{
			addr := addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.IntKey(i)).Absolute(addrs.RootModuleInstance)

			if rcs := plan.Changes.ResourceInstance(addr); rcs == nil {
				t.Fatalf("missing changes for %s", addr)
			}
		}
		{
			addr := addrs.Resource{
				Mode: addrs.DataResourceMode,
				Type: "aws_vpc",
				Name: "bar",
			}.Instance(addrs.IntKey(i)).Absolute(addrs.RootModuleInstance)

			if rcs := plan.Changes.ResourceInstance(addr); rcs == nil {
				t.Fatalf("missing changes for %s", addr)
			}
		}
	}
}

// interpolated lists need to be stored in the original order.
func TestContext2Plan_listOrder(t *testing.T) {
	m := testModule(t, "plan-list-order")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.List(cty.String), Optional: true},
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
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	changes := plan.Changes
	rDiffA := changes.ResourceInstance(addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "aws_instance",
		Name: "a",
	}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance))
	rDiffB := changes.ResourceInstance(addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "aws_instance",
		Name: "b",
	}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance))

	if !cmp.Equal(rDiffA, rDiffB) {
		t.Fatal("aws_instance.a and aws_instance.b diffs should match:\n", legacyDiffComparisonString(plan.Changes))
	}
}

// Make sure ignore-changes doesn't interfere with set/list/map diffs.
// If a resource was being replaced by a RequiresNew attribute that gets
// ignored, we need to filter the diff properly to properly update rather than
// replace.
func TestContext2Plan_ignoreChangesWithFlatmaps(t *testing.T) {
	m := testModule(t, "plan-ignore-changes-with-flatmaps")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"user_data":   {Type: cty.String, Optional: true},
					"require_new": {Type: cty.String, Optional: true},

					// This test predates the 0.12 work to integrate cty and
					// HCL, and so it was ported as-is where its expected
					// test output was clearly expecting a list of maps here
					// even though it is named "set".
					"set": {Type: cty.List(cty.Map(cty.String)), Optional: true},
					"lst": {Type: cty.List(cty.String), Optional: true},
				},
			},
		},
	}
	s := mustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.foo": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
							Attributes: map[string]string{
								"user_data":   "x",
								"require_new": "",
								"set.#":       "1",
								"set.0.a":     "1",
								"lst.#":       "1",
								"lst.0":       "j",
							},
						},
					},
				},
			},
		},
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyDiffComparisonString(plan.Changes))
	expected := strings.TrimSpace(testTFPlanDiffIgnoreChangesWithFlatmaps)
	if actual != expected {
		t.Fatalf("bad:\n%s\n\nexpected\n\n%s", actual, expected)
	}
}

// TestContext2Plan_resourceNestedCount ensures resource sets that depend on
// the count of another resource set (ie: count of a data source that depends
// on another data source's instance count - data.x.foo.*.id) get properly
// normalized to the indexes they should be. This case comes up when there is
// an existing state (after an initial apply).
func TestContext2Plan_resourceNestedCount(t *testing.T) {
	m := testModule(t, "nested-resource-count-plan")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	p.RefreshFn = func(i *InstanceInfo, is *InstanceState) (*InstanceState, error) {
		return is, nil
	}
	s := mustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.foo.0": &ResourceState{
						Type:     "aws_instance",
						Provider: "provider.aws",
						Primary: &InstanceState{
							ID: "foo0",
							Attributes: map[string]string{
								"id": "foo0",
							},
						},
					},
					"aws_instance.foo.1": &ResourceState{
						Type:     "aws_instance",
						Provider: "provider.aws",
						Primary: &InstanceState{
							ID: "foo1",
							Attributes: map[string]string{
								"id": "foo1",
							},
						},
					},
					"aws_instance.bar.0": &ResourceState{
						Type:         "aws_instance",
						Provider:     "provider.aws",
						Dependencies: []string{"aws_instance.foo"},
						Primary: &InstanceState{
							ID: "bar0",
							Attributes: map[string]string{
								"id": "bar0",
							},
						},
					},
					"aws_instance.bar.1": &ResourceState{
						Type:         "aws_instance",
						Provider:     "provider.aws",
						Dependencies: []string{"aws_instance.foo"},
						Primary: &InstanceState{
							ID: "bar1",
							Attributes: map[string]string{
								"id": "bar1",
							},
						},
					},
					"aws_instance.baz.0": &ResourceState{
						Type:         "aws_instance",
						Provider:     "provider.aws",
						Dependencies: []string{"aws_instance.bar"},
						Primary: &InstanceState{
							ID: "baz0",
							Attributes: map[string]string{
								"id": "baz0",
							},
						},
					},
					"aws_instance.baz.1": &ResourceState{
						Type:         "aws_instance",
						Provider:     "provider.aws",
						Dependencies: []string{"aws_instance.bar"},
						Primary: &InstanceState{
							ID: "baz1",
							Attributes: map[string]string{
								"id": "baz1",
							},
						},
					},
				},
			},
		},
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	diags := ctx.Validate()
	if diags.HasErrors() {
		t.Fatalf("validate errors: %s", diags.Err())
	}

	_, diags = ctx.Refresh()
	if diags.HasErrors() {
		t.Fatalf("refresh errors: %s", diags.Err())
	}

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), plan.Changes))
	expected := strings.TrimSpace(`
DIFF:



STATE:

aws_instance.bar.0:
  ID = bar0
  provider = provider.aws

  Dependencies:
    aws_instance.foo.*
aws_instance.bar.1:
  ID = bar1
  provider = provider.aws

  Dependencies:
    aws_instance.foo.*
aws_instance.baz.0:
  ID = baz0
  provider = provider.aws

  Dependencies:
    aws_instance.bar.*
aws_instance.baz.1:
  ID = baz1
  provider = provider.aws

  Dependencies:
    aws_instance.bar.*
aws_instance.foo.0:
  ID = foo0
  provider = provider.aws
aws_instance.foo.1:
  ID = foo1
  provider = provider.aws
`)
	if actual != expected {
		t.Fatalf("bad:\n%s\n\nexpected\n\n%s", actual, expected)
	}
}

// Higher level test at TestResource_dataSourceListApplyPanic
func TestContext2Plan_computedAttrRefTypeMismatch(t *testing.T) {
	m := testModule(t, "plan-computed-attr-ref-type-mismatch")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	p.ValidateResourceFn = func(t string, c *ResourceConfig) (ws []string, es []error) {
		// Emulate the type checking behavior of helper/schema based validation
		if t == "aws_instance" {
			ami, _ := c.Get("ami")
			switch a := ami.(type) {
			case string:
				// ok
			default:
				es = append(es, fmt.Errorf("Expected ami to be string, got %T", a))
			}
		}
		return
	}
	p.DiffFn = func(
		info *InstanceInfo,
		state *InstanceState,
		c *ResourceConfig) (*InstanceDiff, error) {
		switch info.Type {
		case "aws_ami_list":
			// Emulate a diff that says "we'll create this list and ids will be populated"
			return &InstanceDiff{
				Attributes: map[string]*ResourceAttrDiff{
					"ids.#": &ResourceAttrDiff{NewComputed: true},
				},
			}, nil
		case "aws_instance":
			// If we get to the diff for instance, we should be able to assume types
			ami, _ := c.Get("ami")
			_ = ami.(string)
		}
		return nil, nil
	}
	p.ApplyFn = func(info *InstanceInfo, s *InstanceState, d *InstanceDiff) (*InstanceState, error) {
		if info.Type != "aws_ami_list" {
			t.Fatalf("Reached apply for unexpected resource type! %s", info.Type)
		}
		// Pretend like we make a thing and the computed list "ids" is populated
		return &InstanceState{
			ID: "someid",
			Attributes: map[string]string{
				"ids.#": "2",
				"ids.0": "ami-abc123",
				"ids.1": "ami-bcd345",
			},
		}, nil
	}
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	_, diags := ctx.Plan()
	if !diags.HasErrors() {
		t.Fatalf("Succeeded; want type mismatch error for 'ami' argument")
	}

	expected := `Inappropriate value for attribute "ami"`
	if errStr := diags.Err().Error(); !strings.Contains(errStr, expected) {
		t.Fatalf("expected:\n\n%s\n\nto contain:\n\n%s", errStr, expected)
	}
}

func TestContext2Plan_selfRef(t *testing.T) {
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
	}

	m := testModule(t, "plan-self-ref")
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
		t.Fatalf("unexpected validation failure: %s", diags.Err())
	}

	_, diags = c.Plan()
	if !diags.HasErrors() {
		t.Fatalf("plan succeeded; want error")
	}

	gotErrStr := diags.Err().Error()
	wantErrStr := "Self-referential block"
	if !strings.Contains(gotErrStr, wantErrStr) {
		t.Fatalf("missing expected error\ngot: %s\n\nwant: error containing %q", gotErrStr, wantErrStr)
	}
}

func TestContext2Plan_selfRefMulti(t *testing.T) {
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
	}

	m := testModule(t, "plan-self-ref-multi")
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
		t.Fatalf("unexpected validation failure: %s", diags.Err())
	}

	_, diags = c.Plan()
	if !diags.HasErrors() {
		t.Fatalf("plan succeeded; want error")
	}

	gotErrStr := diags.Err().Error()
	wantErrStr := "Self-referential block"
	if !strings.Contains(gotErrStr, wantErrStr) {
		t.Fatalf("missing expected error\ngot: %s\n\nwant: error containing %q", gotErrStr, wantErrStr)
	}
}

func TestContext2Plan_selfRefMultiAll(t *testing.T) {
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.List(cty.String), Optional: true},
				},
			},
		},
	}

	m := testModule(t, "plan-self-ref-multi-all")
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
		t.Fatalf("unexpected validation failure: %s", diags.Err())
	}

	_, diags = c.Plan()
	if !diags.HasErrors() {
		t.Fatalf("plan succeeded; want error")
	}

	gotErrStr := diags.Err().Error()
	wantErrStr := "Self-referential block"
	if !strings.Contains(gotErrStr, wantErrStr) {
		t.Fatalf("missing expected error\ngot: %s\n\nwant: error containing %q", gotErrStr, wantErrStr)
	}
}
