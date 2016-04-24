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

func TestContext2Plan(t *testing.T) {
	m := testModule(t, "plan-good")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if len(plan.Diff.RootModule().Resources) < 2 {
		t.Fatalf("bad: %#v", plan.Diff.RootModule().Resources)
	}

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContext2Plan_createBefore_maintainRoot(t *testing.T) {
	m := testModule(t, "plan-cbd-maintain-root")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		Variables: map[string]string{
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
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
			Providers: map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
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
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		Variables: map[string]string{
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
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		Providers: map[string]ResourceProviderFactory{
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
					calls = append(calls, v.(string))
					return testDiffFn(info, state, c)
				}
				return p, nil
			},
		},
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

func TestContext2Plan_moduleProviderDefaults(t *testing.T) {
	var l sync.Mutex
	var calls []string
	toCount := 0

	m := testModule(t, "plan-module-provider-defaults")
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": func() (ResourceProvider, error) {
				l.Lock()
				defer l.Unlock()

				p := testProvider("aws")
				p.ConfigureFn = func(c *ResourceConfig) error {
					if v, ok := c.Get("from"); !ok || v.(string) != "root" {
						return fmt.Errorf("bad")
					}
					if v, ok := c.Get("to"); ok && v.(string) == "child" {
						toCount++
					}

					return nil
				}
				p.DiffFn = func(
					info *InstanceInfo,
					state *InstanceState,
					c *ResourceConfig) (*InstanceDiff, error) {
					v, _ := c.Get("from")
					calls = append(calls, v.(string))
					return testDiffFn(info, state, c)
				}
				return p, nil
			},
		},
	})

	_, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if toCount != 1 {
		t.Fatalf(
			"provider in child didn't set proper config\n\n"+
				"toCount: %d", toCount)
	}

	actual := calls
	sort.Strings(actual)
	expected := []string{"child", "root"}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestContext2Plan_moduleProviderDefaultsVar(t *testing.T) {
	var l sync.Mutex
	var calls []string

	m := testModule(t, "plan-module-provider-defaults-var")
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
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

					calls = append(calls, buf.String())
					return nil
				}
				p.DiffFn = testDiffFn
				return p, nil
			},
		},
		Variables: map[string]string{
			"foo": "root",
		},
	})

	_, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := []string{
		"root\n",
		"root\nchild\n",
	}
	if !reflect.DeepEqual(calls, expected) {
		t.Fatalf("BAD: %#v", calls)
	}
}

func TestContext2Plan_moduleVar(t *testing.T) {
	m := testModule(t, "plan-module-var")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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

func TestContext2Plan_moduleVarWrongType(t *testing.T) {
	m := testModule(t, "plan-module-wrong-var-type")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	_, err := ctx.Plan()
	if err == nil {
		t.Fatalf("should error")
	}
}

func TestContext2Plan_moduleVarWrongTypeNested(t *testing.T) {
	m := testModule(t, "plan-module-wrong-var-type-nested")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		Providers: map[string]ResourceProviderFactory{
			"null": testProviderFuncFixed(p),
		},
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
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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

func TestContext2Plan_preventDestroy_destroyPlan(t *testing.T) {
	m := testModule(t, "plan-prevent-destroy-good")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
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

func TestContext2Plan_computed(t *testing.T) {
	m := testModule(t, "plan-computed")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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

func TestContext2Plan_computedList(t *testing.T) {
	m := testModule(t, "plan-computed-list")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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

func TestContext2Plan_count(t *testing.T) {
	m := testModule(t, "plan-count")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	_, err := ctx.Plan()
	if err == nil {
		t.Fatal("should error")
	}
}

func TestContext2Plan_countIndex(t *testing.T) {
	m := testModule(t, "plan-count-index")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		Variables: map[string]string{
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
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanCountZeroStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContext2Plan_countOneIndex(t *testing.T) {
	m := testModule(t, "plan-count-one-index")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		t.Fatalf("bad:\n%s", actual)
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
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		t.Fatalf("bad:\n%s", actual)
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
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		t.Fatalf("bad:\n%s", actual)
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
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
						Tainted: []*InstanceState{
							&InstanceState{
								ID: "baz",
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

func TestContext2Plan_multiple_taint(t *testing.T) {
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
						Tainted: []*InstanceState{
							&InstanceState{
								ID: "baz",
							},
							&InstanceState{
								ID: "zip",
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
		State: s,
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(plan.String())
	expected := strings.TrimSpace(testTerraformPlanMultipleTaintStr)
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
						Tainted: []*InstanceState{
							&InstanceState{ID: "bar"},
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
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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

STATE:

aws_instance.foo.0: (1 tainted)
  ID = <not created>
  Tainted ID 1 = bar
aws_instance.foo.1:
  ID = bar
aws_instance.foo.2:
  ID = bar
		`)
		if actual != expected {
			t.Fatalf("bad:\n%s", actual)
		}
	}
}

func TestContext2Plan_targeted(t *testing.T) {
	m := testModule(t, "plan-targeted")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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

func TestContext2Plan_targetedOrphan(t *testing.T) {
	m := testModule(t, "plan-targeted-orphan")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
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
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
aws_instance.foo.10:
  ID = i-abc10
aws_instance.foo.11:
  ID = i-abc11
aws_instance.foo.12:
  ID = i-abc12
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
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		Variables: map[string]string{
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
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		Variables: map[string]string{
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
