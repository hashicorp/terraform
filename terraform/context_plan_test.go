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
)

func TestContext2Plan_basic(t *testing.T) {
	m := testModule(t, "plan-good")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		ProviderSHA256s: map[string][]byte{
			"aws": []byte("placeholder"),
		},
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if len(plan.Diff.RootModule().Resources) < 2 {
		t.Fatalf("bad: %#v", plan.Diff.RootModule().Resources)
	}

	if !reflect.DeepEqual(plan.ProviderSHA256s, ctx.providerSHA256s) {
		t.Errorf("wrong ProviderSHA256s %#v; want %#v", plan.ProviderSHA256s, ctx.providerSHA256s)
	}

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContext2Plan_createBefore_deposed(t *testing.T) {
	m := testModule(t, "plan-cbd")
	p := testProvider("aws")
	p.DiffFn = testDiffFn

	s := &State{
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
	}

	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(plan.String())
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
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Variables: map[string]interface{}{
			"in": "a,b,c",
		},
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(plan.String())
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
		Module: m,
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

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanEmptyStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContext2Plan_escapedVar(t *testing.T) {
	m := testModule(t, "plan-escaped-var")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanEscapedVarStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContext2Plan_minimal(t *testing.T) {
	m := testModule(t, "plan-empty")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanEmptyStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContext2Plan_modules(t *testing.T) {
	m := testModule(t, "plan-modules")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanModulesStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

// GH-1475
func TestContext2Plan_moduleCycle(t *testing.T) {
	m := testModule(t, "plan-module-cycle")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanModuleCycleStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContext2Plan_moduleDeadlock(t *testing.T) {
	testCheckDeadlock(t, func() {
		m := testModule(t, "plan-module-deadlock")
		p := testProvider("aws")
		p.DiffFn = testDiffFn

		ctx := testContext2(t, &ContextOpts{
			Module: m,
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

		actual := strings.TrimSpace(plan.String())
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
		Module: m,
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

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanModuleInputStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContext2Plan_moduleInputComputed(t *testing.T) {
	m := testModule(t, "plan-module-input-computed")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanModuleInputComputedStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContext2Plan_moduleInputFromVar(t *testing.T) {
	m := testModule(t, "plan-module-input-var")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Variables: map[string]interface{}{
			"foo": "52",
		},
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanModuleInputVarStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContext2Plan_moduleMultiVar(t *testing.T) {
	m := testModule(t, "plan-module-multi-var")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanModuleMultiVarStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContext2Plan_moduleOrphans(t *testing.T) {
	m := testModule(t, "plan-modules-remove")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	s := &State{
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
	}
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanModuleOrphansStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

// https://github.com/hashicorp/terraform/issues/3114
func TestContext2Plan_moduleOrphansWithProvisioner(t *testing.T) {
	m := testModule(t, "plan-modules-remove-provisioners")
	p := testProvider("aws")
	pr := testProvisioner()
	p.DiffFn = testDiffFn
	s := &State{
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
	}
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(`
DIFF:

module.parent.childone:
  DESTROY: aws_instance.foo
module.parent.childtwo:
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
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContext2Plan_moduleProviderInherit(t *testing.T) {
	var l sync.Mutex
	var calls []string

	m := testModule(t, "plan-module-provider-inherit")
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": func() (ResourceProvider, error) {
					l.Lock()
					defer l.Unlock()

					p := testProvider("aws")
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
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": func() (ResourceProvider, error) {
					l.Lock()
					defer l.Unlock()

					var from string
					p := testProvider("aws")
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
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": func() (ResourceProvider, error) {
					l.Lock()
					defer l.Unlock()

					p := testProvider("aws")
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
		Variables: map[string]interface{}{
			"foo": "root",
		},
	})

	_, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := []string{
		"root\n",
		// this test originally verified that a parent provider config can
		// partially override a child. That's no longer the case, so the child
		// config is used in its entirety here.
		//"root\nchild\n",
		"child\nchild\n",
	}
	if !reflect.DeepEqual(calls, expected) {
		t.Fatalf("expected:\n%#v\ngot:\n%#v\n", expected, calls)
	}
}

func TestContext2Plan_moduleProviderVar(t *testing.T) {
	m := testModule(t, "plan-module-provider-var")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanModuleProviderVarStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContext2Plan_moduleVar(t *testing.T) {
	m := testModule(t, "plan-module-var")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanModuleVarStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContext2Plan_moduleVarWrongTypeBasic(t *testing.T) {
	m := testModule(t, "plan-module-wrong-var-type")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	_, err := ctx.Plan()
	if err == nil {
		t.Fatalf("should error")
	}
}

func TestContext2Plan_moduleVarWrongTypeNested(t *testing.T) {
	m := testModule(t, "plan-module-wrong-var-type-nested")
	p := testProvider("null")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"null": testProviderFuncFixed(p),
			},
		),
	})

	_, err := ctx.Plan()
	if err == nil {
		t.Fatalf("should error")
	}
}

func TestContext2Plan_moduleVarWithDefaultValue(t *testing.T) {
	m := testModule(t, "plan-module-var-with-default-value")
	p := testProvider("null")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"null": testProviderFuncFixed(p),
			},
		),
	})

	_, err := ctx.Plan()
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
}

func TestContext2Plan_moduleVarComputed(t *testing.T) {
	m := testModule(t, "plan-module-var-computed")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanModuleVarComputedStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContext2Plan_nil(t *testing.T) {
	m := testModule(t, "plan-nil")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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
								ID: "bar",
							},
						},
					},
				},
			},
		},
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if len(plan.Diff.RootModule().Resources) != 0 {
		t.Fatalf("bad: %#v", plan.Diff.RootModule().Resources)
	}
}

func TestContext2Plan_preventDestroy_bad(t *testing.T) {
	m := testModule(t, "plan-prevent-destroy-bad")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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
								ID: "i-abc123",
							},
						},
					},
				},
			},
		},
	})

	plan, err := ctx.Plan()

	expectedErr := "aws_instance.foo: the plan would destroy"
	if !strings.Contains(fmt.Sprintf("%s", err), expectedErr) {
		t.Fatalf("expected err would contain %q\nerr: %s\nplan: %s",
			expectedErr, err, plan)
	}
}

func TestContext2Plan_preventDestroy_good(t *testing.T) {
	m := testModule(t, "plan-prevent-destroy-good")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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
								ID: "i-abc123",
							},
						},
					},
				},
			},
		},
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if !plan.Diff.Empty() {
		t.Fatalf("Expected empty plan, got %s", plan.String())
	}
}

func TestContext2Plan_preventDestroy_countBad(t *testing.T) {
	m := testModule(t, "plan-prevent-destroy-count-bad")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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
		},
	})

	plan, err := ctx.Plan()

	expectedErr := "aws_instance.foo.1: the plan would destroy"
	if !strings.Contains(fmt.Sprintf("%s", err), expectedErr) {
		t.Fatalf("expected err would contain %q\nerr: %s\nplan: %s",
			expectedErr, err, plan)
	}
}

func TestContext2Plan_preventDestroy_countGood(t *testing.T) {
	m := testModule(t, "plan-prevent-destroy-count-good")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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
		},
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if plan.Diff.Empty() {
		t.Fatalf("Expected non-empty plan, got %s", plan.String())
	}
}

func TestContext2Plan_preventDestroy_countGoodNoChange(t *testing.T) {
	m := testModule(t, "plan-prevent-destroy-count-good")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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
		},
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if !plan.Diff.Empty() {
		t.Fatalf("Expected empty plan, got %s", plan.String())
	}
}

func TestContext2Plan_preventDestroy_destroyPlan(t *testing.T) {
	m := testModule(t, "plan-prevent-destroy-good")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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
								ID: "i-abc123",
							},
						},
					},
				},
			},
		},
		Destroy: true,
	})

	plan, err := ctx.Plan()

	expectedErr := "aws_instance.foo: the plan would destroy"
	if !strings.Contains(fmt.Sprintf("%s", err), expectedErr) {
		t.Fatalf("expected err would contain %q\nerr: %s\nplan: %s",
			expectedErr, err, plan)
	}
}

func TestContext2Plan_provisionerCycle(t *testing.T) {
	m := testModule(t, "plan-provisioner-cycle")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	pr := testProvisioner()
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ResourceProvisionerFactory{
			"local-exec": testProvisionerFuncFixed(pr),
		},
	})

	_, err := ctx.Plan()
	if err == nil {
		t.Fatalf("should error")
	}
}

func TestContext2Plan_computed(t *testing.T) {
	m := testModule(t, "plan-computed")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanComputedStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContext2Plan_computedDataResource(t *testing.T) {
	m := testModule(t, "plan-computed-data-resource")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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

	if got := len(plan.Diff.Modules); got != 1 {
		t.Fatalf("got %d modules; want 1", got)
	}

	moduleDiff := plan.Diff.Modules[0]

	if _, ok := moduleDiff.Resources["aws_instance.foo"]; !ok {
		t.Fatalf("missing diff for aws_instance.foo")
	}
	iDiff, ok := moduleDiff.Resources["data.aws_vpc.bar"]
	if !ok {
		t.Fatalf("missing diff for data.aws_vpc.bar")
	}

	expectedDiff := &InstanceDiff{
		Attributes: map[string]*ResourceAttrDiff{
			"id": {
				NewComputed: true,
				RequiresNew: true,
				Type:        DiffAttrOutput,
			},
		},
	}
	if same, _ := expectedDiff.Same(iDiff); !same {
		t.Fatalf(
			"incorrect diff for data.aws_vpc.bar\ngot:  %#v\nwant: %#v",
			iDiff, expectedDiff,
		)
	}
}

func TestContext2Plan_computedDataCountResource(t *testing.T) {
	m := testModule(t, "plan-computed-data-count")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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

	if got := len(plan.Diff.Modules); got != 1 {
		t.Fatalf("got %d modules; want 1", got)
	}

	moduleDiff := plan.Diff.Modules[0]

	// make sure we created 3 "bar"s
	for i := 0; i < 3; i++ {
		resource := fmt.Sprintf("data.aws_vpc.bar.%d", i)
		if _, ok := moduleDiff.Resources[resource]; !ok {
			t.Fatalf("missing diff for %s", resource)
		}
	}
}

func TestContext2Plan_localValueCount(t *testing.T) {
	m := testModule(t, "plan-local-value-count")
	p := testProvider("test")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"test": testProviderFuncFixed(p),
			},
		),
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if got := len(plan.Diff.Modules); got != 1 {
		t.Fatalf("got %d modules; want 1", got)
	}

	moduleDiff := plan.Diff.Modules[0]

	// make sure we created 3 "bar"s
	for i := 0; i < 3; i++ {
		resource := fmt.Sprintf("test_resource.foo.%d", i)
		if _, ok := moduleDiff.Resources[resource]; !ok {
			t.Fatalf("missing diff for %s", resource)
		}
	}
}

// Higher level test at TestResource_dataSourceListPlanPanic
func TestContext2Plan_dataSourceTypeMismatch(t *testing.T) {
	m := testModule(t, "plan-data-source-type-mismatch")
	p := testProvider("aws")
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
		Module: m,
		// Pretend like we ran a Refresh and the AZs data source was populated.
		State: &State{
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
		},
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	_, err := ctx.Plan()

	if err == nil {
		t.Fatalf("Expected err, got none!")
	}
	expected := "Expected ami to be string"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected:\n\n%s\n\nto contain:\n\n%s", err, expected)
	}
}

func TestContext2Plan_dataResourceBecomesComputed(t *testing.T) {
	m := testModule(t, "plan-data-resource-becomes-computed")
	p := testProvider("aws")

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
		Module: m,
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
		},
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if got := len(plan.Diff.Modules); got != 1 {
		t.Fatalf("got %d modules; want 1", got)
	}

	if !p.ReadDataDiffCalled {
		t.Fatal("ReadDataDiff wasn't called, but should've been")
	}
	if got, want := p.ReadDataDiffInfo.Id, "data.aws_data_resource.foo"; got != want {
		t.Fatalf("ReadDataDiff info id is %s; want %s", got, want)
	}

	moduleDiff := plan.Diff.Modules[0]

	iDiff, ok := moduleDiff.Resources["data.aws_data_resource.foo"]
	if !ok {
		t.Fatalf("missing diff for data.aws_data_resource.foo")
	}

	// This is added by the diff but we want to verify that we got
	// the same diff as above minus the dynamic stuff.
	delete(iDiff.Attributes, "id")

	if same, _ := p.ReadDataDiffReturn.Same(iDiff); !same {
		t.Fatalf(
			"incorrect diff for data.data_resource.foo\ngot:  %#v\nwant: %#v",
			iDiff, p.ReadDataDiffReturn,
		)
	}
}

func TestContext2Plan_computedList(t *testing.T) {
	m := testModule(t, "plan-computed-list")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanComputedListStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

// GH-8695. This tests that you can index into a computed list on a
// splatted resource.
func TestContext2Plan_computedMultiIndex(t *testing.T) {
	m := testModule(t, "plan-computed-multi-index")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanComputedMultiIndexStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContext2Plan_count(t *testing.T) {
	m := testModule(t, "plan-count")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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

	if len(plan.Diff.RootModule().Resources) < 6 {
		t.Fatalf("bad: %#v", plan.Diff.RootModule().Resources)
	}

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanCountStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContext2Plan_countComputed(t *testing.T) {
	m := testModule(t, "plan-count-computed")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	_, err := ctx.Plan()

	expectedErr := "aws_instance.bar: value of 'count'"
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
		Module: m,
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

	actual := strings.TrimSpace(plan.String())
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
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContext2Plan_countModuleStaticGrandchild(t *testing.T) {
	m := testModule(t, "plan-count-module-static-grandchild")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(`
DIFF:

module.child.child:
  CREATE: aws_instance.foo.0
  CREATE: aws_instance.foo.1
  CREATE: aws_instance.foo.2

STATE:

<no state>
`)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContext2Plan_countIndex(t *testing.T) {
	m := testModule(t, "plan-count-index")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanCountIndexStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContext2Plan_countIndexZero(t *testing.T) {
	m := testModule(t, "plan-count-index-zero")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanCountIndexZeroStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContext2Plan_countVar(t *testing.T) {
	m := testModule(t, "plan-count-var")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Variables: map[string]interface{}{
			"count": "3",
		},
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanCountVarStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContext2Plan_countZero(t *testing.T) {
	m := testModule(t, "plan-count-zero")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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

	actual := strings.TrimSpace(plan.String())
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
		Module: m,
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

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanCountOneIndexStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContext2Plan_countDecreaseToOne(t *testing.T) {
	m := testModule(t, "plan-count-dec")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	s := &State{
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
	}
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanCountDecreaseStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContext2Plan_countIncreaseFromNotSet(t *testing.T) {
	m := testModule(t, "plan-count-inc")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	s := &State{
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
	}
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanCountIncreaseStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContext2Plan_countIncreaseFromOne(t *testing.T) {
	m := testModule(t, "plan-count-inc")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	s := &State{
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
	}
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanCountIncreaseFromOneStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
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
	s := &State{
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
	}
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanCountIncreaseFromOneCorruptedStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
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
	p.DiffFn = testDiffFn
	s := &State{
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
	}
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(plan.String())
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
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContext2Plan_destroy(t *testing.T) {
	m := testModule(t, "plan-destroy")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	s := &State{
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
	}
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State:   s,
		Destroy: true,
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if len(plan.Diff.RootModule().Resources) != 2 {
		t.Fatalf("bad: %#v", plan.Diff.RootModule().Resources)
	}

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanDestroyStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContext2Plan_moduleDestroy(t *testing.T) {
	m := testModule(t, "plan-module-destroy")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	s := &State{
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
	}
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State:   s,
		Destroy: true,
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(plan.String())
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
	s := &State{
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
	}
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State:   s,
		Destroy: true,
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanModuleDestroyCycleStr)
	if actual != expected {
		t.Fatalf("bad:\n%s\n\nexpected:\n\n%s", actual, expected)
	}
}

func TestContext2Plan_moduleDestroyMultivar(t *testing.T) {
	m := testModule(t, "plan-module-destroy-multivar")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	s := &State{
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
	}
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State:   s,
		Destroy: true,
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(plan.String())
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
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanPathVarStr)

	// Warning: this ordering REALLY matters for this test. The
	// order is: cwd, module, root.
	expected = fmt.Sprintf(
		expected,
		cwd,
		m.Config().Dir,
		m.Config().Dir)

	if actual != expected {
		t.Fatalf("bad:\n%s\n\nexpected:\n\n%s", actual, expected)
	}
}

func TestContext2Plan_diffVar(t *testing.T) {
	m := testModule(t, "plan-diffvar")
	p := testProvider("aws")
	s := &State{
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
	}
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(plan.String())
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
		Module: m,
		Hooks:  []Hook{h},
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	_, err := ctx.Plan()
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

func TestContext2Plan_orphan(t *testing.T) {
	m := testModule(t, "plan-orphan")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	s := &State{
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
	}
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanOrphanStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

// This tests that configurations with UUIDs don't produce errors.
// For shadows, this would produce errors since a UUID changes every time.
func TestContext2Plan_shadowUuid(t *testing.T) {
	m := testModule(t, "plan-shadow-uuid")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	_, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestContext2Plan_state(t *testing.T) {
	m := testModule(t, "plan-good")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	s := &State{
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
	}
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if len(plan.Diff.RootModule().Resources) < 2 {
		t.Fatalf("bad: %#v", plan.Diff.RootModule().Resources)
	}

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanStateStr)
	if actual != expected {
		t.Fatalf("bad:\n%s\n\nexpected:\n\n%s", actual, expected)
	}
}

func TestContext2Plan_taint(t *testing.T) {
	m := testModule(t, "plan-taint")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	s := &State{
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
	}
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanTaintStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContext2Apply_taintIgnoreChanges(t *testing.T) {
	m := testModule(t, "plan-taint-ignore-changes")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	s := &State{
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
	}
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanTaintIgnoreChangesStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

// Fails about 50% of the time before the fix for GH-4982, covers the fix.
func TestContext2Plan_taintDestroyInterpolatedCountRace(t *testing.T) {
	m := testModule(t, "plan-taint-interpolated-count")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	s := &State{
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
	}
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	for i := 0; i < 100; i++ {
		plan, err := ctx.Plan()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		actual := strings.TrimSpace(plan.String())
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
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Targets: []string{"aws_instance.foo"},
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(plan.String())
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
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Targets: []string{"module.B"},
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(plan.String())
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
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"null": testProviderFuncFixed(p),
			},
		),
		Targets: []string{"module.child2"},
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(plan.String())
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
		Module: m,
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
		},
		Destroy: true,
		Targets: []string{"aws_instance.orphan"},
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(plan.String())
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
		Module: m,
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
		},
		Destroy: true,
		Targets: []string{"module.child.aws_instance.orphan"},
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(plan.String())
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
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Targets: []string{"aws_instance.blue", "module.blue_mod"},
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(plan.String())
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
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: &State{
			Modules: []*ModuleState{
				&ModuleState{
					Path:      rootModulePath,
					Resources: resources,
				},
			},
		},
		Targets: []string{"aws_instance.foo[1]"},
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(plan.String())
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
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Variables: map[string]interface{}{
			"foo": "bar",
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
		Module: m,
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
	p.DiffFn = testDiffFn
	s := &State{
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
	}
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Variables: map[string]interface{}{
			"foo": "ami-1234abcd",
		},
		State: s,
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if len(plan.Diff.RootModule().Resources) < 1 {
		t.Fatalf("bad: %#v", plan.Diff.RootModule().Resources)
	}

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanIgnoreChangesStr)
	if actual != expected {
		t.Fatalf("bad:\n%s\n\nexpected\n\n%s", actual, expected)
	}
}

func TestContext2Plan_ignoreChangesWildcard(t *testing.T) {
	m := testModule(t, "plan-ignore-changes-wildcard")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	s := &State{
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
	}
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Variables: map[string]interface{}{
			"foo": "ami-1234abcd",
			"bar": "t2.small",
		},
		State: s,
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if len(plan.Diff.RootModule().Resources) > 0 {
		t.Fatalf("bad: %#v", plan.Diff.RootModule().Resources)
	}

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanIgnoreChangesWildcardStr)
	if actual != expected {
		t.Fatalf("bad:\n%s\n\nexpected\n\n%s", actual, expected)
	}
}

func TestContext2Plan_moduleMapLiteral(t *testing.T) {
	m := testModule(t, "plan-module-map-literal")
	p := testProvider("aws")
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
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestContext2Plan_computedValueInMap(t *testing.T) {
	m := testModule(t, "plan-computed-value-in-map")
	p := testProvider("aws")
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
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanComputedValueInMap)
	if actual != expected {
		t.Fatalf("bad:\n%s\n\nexpected\n\n%s", actual, expected)
	}
}

func TestContext2Plan_moduleVariableFromSplat(t *testing.T) {
	m := testModule(t, "plan-module-variable-from-splat")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanModuleVariableFromSplat)
	if actual != expected {
		t.Fatalf("bad:\n%s\n\nexpected\n\n%s", actual, expected)
	}
}

func TestContext2Plan_createBeforeDestroy_depends_datasource(t *testing.T) {
	m := testModule(t, "plan-cdb-depends-datasource")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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

	if got := len(plan.Diff.Modules); got != 1 {
		t.Fatalf("got %d modules; want 1", got)
	}

	moduleDiff := plan.Diff.Modules[0]

	if _, ok := moduleDiff.Resources["aws_instance.foo.0"]; !ok {
		t.Fatalf("missing diff for aws_instance.foo.0")
	}
	if _, ok := moduleDiff.Resources["aws_instance.foo.1"]; !ok {
		t.Fatalf("missing diff for aws_instance.foo.1")
	}
	if _, ok := moduleDiff.Resources["data.aws_vpc.bar.0"]; !ok {
		t.Fatalf("missing diff for data.aws_vpc.bar.0")
	}
	if _, ok := moduleDiff.Resources["data.aws_vpc.bar.1"]; !ok {
		t.Fatalf("missing diff for data.aws_vpc.bar.1")
	}
}

// interpolated lists need to be stored in the original order.
func TestContext2Plan_listOrder(t *testing.T) {
	m := testModule(t, "plan-list-order")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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

	rDiffs := plan.Diff.Modules[0].Resources
	rDiffA := rDiffs["aws_instance.a"]
	rDiffB := rDiffs["aws_instance.b"]

	if !rDiffA.Equal(rDiffB) {
		t.Fatal("aws_instance.a and aws_instance.b diffs should match:\n", plan)
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
	s := &State{
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
	}
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(plan.Diff.String())
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
	s := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.foo.0": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo0",
							Attributes: map[string]string{
								"id": "foo0",
							},
						},
					},
					"aws_instance.foo.1": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo1",
							Attributes: map[string]string{
								"id": "foo1",
							},
						},
					},
					"aws_instance.bar.0": &ResourceState{
						Type:         "aws_instance",
						Dependencies: []string{"aws_instance.foo.*"},
						Primary: &InstanceState{
							ID: "bar0",
							Attributes: map[string]string{
								"id": "bar0",
							},
						},
					},
					"aws_instance.bar.1": &ResourceState{
						Type:         "aws_instance",
						Dependencies: []string{"aws_instance.foo.*"},
						Primary: &InstanceState{
							ID: "bar1",
							Attributes: map[string]string{
								"id": "bar1",
							},
						},
					},
					"aws_instance.baz.0": &ResourceState{
						Type:         "aws_instance",
						Dependencies: []string{"aws_instance.bar.*"},
						Primary: &InstanceState{
							ID: "baz0",
							Attributes: map[string]string{
								"id": "baz0",
							},
						},
					},
					"aws_instance.baz.1": &ResourceState{
						Type:         "aws_instance",
						Dependencies: []string{"aws_instance.bar.*"},
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
	}
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	w, e := ctx.Validate()
	if len(w) > 0 {
		t.Fatalf("warnings generated on validate: %#v", w)
	}
	if len(e) > 0 {
		t.Fatalf("errors generated on validate: %#v", e)
	}

	_, err := ctx.Refresh()
	if err != nil {
		t.Fatalf("refresh err: %s", err)
	}

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("plan err: %s", err)
	}

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(`
DIFF:



STATE:

aws_instance.bar.0:
  ID = bar0

  Dependencies:
    aws_instance.foo.*
aws_instance.bar.1:
  ID = bar1

  Dependencies:
    aws_instance.foo.*
aws_instance.baz.0:
  ID = baz0

  Dependencies:
    aws_instance.bar.*
aws_instance.baz.1:
  ID = baz1

  Dependencies:
    aws_instance.bar.*
aws_instance.foo.0:
  ID = foo0
aws_instance.foo.1:
  ID = foo1
`)
	if actual != expected {
		t.Fatalf("bad:\n%s\n\nexpected\n\n%s", actual, expected)
	}
}

func TestContext2Plan_invalidOutput(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
data "aws_instance" "name" {}

output "out" {
  value = "${data.aws_instance.name.missing}"
}`,
	})

	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	// if this ever fails to pass validate, add a resource to reference in the config
	w, e := ctx.Validate()
	if len(w) > 0 {
		t.Fatalf("warnings generated on validate: %#v", w)
	}
	if len(e) > 0 {
		t.Fatalf("errors generated on validate: %v", e)
	}

	_, err := ctx.Refresh()
	if err != nil {
		t.Fatalf("refresh err: %s", err)
	}

	_, err = ctx.Plan()
	switch {
	case featureOutputErrors:
		if err == nil {
			t.Fatal("expected error")
		}
	default:
		if err != nil {
			t.Fatalf("plan err: %s", err)
		}
	}
}

func TestContext2Plan_invalidModuleOutput(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"child/main.tf": `
data "aws_instance" "name" {}

output "out" {
  value = "${data.aws_instance.name.missing}"
}`,
		"main.tf": `
module "child" {
  source = "./child"
}

resource "aws_instance" "foo" {
  foo = "${module.child.out}"
}`,
	})

	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	// if this ever fails to pass validate, add a resource to reference in the config
	w, e := ctx.Validate()
	if len(w) > 0 {
		t.Fatalf("warnings generated on validate: %#v", w)
	}
	if len(e) > 0 {
		t.Fatalf("errors generated on validate: %v", e)
	}

	_, err := ctx.Refresh()
	if err != nil {
		t.Fatalf("refresh err: %s", err)
	}

	_, err = ctx.Plan()
	switch {
	case featureOutputErrors:
		if err == nil {
			t.Fatal("expected error")
		}
	default:
		if err != nil {
			t.Fatalf("plan err: %s", err)
		}
	}

}
