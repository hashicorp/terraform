package terraform

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform/config/module"
)

func TestContext2Apply(t *testing.T) {
	m := testModule(t, "apply-good")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	mod := state.RootModule()
	if len(mod.Resources) < 2 {
		t.Fatalf("bad: %#v", mod.Resources)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContext2Apply_providerAlias(t *testing.T) {
	m := testModule(t, "apply-provider-alias")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	mod := state.RootModule()
	if len(mod.Resources) < 2 {
		t.Fatalf("bad: %#v", mod.Resources)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyProviderAliasStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

// GH-2870
func TestContext2Apply_providerWarning(t *testing.T) {
	m := testModule(t, "apply-provider-warning")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	p.ValidateFn = func(c *ResourceConfig) (ws []string, es []error) {
		ws = append(ws, "Just a warning")
		return
	}
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(`
aws_instance.foo:
  ID = foo
	`)
	if actual != expected {
		t.Fatalf("got: \n%s\n\nexpected:\n%s", actual, expected)
	}

	if !p.ConfigureCalled {
		t.Fatalf("provider Configure() was never called!")
	}
}

func TestContext2Apply_emptyModule(t *testing.T) {
	m := testModule(t, "apply-empty-module")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	actual = strings.Replace(actual, "  ", "", -1)
	expected := strings.TrimSpace(testTerraformApplyEmptyModuleStr)
	if actual != expected {
		t.Fatalf("bad: \n%s\nexpect:\n%s", actual, expected)
	}
}

func TestContext2Apply_createBeforeDestroy(t *testing.T) {
	m := testModule(t, "apply-good-create-before")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.bar": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
							Attributes: map[string]string{
								"require_new": "abc",
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

	if p, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	} else {
		t.Logf(p.String())
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	mod := state.RootModule()
	if len(mod.Resources) != 1 {
		t.Fatalf("bad: %s", state)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyCreateBeforeStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContext2Apply_createBeforeDestroyUpdate(t *testing.T) {
	m := testModule(t, "apply-good-create-before-update")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.bar": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
							Attributes: map[string]string{
								"foo": "bar",
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

	if p, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	} else {
		t.Logf(p.String())
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	mod := state.RootModule()
	if len(mod.Resources) != 1 {
		t.Fatalf("bad: %s", state)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyCreateBeforeUpdateStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContext2Apply_destroyComputed(t *testing.T) {
	m := testModule(t, "apply-destroy-computed")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.foo": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
							Attributes: map[string]string{
								"output": "value",
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
		State:   state,
		Destroy: true,
	})

	if p, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	} else {
		t.Logf(p.String())
	}

	if _, err := ctx.Apply(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

// https://github.com/hashicorp/terraform/pull/5096
func TestContext2Apply_destroySkipsCBD(t *testing.T) {
	// Config contains CBD resource depending on non-CBD resource, which triggers
	// a cycle if they are both replaced, but should _not_ trigger a cycle when
	// just doing a `terraform destroy`.
	m := testModule(t, "apply-destroy-cbd")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	state := &State{
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
					"aws_instance.bar": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
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
		State:   state,
		Destroy: true,
	})

	if p, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	} else {
		t.Logf(p.String())
	}

	if _, err := ctx.Apply(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

// https://github.com/hashicorp/terraform/issues/2892
func TestContext2Apply_destroyCrossProviders(t *testing.T) {
	m := testModule(t, "apply-destroy-cross-providers")

	p_aws := testProvider("aws")
	p_aws.ApplyFn = testApplyFn
	p_aws.DiffFn = testDiffFn

	p_tf := testProvider("terraform")
	p_tf.ApplyFn = testApplyFn
	p_tf.DiffFn = testDiffFn

	providers := map[string]ResourceProviderFactory{
		"aws":       testProviderFuncFixed(p_aws),
		"terraform": testProviderFuncFixed(p_tf),
	}

	// Bug only appears from time to time,
	// so we run this test multiple times
	// to check for the race-condition
	for i := 0; i <= 10; i++ {
		ctx := getContextForApply_destroyCrossProviders(
			t, m, providers)

		if p, err := ctx.Plan(); err != nil {
			t.Fatalf("err: %s", err)
		} else {
			t.Logf(p.String())
		}

		if _, err := ctx.Apply(); err != nil {
			t.Fatalf("err: %s", err)
		}
	}
}

func getContextForApply_destroyCrossProviders(
	t *testing.T,
	m *module.Tree,
	providers map[string]ResourceProviderFactory) *Context {
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"terraform_remote_state.shared": &ResourceState{
						Type: "terraform_remote_state",
						Primary: &InstanceState{
							ID: "remote-2652591293",
							Attributes: map[string]string{
								"output.env_name": "test",
							},
						},
					},
				},
			},
			&ModuleState{
				Path: []string{"root", "example"},
				Resources: map[string]*ResourceState{
					"aws_vpc.bar": &ResourceState{
						Type: "aws_vpc",
						Primary: &InstanceState{
							ID: "vpc-aaabbb12",
							Attributes: map[string]string{
								"value": "test",
							},
						},
					},
				},
			},
		},
	}
	ctx := testContext2(t, &ContextOpts{
		Module:    m,
		Providers: providers,
		State:     state,
		Destroy:   true,
	})

	return ctx
}

func TestContext2Apply_minimal(t *testing.T) {
	m := testModule(t, "apply-minimal")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	if _, err := ctx.Plan(); err != nil {
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

func TestContext2Apply_badDiff(t *testing.T) {
	m := testModule(t, "apply-good")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	p.DiffFn = func(*InstanceInfo, *InstanceState, *ResourceConfig) (*InstanceDiff, error) {
		return &InstanceDiff{
			Attributes: map[string]*ResourceAttrDiff{
				"newp": nil,
			},
		}, nil
	}

	if _, err := ctx.Apply(); err == nil {
		t.Fatal("should error")
	}
}

func TestContext2Apply_cancel(t *testing.T) {
	stopped := false

	m := testModule(t, "apply-cancel")
	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	p.ApplyFn = func(*InstanceInfo, *InstanceState, *InstanceDiff) (*InstanceState, error) {
		if !stopped {
			stopped = true
			go ctx.Stop()

			for {
				if ctx.sh.Stopped() {
					break
				}
			}
		}

		return &InstanceState{
			ID: "foo",
			Attributes: map[string]string{
				"num": "2",
			},
		}, nil
	}
	p.DiffFn = func(*InstanceInfo, *InstanceState, *ResourceConfig) (*InstanceDiff, error) {
		return &InstanceDiff{
			Attributes: map[string]*ResourceAttrDiff{
				"num": &ResourceAttrDiff{
					New: "bar",
				},
			},
		}, nil
	}

	if _, err := ctx.Plan(); err != nil {
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

	mod := state.RootModule()
	if len(mod.Resources) != 1 {
		t.Fatalf("bad: %s", state.String())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyCancelStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContext2Apply_compute(t *testing.T) {
	m := testModule(t, "apply-compute")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	if _, err := ctx.Plan(); err != nil {
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

func TestContext2Apply_countDecrease(t *testing.T) {
	m := testModule(t, "apply-count-dec")
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
							Attributes: map[string]string{
								"foo":  "foo",
								"type": "aws_instance",
							},
						},
					},
					"aws_instance.foo.2": &ResourceState{
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

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyCountDecStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContext2Apply_countDecreaseToOne(t *testing.T) {
	m := testModule(t, "apply-count-dec-one")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
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

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyCountDecToOneStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

// https://github.com/PeoplePerHour/terraform/pull/11
//
// This tests a case where both a "resource" and "resource.0" are in
// the state file, which apparently is a reasonable backwards compatibility
// concern found in the above 3rd party repo.
func TestContext2Apply_countDecreaseToOneCorrupted(t *testing.T) {
	m := testModule(t, "apply-count-dec-one")
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
							ID: "baz",
							Attributes: map[string]string{
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

	if p, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	} else {
		testStringMatch(t, p, testTerraformApplyCountDecToOneCorruptedPlanStr)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyCountDecToOneCorruptedStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContext2Apply_countTainted(t *testing.T) {
	m := testModule(t, "apply-count-tainted")
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
							&InstanceState{
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
		},
	}
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		State: s,
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyCountTaintedStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContext2Apply_countVariable(t *testing.T) {
	m := testModule(t, "apply-count-variable")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyCountVariableStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContext2Apply_mapVariableOverride(t *testing.T) {
	m := testModule(t, "apply-map-var-override")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		Variables: map[string]string{
			"images.us-west-2": "overridden",
		},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(`
aws_instance.bar:
  ID = foo
  ami = overridden
  type = aws_instance
aws_instance.foo:
  ID = foo
  ami = image-1234
  type = aws_instance
	`)
	if actual != expected {
		t.Fatalf("got: \n%s\nexpected: \n%s", actual, expected)
	}
}

func TestContext2Apply_module(t *testing.T) {
	m := testModule(t, "apply-module")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyModuleStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContext2Apply_moduleDestroyOrder(t *testing.T) {
	m := testModule(t, "apply-module-destroy-order")
	p := testProvider("aws")
	p.DiffFn = testDiffFn

	// Create a custom apply function to track the order they were destroyed
	var order []string
	var orderLock sync.Mutex
	p.ApplyFn = func(
		info *InstanceInfo,
		is *InstanceState,
		id *InstanceDiff) (*InstanceState, error) {
		orderLock.Lock()
		defer orderLock.Unlock()

		order = append(order, is.ID)
		return nil, nil
	}

	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.b": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "b",
						},
					},
				},
			},

			&ModuleState{
				Path: []string{"root", "child"},
				Resources: map[string]*ResourceState{
					"aws_instance.a": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "a",
						},
					},
				},
				Outputs: map[string]string{
					"a_output": "a",
				},
			},
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		State:   state,
		Destroy: true,
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := []string{"b", "a"}
	if !reflect.DeepEqual(order, expected) {
		t.Fatalf("bad: %#v", order)
	}

	{
		actual := strings.TrimSpace(state.String())
		expected := strings.TrimSpace(testTerraformApplyModuleDestroyOrderStr)
		if actual != expected {
			t.Fatalf("bad: \n%s", actual)
		}
	}
}

func TestContext2Apply_moduleOrphanProvider(t *testing.T) {
	m := testModule(t, "apply-module-orphan-provider-inherit")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	p.ConfigureFn = func(c *ResourceConfig) error {
		if _, ok := c.Get("value"); !ok {
			return fmt.Errorf("value is not found")
		}

		return nil
	}

	// Create a state with an orphan module
	state := &State{
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
	}

	ctx := testContext2(t, &ContextOpts{
		Module: m,
		State:  state,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	if _, err := ctx.Apply(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestContext2Apply_moduleGrandchildProvider(t *testing.T) {
	m := testModule(t, "apply-module-grandchild-provider-inherit")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	var callLock sync.Mutex
	called := false
	p.ConfigureFn = func(c *ResourceConfig) error {
		if _, ok := c.Get("value"); !ok {
			return fmt.Errorf("value is not found")
		}
		callLock.Lock()
		called = true
		callLock.Unlock()

		return nil
	}

	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	if _, err := ctx.Apply(); err != nil {
		t.Fatalf("err: %s", err)
	}

	callLock.Lock()
	defer callLock.Unlock()
	if called != true {
		t.Fatalf("err: configure never called")
	}
}

// This tests an issue where all the providers in a module but not
// in the root weren't being added to the root properly. In this test
// case: aws is explicitly added to root, but "test" should be added to.
// With the bug, it wasn't.
func TestContext2Apply_moduleOnlyProvider(t *testing.T) {
	m := testModule(t, "apply-module-only-provider")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	pTest := testProvider("test")
	pTest.ApplyFn = testApplyFn
	pTest.DiffFn = testDiffFn

	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws":  testProviderFuncFixed(p),
			"test": testProviderFuncFixed(pTest),
		},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyModuleOnlyProviderStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContext2Apply_moduleProviderAlias(t *testing.T) {
	m := testModule(t, "apply-module-provider-alias")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyModuleProviderAliasStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContext2Apply_moduleProviderAliasTargets(t *testing.T) {
	m := testModule(t, "apply-module-provider-alias")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		Targets: []string{"no.thing"},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(`
<no state>
	`)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContext2Apply_moduleProviderCloseNested(t *testing.T) {
	m := testModule(t, "apply-module-provider-close-nested")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		State: &State{
			Modules: []*ModuleState{
				&ModuleState{
					Path: []string{"root", "child", "subchild"},
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
		Destroy: true,
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	if _, err := ctx.Apply(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestContext2Apply_moduleVarResourceCount(t *testing.T) {
	m := testModule(t, "apply-module-var-resource-count")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		Variables: map[string]string{
			"count": "2",
		},
		Destroy: true,
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	if _, err := ctx.Apply(); err != nil {
		t.Fatalf("err: %s", err)
	}

	ctx = testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		Variables: map[string]string{
			"count": "5",
		},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	if _, err := ctx.Apply(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

// GH-819
func TestContext2Apply_moduleBool(t *testing.T) {
	m := testModule(t, "apply-module-bool")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyModuleBoolStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContext2Apply_multiProvider(t *testing.T) {
	m := testModule(t, "apply-multi-provider")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	pDO := testProvider("do")
	pDO.ApplyFn = testApplyFn
	pDO.DiffFn = testDiffFn

	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
			"do":  testProviderFuncFixed(pDO),
		},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	mod := state.RootModule()
	if len(mod.Resources) < 2 {
		t.Fatalf("bad: %#v", mod.Resources)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyMultiProviderStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContext2Apply_multiVar(t *testing.T) {
	m := testModule(t, "apply-multi-var")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	// First, apply with a count of 3
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		Variables: map[string]string{
			"count": "3",
		},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := state.RootModule().Outputs["output"]
	expected := "bar0,bar1,bar2"
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}

	// Apply again, reduce the count to 1
	{
		ctx := testContext2(t, &ContextOpts{
			Module: m,
			State:  state,
			Providers: map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
			Variables: map[string]string{
				"count": "1",
			},
		})

		if _, err := ctx.Plan(); err != nil {
			t.Fatalf("err: %s", err)
		}

		state, err := ctx.Apply()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		actual := state.RootModule().Outputs["output"]
		expected := "bar0"
		if actual != expected {
			t.Fatalf("bad: \n%s", actual)
		}
	}
}

func TestContext2Apply_nilDiff(t *testing.T) {
	m := testModule(t, "apply-good")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	p.DiffFn = func(*InstanceInfo, *InstanceState, *ResourceConfig) (*InstanceDiff, error) {
		return nil, nil
	}

	if _, err := ctx.Apply(); err == nil {
		t.Fatal("should error")
	}
}

func TestContext2Apply_outputOrphan(t *testing.T) {
	m := testModule(t, "apply-output-orphan")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Outputs: map[string]string{
					"foo": "bar",
					"bar": "baz",
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

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyOutputOrphanStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContext2Apply_providerComputedVar(t *testing.T) {
	m := testModule(t, "apply-provider-computed")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	pTest := testProvider("test")
	pTest.ApplyFn = testApplyFn
	pTest.DiffFn = testDiffFn

	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws":  testProviderFuncFixed(p),
			"test": testProviderFuncFixed(pTest),
		},
	})

	p.ConfigureFn = func(c *ResourceConfig) error {
		if c.IsComputed("value") {
			return fmt.Errorf("value is computed")
		}

		v, ok := c.Get("value")
		if !ok {
			return fmt.Errorf("value is not found")
		}
		if v != "yes" {
			return fmt.Errorf("value is not 'yes': %v", v)
		}

		return nil
	}

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	if _, err := ctx.Apply(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestContext2Apply_Provisioner_compute(t *testing.T) {
	m := testModule(t, "apply-provisioner-compute")
	p := testProvider("aws")
	pr := testProvisioner()
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	pr.ApplyFn = func(rs *InstanceState, c *ResourceConfig) error {
		val, ok := c.Config["foo"]
		if !ok || val != "computed_dynamical" {
			t.Fatalf("bad value for foo: %v %#v", val, c)
		}

		return nil
	}
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		Provisioners: map[string]ResourceProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
		Variables: map[string]string{
			"value": "1",
		},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyProvisionerStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}

	// Verify apply was invoked
	if !pr.ApplyCalled {
		t.Fatalf("provisioner not invoked")
	}
}

func TestContext2Apply_provisionerCreateFail(t *testing.T) {
	m := testModule(t, "apply-provisioner-fail-create")
	p := testProvider("aws")
	pr := testProvisioner()
	p.DiffFn = testDiffFn

	p.ApplyFn = func(
		info *InstanceInfo,
		is *InstanceState,
		id *InstanceDiff) (*InstanceState, error) {
		is.ID = "foo"
		return is, fmt.Errorf("error")
	}

	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		Provisioners: map[string]ResourceProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err == nil {
		t.Fatal("should error")
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyProvisionerFailCreateStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContext2Apply_provisionerCreateFailNoId(t *testing.T) {
	m := testModule(t, "apply-provisioner-fail-create")
	p := testProvider("aws")
	pr := testProvisioner()
	p.DiffFn = testDiffFn

	p.ApplyFn = func(
		info *InstanceInfo,
		is *InstanceState,
		id *InstanceDiff) (*InstanceState, error) {
		return nil, fmt.Errorf("error")
	}

	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		Provisioners: map[string]ResourceProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err == nil {
		t.Fatal("should error")
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyProvisionerFailCreateNoIdStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContext2Apply_provisionerFail(t *testing.T) {
	m := testModule(t, "apply-provisioner-fail")
	p := testProvider("aws")
	pr := testProvisioner()
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	pr.ApplyFn = func(*InstanceState, *ResourceConfig) error {
		return fmt.Errorf("EXPLOSION")
	}

	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		Provisioners: map[string]ResourceProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
		Variables: map[string]string{
			"value": "1",
		},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err == nil {
		t.Fatal("should error")
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyProvisionerFailStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContext2Apply_provisionerFail_createBeforeDestroy(t *testing.T) {
	m := testModule(t, "apply-provisioner-fail-create-before")
	p := testProvider("aws")
	pr := testProvisioner()
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	pr.ApplyFn = func(*InstanceState, *ResourceConfig) error {
		return fmt.Errorf("EXPLOSION")
	}

	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.bar": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
							Attributes: map[string]string{
								"require_new": "abc",
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
		Provisioners: map[string]ResourceProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
		State: state,
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err == nil {
		t.Fatal("should error")
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyProvisionerFailCreateBeforeDestroyStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContext2Apply_error_createBeforeDestroy(t *testing.T) {
	m := testModule(t, "apply-error-create-before")
	p := testProvider("aws")
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.bar": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
							Attributes: map[string]string{
								"require_new": "abc",
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
	p.ApplyFn = func(info *InstanceInfo, is *InstanceState, id *InstanceDiff) (*InstanceState, error) {
		return nil, fmt.Errorf("error")
	}
	p.DiffFn = testDiffFn

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err == nil {
		t.Fatal("should have error")
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyErrorCreateBeforeDestroyStr)
	if actual != expected {
		t.Fatalf("bad: \n%s\n\nExpected:\n\n%s", actual, expected)
	}
}

func TestContext2Apply_errorDestroy_createBeforeDestroy(t *testing.T) {
	m := testModule(t, "apply-error-create-before")
	p := testProvider("aws")
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.bar": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
							Attributes: map[string]string{
								"require_new": "abc",
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
	p.ApplyFn = func(info *InstanceInfo, is *InstanceState, id *InstanceDiff) (*InstanceState, error) {
		// Fail the destroy!
		if id.Destroy {
			return is, fmt.Errorf("error")
		}

		// Create should work
		is = &InstanceState{
			ID: "foo",
		}
		return is, nil
	}
	p.DiffFn = testDiffFn

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err == nil {
		t.Fatal("should have error")
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyErrorDestroyCreateBeforeDestroyStr)
	if actual != expected {
		t.Fatalf("bad: actual:\n%s\n\nexpected:\n%s", actual, expected)
	}
}

func TestContext2Apply_multiDepose_createBeforeDestroy(t *testing.T) {
	m := testModule(t, "apply-multi-depose-create-before-destroy")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	ps := map[string]ResourceProviderFactory{"aws": testProviderFuncFixed(p)}
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.web": &ResourceState{
						Type:    "aws_instance",
						Primary: &InstanceState{ID: "foo"},
					},
				},
			},
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Module:    m,
		Providers: ps,
		State:     state,
	})
	createdInstanceId := "bar"
	// Create works
	createFunc := func(is *InstanceState) (*InstanceState, error) {
		return &InstanceState{ID: createdInstanceId}, nil
	}
	// Destroy starts broken
	destroyFunc := func(is *InstanceState) (*InstanceState, error) {
		return is, fmt.Errorf("destroy failed")
	}
	p.ApplyFn = func(info *InstanceInfo, is *InstanceState, id *InstanceDiff) (*InstanceState, error) {
		if id.Destroy {
			return destroyFunc(is)
		} else {
			return createFunc(is)
		}
	}

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Destroy is broken, so even though CBD successfully replaces the instance,
	// we'll have to save the Deposed instance to destroy later
	state, err := ctx.Apply()
	if err == nil {
		t.Fatal("should have error")
	}

	checkStateString(t, state, `
aws_instance.web: (1 deposed)
  ID = bar
  Deposed ID 1 = foo
	`)

	createdInstanceId = "baz"
	ctx = testContext2(t, &ContextOpts{
		Module:    m,
		Providers: ps,
		State:     state,
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	// We're replacing the primary instance once again. Destroy is _still_
	// broken, so the Deposed list gets longer
	state, err = ctx.Apply()
	if err == nil {
		t.Fatal("should have error")
	}

	checkStateString(t, state, `
aws_instance.web: (2 deposed)
  ID = baz
  Deposed ID 1 = foo
  Deposed ID 2 = bar
	`)

	// Destroy partially fixed!
	destroyFunc = func(is *InstanceState) (*InstanceState, error) {
		if is.ID == "foo" || is.ID == "baz" {
			return nil, nil
		} else {
			return is, fmt.Errorf("destroy partially failed")
		}
	}

	createdInstanceId = "qux"
	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}
	state, err = ctx.Apply()
	// Expect error because 1/2 of Deposed destroys failed
	if err == nil {
		t.Fatal("should have error")
	}

	// foo and baz are now gone, bar sticks around
	checkStateString(t, state, `
aws_instance.web: (1 deposed)
  ID = qux
  Deposed ID 1 = bar
	`)

	// Destroy working fully!
	destroyFunc = func(is *InstanceState) (*InstanceState, error) {
		return nil, nil
	}

	createdInstanceId = "quux"
	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}
	state, err = ctx.Apply()
	if err != nil {
		t.Fatal("should not have error:", err)
	}

	// And finally the state is clean
	checkStateString(t, state, `
aws_instance.web:
  ID = quux
	`)
}

func TestContext2Apply_provisionerResourceRef(t *testing.T) {
	m := testModule(t, "apply-provisioner-resource-ref")
	p := testProvider("aws")
	pr := testProvisioner()
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	pr.ApplyFn = func(rs *InstanceState, c *ResourceConfig) error {
		val, ok := c.Config["foo"]
		if !ok || val != "2" {
			t.Fatalf("bad value for foo: %v %#v", val, c)
		}

		return nil
	}

	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		Provisioners: map[string]ResourceProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyProvisionerResourceRefStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}

	// Verify apply was invoked
	if !pr.ApplyCalled {
		t.Fatalf("provisioner not invoked")
	}
}

func TestContext2Apply_provisionerSelfRef(t *testing.T) {
	m := testModule(t, "apply-provisioner-self-ref")
	p := testProvider("aws")
	pr := testProvisioner()
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	pr.ApplyFn = func(rs *InstanceState, c *ResourceConfig) error {
		val, ok := c.Config["command"]
		if !ok || val != "bar" {
			t.Fatalf("bad value for command: %v %#v", val, c)
		}

		return nil
	}

	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		Provisioners: map[string]ResourceProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyProvisionerSelfRefStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}

	// Verify apply was invoked
	if !pr.ApplyCalled {
		t.Fatalf("provisioner not invoked")
	}
}

func TestContext2Apply_provisionerMultiSelfRef(t *testing.T) {
	var lock sync.Mutex
	commands := make([]string, 0, 5)

	m := testModule(t, "apply-provisioner-multi-self-ref")
	p := testProvider("aws")
	pr := testProvisioner()
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	pr.ApplyFn = func(rs *InstanceState, c *ResourceConfig) error {
		lock.Lock()
		defer lock.Unlock()

		val, ok := c.Config["command"]
		if !ok {
			t.Fatalf("bad value for command: %v %#v", val, c)
		}

		commands = append(commands, val.(string))
		return nil
	}

	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		Provisioners: map[string]ResourceProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyProvisionerMultiSelfRefStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}

	// Verify apply was invoked
	if !pr.ApplyCalled {
		t.Fatalf("provisioner not invoked")
	}

	// Verify our result
	sort.Strings(commands)
	expectedCommands := []string{"number 0", "number 1", "number 2"}
	if !reflect.DeepEqual(commands, expectedCommands) {
		t.Fatalf("bad: %#v", commands)
	}
}

// Provisioner should NOT run on a diff, only create
func TestContext2Apply_Provisioner_Diff(t *testing.T) {
	m := testModule(t, "apply-provisioner-diff")
	p := testProvider("aws")
	pr := testProvisioner()
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	pr.ApplyFn = func(rs *InstanceState, c *ResourceConfig) error {
		return nil
	}
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		Provisioners: map[string]ResourceProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyProvisionerDiffStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}

	// Verify apply was invoked
	if !pr.ApplyCalled {
		t.Fatalf("provisioner not invoked")
	}
	pr.ApplyCalled = false

	// Change the state to force a diff
	mod := state.RootModule()
	mod.Resources["aws_instance.bar"].Primary.Attributes["foo"] = "baz"

	// Re-create context with state
	ctx = testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		Provisioners: map[string]ResourceProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
		State: state,
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state2, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual = strings.TrimSpace(state2.String())
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}

	// Verify apply was NOT invoked
	if pr.ApplyCalled {
		t.Fatalf("provisioner invoked")
	}
}

func TestContext2Apply_outputDiffVars(t *testing.T) {
	m := testModule(t, "apply-good")
	p := testProvider("aws")
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

	p.ApplyFn = func(info *InstanceInfo, s *InstanceState, d *InstanceDiff) (*InstanceState, error) {
		for k, ad := range d.Attributes {
			if ad.NewComputed {
				return nil, fmt.Errorf("%s: computed", k)
			}
		}

		result := s.MergeDiff(d)
		result.ID = "foo"
		return result, nil
	}
	p.DiffFn = func(*InstanceInfo, *InstanceState, *ResourceConfig) (*InstanceDiff, error) {
		return &InstanceDiff{
			Attributes: map[string]*ResourceAttrDiff{
				"foo": &ResourceAttrDiff{
					NewComputed: true,
					Type:        DiffAttrOutput,
				},
				"bar": &ResourceAttrDiff{
					New: "baz",
				},
			},
		}, nil
	}

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}
	if _, err := ctx.Apply(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestContext2Apply_Provisioner_ConnInfo(t *testing.T) {
	m := testModule(t, "apply-provisioner-conninfo")
	p := testProvider("aws")
	pr := testProvisioner()

	p.ApplyFn = func(info *InstanceInfo, s *InstanceState, d *InstanceDiff) (*InstanceState, error) {
		if s.Ephemeral.ConnInfo == nil {
			t.Fatalf("ConnInfo not initialized")
		}

		result, _ := testApplyFn(info, s, d)
		result.Ephemeral.ConnInfo = map[string]string{
			"type": "ssh",
			"host": "127.0.0.1",
			"port": "22",
		}
		return result, nil
	}
	p.DiffFn = testDiffFn

	pr.ApplyFn = func(rs *InstanceState, c *ResourceConfig) error {
		conn := rs.Ephemeral.ConnInfo
		if conn["type"] != "telnet" {
			t.Fatalf("Bad: %#v", conn)
		}
		if conn["host"] != "127.0.0.1" {
			t.Fatalf("Bad: %#v", conn)
		}
		if conn["port"] != "2222" {
			t.Fatalf("Bad: %#v", conn)
		}
		if conn["user"] != "superuser" {
			t.Fatalf("Bad: %#v", conn)
		}
		if conn["pass"] != "test" {
			t.Fatalf("Bad: %#v", conn)
		}

		return nil
	}

	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		Provisioners: map[string]ResourceProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
		Variables: map[string]string{
			"value": "1",
			"pass":  "test",
		},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyProvisionerStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}

	// Verify apply was invoked
	if !pr.ApplyCalled {
		t.Fatalf("provisioner not invoked")
	}
}

func TestContext2Apply_destroy(t *testing.T) {
	m := testModule(t, "apply-destroy")
	h := new(HookRecordApplyOrder)
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Hooks:  []Hook{h},
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	// First plan and apply a create operation
	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Next, plan and apply a destroy operation
	h.Active = true
	ctx = testContext2(t, &ContextOpts{
		Destroy: true,
		State:   state,
		Module:  m,
		Hooks:   []Hook{h},
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err = ctx.Apply()
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
		t.Fatalf("expected: %#v\n\ngot:%#v", expected2, actual2)
	}
}

func TestContext2Apply_destroyNestedModule(t *testing.T) {
	m := testModule(t, "apply-destroy-nested-module")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	s := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: []string{"root", "child", "subchild"},
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
	}

	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		State: s,
	})

	// First plan and apply a create operation
	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Test that things were destroyed
	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyDestroyNestedModuleStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContext2Apply_destroyDeeplyNestedModule(t *testing.T) {
	m := testModule(t, "apply-destroy-deeply-nested-module")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	s := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: []string{"root", "child", "subchild", "subsubchild"},
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
	}

	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		State: s,
	})

	// First plan and apply a create operation
	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Test that things were destroyed
	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(`
module.child.subchild.subsubchild:
  <no state>
	`)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

// https://github.com/hashicorp/terraform/issues/5440
func TestContext2Apply_destroyModuleWithAttrsReferencingResource(t *testing.T) {
	m := testModule(t, "apply-destroy-module-with-attrs")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	var state *State
	var err error
	{
		ctx := testContext2(t, &ContextOpts{
			Module: m,
			Providers: map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
			Variables: map[string]string{
				"key_name": "foobarkey",
			},
		})

		// First plan and apply a create operation
		if _, err := ctx.Plan(); err != nil {
			t.Fatalf("err: %s", err)
		}

		state, err = ctx.Apply()
		if err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	h := new(HookRecordApplyOrder)
	h.Active = true

	{
		ctx := testContext2(t, &ContextOpts{
			Destroy: true,
			Module:  m,
			State:   state,
			Hooks:   []Hook{h},
			Providers: map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
			Variables: map[string]string{
				"key_name": "foobarkey",
			},
		})

		// First plan and apply a create operation
		plan, err := ctx.Plan()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		var buf bytes.Buffer
		if err := WritePlan(plan, &buf); err != nil {
			t.Fatalf("err: %s", err)
		}

		planFromFile, err := ReadPlan(&buf)
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		ctx = planFromFile.Context(&ContextOpts{
			Providers: map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		})

		state, err = ctx.Apply()
		if err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	//Test that things were destroyed
	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(`
<no state>
module.child:
  <no state>
		`)
	if actual != expected {
		t.Fatalf("expected: \n%s\n\nbad: \n%s", expected, actual)
	}
}

func TestContext2Apply_destroyOutputs(t *testing.T) {
	m := testModule(t, "apply-destroy-outputs")
	h := new(HookRecordApplyOrder)
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Hooks:  []Hook{h},
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	// First plan and apply a create operation
	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()

	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Next, plan and apply a destroy operation
	h.Active = true
	ctx = testContext2(t, &ContextOpts{
		Destroy: true,
		State:   state,
		Module:  m,
		Hooks:   []Hook{h},
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err = ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	mod := state.RootModule()
	if len(mod.Resources) > 0 {
		t.Fatalf("bad: %#v", mod)
	}
}

func TestContext2Apply_destroyOrphan(t *testing.T) {
	m := testModule(t, "apply-error")
	p := testProvider("aws")
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

	p.ApplyFn = func(info *InstanceInfo, s *InstanceState, d *InstanceDiff) (*InstanceState, error) {
		if d.Destroy {
			return nil, nil
		}

		result := s.MergeDiff(d)
		result.ID = "foo"
		return result, nil
	}
	p.DiffFn = func(*InstanceInfo, *InstanceState, *ResourceConfig) (*InstanceDiff, error) {
		return &InstanceDiff{
			Attributes: map[string]*ResourceAttrDiff{
				"num": &ResourceAttrDiff{
					New: "bar",
				},
			},
		}, nil
	}

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	mod := state.RootModule()
	if _, ok := mod.Resources["aws_instance.baz"]; ok {
		t.Fatalf("bad: %#v", mod.Resources)
	}
}

func TestContext2Apply_destroyTaintedProvisioner(t *testing.T) {
	m := testModule(t, "apply-destroy-provisioner")
	p := testProvider("aws")
	pr := testProvisioner()
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	called := false
	pr.ApplyFn = func(rs *InstanceState, c *ResourceConfig) error {
		called = true
		return nil
	}

	s := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.foo": &ResourceState{
						Type: "aws_instance",
						Tainted: []*InstanceState{
							&InstanceState{
								ID: "bar",
								Attributes: map[string]string{
									"id": "bar",
								},
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
		Provisioners: map[string]ResourceProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
		State:   s,
		Destroy: true,
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if called {
		t.Fatal("provisioner should not be called")
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace("<no state>")
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContext2Apply_error(t *testing.T) {
	errored := false

	m := testModule(t, "apply-error")
	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	p.ApplyFn = func(*InstanceInfo, *InstanceState, *InstanceDiff) (*InstanceState, error) {
		if errored {
			state := &InstanceState{
				ID: "bar",
			}
			return state, fmt.Errorf("error")
		}
		errored = true

		return &InstanceState{
			ID: "foo",
			Attributes: map[string]string{
				"num": "2",
			},
		}, nil
	}
	p.DiffFn = func(*InstanceInfo, *InstanceState, *ResourceConfig) (*InstanceDiff, error) {
		return &InstanceDiff{
			Attributes: map[string]*ResourceAttrDiff{
				"num": &ResourceAttrDiff{
					New: "bar",
				},
			},
		}, nil
	}

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err == nil {
		t.Fatal("should have error")
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyErrorStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContext2Apply_errorPartial(t *testing.T) {
	errored := false

	m := testModule(t, "apply-error")
	p := testProvider("aws")
	s := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
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
	}
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		State: s,
	})

	p.ApplyFn = func(info *InstanceInfo, s *InstanceState, d *InstanceDiff) (*InstanceState, error) {
		if errored {
			return s, fmt.Errorf("error")
		}
		errored = true

		return &InstanceState{
			ID: "foo",
			Attributes: map[string]string{
				"num": "2",
			},
		}, nil
	}
	p.DiffFn = func(*InstanceInfo, *InstanceState, *ResourceConfig) (*InstanceDiff, error) {
		return &InstanceDiff{
			Attributes: map[string]*ResourceAttrDiff{
				"num": &ResourceAttrDiff{
					New: "bar",
				},
			},
		}, nil
	}

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err == nil {
		t.Fatal("should have error")
	}

	mod := state.RootModule()
	if len(mod.Resources) != 2 {
		t.Fatalf("bad: %#v", mod.Resources)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyErrorPartialStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContext2Apply_hook(t *testing.T) {
	m := testModule(t, "apply-good")
	h := new(MockHook)
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Hooks:  []Hook{h},
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	if _, err := ctx.Plan(); err != nil {
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
	if !h.PostStateUpdateCalled {
		t.Fatalf("should call post state update")
	}
}

func TestContext2Apply_hookOrphan(t *testing.T) {
	m := testModule(t, "apply-blank")
	h := new(MockHook)
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
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
	}

	ctx := testContext2(t, &ContextOpts{
		Module: m,
		State:  state,
		Hooks:  []Hook{h},
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	if _, err := ctx.Plan(); err != nil {
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
	if !h.PostStateUpdateCalled {
		t.Fatalf("should call post state update")
	}
}

func TestContext2Apply_idAttr(t *testing.T) {
	m := testModule(t, "apply-idattr")
	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	p.ApplyFn = func(info *InstanceInfo, s *InstanceState, d *InstanceDiff) (*InstanceState, error) {
		result := s.MergeDiff(d)
		result.ID = "foo"
		result.Attributes = map[string]string{
			"id": "bar",
		}

		return result, nil
	}
	p.DiffFn = func(*InstanceInfo, *InstanceState, *ResourceConfig) (*InstanceDiff, error) {
		return &InstanceDiff{
			Attributes: map[string]*ResourceAttrDiff{
				"num": &ResourceAttrDiff{
					New: "bar",
				},
			},
		}, nil
	}

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	mod := state.RootModule()
	rs, ok := mod.Resources["aws_instance.foo"]
	if !ok {
		t.Fatal("not in state")
	}
	if rs.Primary.ID != "foo" {
		t.Fatalf("bad: %#v", rs.Primary.ID)
	}
	if rs.Primary.Attributes["id"] != "foo" {
		t.Fatalf("bad: %#v", rs.Primary.Attributes)
	}
}

func TestContext2Apply_output(t *testing.T) {
	m := testModule(t, "apply-output")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	if _, err := ctx.Plan(); err != nil {
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

func TestContext2Apply_outputInvalid(t *testing.T) {
	m := testModule(t, "apply-output-invalid")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	_, err := ctx.Plan()
	if err == nil {
		t.Fatalf("err: %s", err)
	}
	if !strings.Contains(err.Error(), "is not a string") {
		t.Fatalf("err: %s", err)
	}
}

func TestContext2Apply_outputAdd(t *testing.T) {
	m1 := testModule(t, "apply-output-add-before")
	p1 := testProvider("aws")
	p1.ApplyFn = testApplyFn
	p1.DiffFn = testDiffFn
	ctx1 := testContext2(t, &ContextOpts{
		Module: m1,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p1),
		},
	})

	if _, err := ctx1.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state1, err := ctx1.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	m2 := testModule(t, "apply-output-add-after")
	p2 := testProvider("aws")
	p2.ApplyFn = testApplyFn
	p2.DiffFn = testDiffFn
	ctx2 := testContext2(t, &ContextOpts{
		Module: m2,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p2),
		},
		State: state1,
	})

	if _, err := ctx2.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state2, err := ctx2.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state2.String())
	expected := strings.TrimSpace(testTerraformApplyOutputAddStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContext2Apply_outputList(t *testing.T) {
	m := testModule(t, "apply-output-list")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyOutputListStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContext2Apply_outputMulti(t *testing.T) {
	m := testModule(t, "apply-output-multi")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	if _, err := ctx.Plan(); err != nil {
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

func TestContext2Apply_outputMultiIndex(t *testing.T) {
	m := testModule(t, "apply-output-multi-index")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	if _, err := ctx.Plan(); err != nil {
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

func TestContext2Apply_taint(t *testing.T) {
	m := testModule(t, "apply-taint")
	p := testProvider("aws")

	// destroyCount tests against regression of
	// https://github.com/hashicorp/terraform/issues/1056
	var destroyCount = int32(0)
	var once sync.Once
	simulateProviderDelay := func() {
		time.Sleep(10 * time.Millisecond)
	}

	p.ApplyFn = func(info *InstanceInfo, s *InstanceState, d *InstanceDiff) (*InstanceState, error) {
		once.Do(simulateProviderDelay)
		if d.Destroy {
			atomic.AddInt32(&destroyCount, 1)
		}
		return testApplyFn(info, s, d)
	}
	p.DiffFn = testDiffFn
	s := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.bar": &ResourceState{
						Type: "aws_instance",
						Tainted: []*InstanceState{
							&InstanceState{
								ID: "baz",
								Attributes: map[string]string{
									"num":  "2",
									"type": "aws_instance",
								},
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

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyTaintStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}

	if destroyCount != 1 {
		t.Fatalf("Expected 1 destroy, got %d", destroyCount)
	}
}

func TestContext2Apply_taintDep(t *testing.T) {
	m := testModule(t, "apply-taint-dep")
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
						Tainted: []*InstanceState{
							&InstanceState{
								ID: "baz",
								Attributes: map[string]string{
									"num":  "2",
									"type": "aws_instance",
								},
							},
						},
					},
					"aws_instance.bar": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
							Attributes: map[string]string{
								"foo":  "baz",
								"num":  "2",
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

	if p, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	} else {
		t.Logf("plan: %s", p)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyTaintDepStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContext2Apply_taintDepRequiresNew(t *testing.T) {
	m := testModule(t, "apply-taint-dep-requires-new")
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
						Tainted: []*InstanceState{
							&InstanceState{
								ID: "baz",
								Attributes: map[string]string{
									"num":  "2",
									"type": "aws_instance",
								},
							},
						},
					},
					"aws_instance.bar": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
							Attributes: map[string]string{
								"foo":  "baz",
								"num":  "2",
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

	if p, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	} else {
		t.Logf("plan: %s", p)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyTaintDepRequireNewStr)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestContext2Apply_targeted(t *testing.T) {
	m := testModule(t, "apply-targeted")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		Targets: []string{"aws_instance.foo"},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	mod := state.RootModule()
	if len(mod.Resources) != 1 {
		t.Fatalf("expected 1 resource, got: %#v", mod.Resources)
	}

	checkStateString(t, state, `
aws_instance.foo:
  ID = foo
  num = 2
  type = aws_instance
	`)
}

func TestContext2Apply_targetedCount(t *testing.T) {
	m := testModule(t, "apply-targeted-count")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		Targets: []string{"aws_instance.foo"},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	checkStateString(t, state, `
aws_instance.foo.0:
  ID = foo
aws_instance.foo.1:
  ID = foo
aws_instance.foo.2:
  ID = foo
	`)
}

func TestContext2Apply_targetedCountIndex(t *testing.T) {
	m := testModule(t, "apply-targeted-count")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		Targets: []string{"aws_instance.foo[1]"},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	checkStateString(t, state, `
aws_instance.foo.1:
  ID = foo
	`)
}

func TestContext2Apply_targetedDestroy(t *testing.T) {
	m := testModule(t, "apply-targeted")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
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
						"aws_instance.foo": resourceState("aws_instance", "i-bcd345"),
						"aws_instance.bar": resourceState("aws_instance", "i-abc123"),
					},
				},
			},
		},
		Targets: []string{"aws_instance.foo"},
		Destroy: true,
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	mod := state.RootModule()
	if len(mod.Resources) != 1 {
		t.Fatalf("expected 1 resource, got: %#v", mod.Resources)
	}

	checkStateString(t, state, `
aws_instance.bar:
  ID = i-abc123
	`)
}

// https://github.com/hashicorp/terraform/issues/4462
func TestContext2Apply_targetedDestroyModule(t *testing.T) {
	m := testModule(t, "apply-targeted-module")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
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
						"aws_instance.foo": resourceState("aws_instance", "i-bcd345"),
						"aws_instance.bar": resourceState("aws_instance", "i-abc123"),
					},
				},
				&ModuleState{
					Path: []string{"root", "child"},
					Resources: map[string]*ResourceState{
						"aws_instance.foo": resourceState("aws_instance", "i-bcd345"),
						"aws_instance.bar": resourceState("aws_instance", "i-abc123"),
					},
				},
			},
		},
		Targets: []string{"module.child.aws_instance.foo"},
		Destroy: true,
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	checkStateString(t, state, `
aws_instance.bar:
  ID = i-abc123
aws_instance.foo:
  ID = i-bcd345

module.child:
  aws_instance.bar:
    ID = i-abc123
	`)
}

func TestContext2Apply_targetedDestroyCountIndex(t *testing.T) {
	m := testModule(t, "apply-targeted-count")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
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
						"aws_instance.foo.0": resourceState("aws_instance", "i-bcd345"),
						"aws_instance.foo.1": resourceState("aws_instance", "i-bcd345"),
						"aws_instance.foo.2": resourceState("aws_instance", "i-bcd345"),
						"aws_instance.bar.0": resourceState("aws_instance", "i-abc123"),
						"aws_instance.bar.1": resourceState("aws_instance", "i-abc123"),
						"aws_instance.bar.2": resourceState("aws_instance", "i-abc123"),
					},
				},
			},
		},
		Targets: []string{
			"aws_instance.foo[2]",
			"aws_instance.bar[1]",
		},
		Destroy: true,
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	checkStateString(t, state, `
aws_instance.bar.0:
  ID = i-abc123
aws_instance.bar.2:
  ID = i-abc123
aws_instance.foo.0:
  ID = i-bcd345
aws_instance.foo.1:
  ID = i-bcd345
	`)
}

func TestContext2Apply_targetedModule(t *testing.T) {
	m := testModule(t, "apply-targeted-module")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		Targets: []string{"module.child"},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	mod := state.ModuleByPath([]string{"root", "child"})
	if mod == nil {
		t.Fatalf("no child module found in the state!\n\n%#v", state)
	}
	if len(mod.Resources) != 2 {
		t.Fatalf("expected 2 resources, got: %#v", mod.Resources)
	}

	checkStateString(t, state, `
<no state>
module.child:
  aws_instance.bar:
    ID = foo
    num = 2
    type = aws_instance
  aws_instance.foo:
    ID = foo
    num = 2
    type = aws_instance
	`)
}

// GH-1858
func TestContext2Apply_targetedModuleDep(t *testing.T) {
	m := testModule(t, "apply-targeted-module-dep")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		Targets: []string{"aws_instance.foo"},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	checkStateString(t, state, `
aws_instance.foo:
  ID = foo
  foo = foo
  type = aws_instance

  Dependencies:
    module.child

module.child:
  aws_instance.mod:
    ID = foo

  Outputs:

  output = foo
	`)
}

func TestContext2Apply_targetedModuleResource(t *testing.T) {
	m := testModule(t, "apply-targeted-module-resource")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		Targets: []string{"module.child.aws_instance.foo"},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	mod := state.ModuleByPath([]string{"root", "child"})
	if len(mod.Resources) != 1 {
		t.Fatalf("expected 1 resource, got: %#v", mod.Resources)
	}

	checkStateString(t, state, `
<no state>
module.child:
  aws_instance.foo:
    ID = foo
    num = 2
    type = aws_instance
	`)
}

func TestContext2Apply_unknownAttribute(t *testing.T) {
	m := testModule(t, "apply-unknown")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	if _, err := ctx.Plan(); err != nil {
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

func TestContext2Apply_unknownAttributeInterpolate(t *testing.T) {
	m := testModule(t, "apply-unknown-interpolate")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	if _, err := ctx.Plan(); err == nil {
		t.Fatal("should error")
	}
}

func TestContext2Apply_vars(t *testing.T) {
	m := testModule(t, "apply-vars")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		Variables: map[string]string{
			"foo":            "us-west-2",
			"amis.us-east-1": "override",
		},
	})

	w, e := ctx.Validate()
	if len(w) > 0 {
		t.Fatalf("bad: %#v", w)
	}
	if len(e) > 0 {
		t.Fatalf("bad: %s", e)
	}

	if _, err := ctx.Plan(); err != nil {
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

func TestContext2Apply_varsEnv(t *testing.T) {
	// Set the env var
	old := tempEnv(t, "TF_VAR_ami", "baz")
	defer os.Setenv("TF_VAR_ami", old)

	m := testModule(t, "apply-vars-env")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	w, e := ctx.Validate()
	if len(w) > 0 {
		t.Fatalf("bad: %#v", w)
	}
	if len(e) > 0 {
		t.Fatalf("bad: %s", e)
	}

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyVarsEnvStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContext2Apply_createBefore_depends(t *testing.T) {
	m := testModule(t, "apply-depends-create-before")
	h := new(HookRecordApplyOrder)
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.web": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
							Attributes: map[string]string{
								"require_new": "ami-old",
							},
						},
					},
					"aws_instance.lb": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "baz",
							Attributes: map[string]string{
								"instance": "bar",
							},
						},
					},
				},
			},
		},
	}
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Hooks:  []Hook{h},
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		State: state,
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	h.Active = true
	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	mod := state.RootModule()
	if len(mod.Resources) < 2 {
		t.Fatalf("bad: %#v", mod.Resources)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyDependsCreateBeforeStr)
	if actual != expected {
		t.Fatalf("bad: \n%s\n%s", actual, expected)
	}

	// Test that things were managed _in the right order_
	order := h.States
	diffs := h.Diffs
	if order[0].ID != "" || diffs[0].Destroy {
		t.Fatalf("should create new instance first: %#v", order)
	}

	if order[1].ID != "baz" {
		t.Fatalf("update must happen after create: %#v", order)
	}

	if order[2].ID != "bar" || !diffs[2].Destroy {
		t.Fatalf("destroy must happen after update: %#v", order)
	}
}

func TestContext2Apply_singleDestroy(t *testing.T) {
	m := testModule(t, "apply-depends-create-before")
	h := new(HookRecordApplyOrder)
	p := testProvider("aws")

	invokeCount := 0
	p.ApplyFn = func(info *InstanceInfo, s *InstanceState, d *InstanceDiff) (*InstanceState, error) {
		invokeCount++
		switch invokeCount {
		case 1:
			if d.Destroy {
				t.Fatalf("should not destroy")
			}
			if s.ID != "" {
				t.Fatalf("should not have ID")
			}
		case 2:
			if d.Destroy {
				t.Fatalf("should not destroy")
			}
			if s.ID != "baz" {
				t.Fatalf("should have id")
			}
		case 3:
			if !d.Destroy {
				t.Fatalf("should destroy")
			}
			if s.ID == "" {
				t.Fatalf("should have ID")
			}
		default:
			t.Fatalf("bad invoke count %d", invokeCount)
		}
		return testApplyFn(info, s, d)
	}
	p.DiffFn = testDiffFn
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.web": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
							Attributes: map[string]string{
								"require_new": "ami-old",
							},
						},
					},
					"aws_instance.lb": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "baz",
							Attributes: map[string]string{
								"instance": "bar",
							},
						},
					},
				},
			},
		},
	}
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Hooks:  []Hook{h},
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		State: state,
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	h.Active = true
	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if invokeCount != 3 {
		t.Fatalf("bad: %d", invokeCount)
	}
}

// GH-5254
func TestContext2Apply_issue5254(t *testing.T) {
	// Create a provider. We use "template" here just to match the repro
	// we got from the issue itself.
	p := testProvider("template")
	p.ResourcesReturn = append(p.ResourcesReturn, ResourceType{
		Name: "template_file",
	})

	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	// Apply cleanly step 0
	ctx := testContext2(t, &ContextOpts{
		Module: testModule(t, "issue-5254/step-0"),
		Providers: map[string]ResourceProviderFactory{
			"template": testProviderFuncFixed(p),
		},
	})

	plan, err := ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Application success. Now make the modification and store a plan
	ctx = testContext2(t, &ContextOpts{
		Module: testModule(t, "issue-5254/step-1"),
		State:  state,
		Providers: map[string]ResourceProviderFactory{
			"template": testProviderFuncFixed(p),
		},
	})

	plan, err = ctx.Plan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Write / Read plan to simulate running it through a Plan file
	var buf bytes.Buffer
	if err := WritePlan(plan, &buf); err != nil {
		t.Fatalf("err: %s", err)
	}

	planFromFile, err := ReadPlan(&buf)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	ctx = planFromFile.Context(&ContextOpts{
		Providers: map[string]ResourceProviderFactory{
			"template": testProviderFuncFixed(p),
		},
	})

	state, err = ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(`
template_file.child:
  ID = foo
  template = Hi
  type = template_file

  Dependencies:
    template_file.parent
template_file.parent:
  ID = foo
  template = Hi
  type = template_file
		`)
	if actual != expected {
		t.Fatalf("expected state: \n%s\ngot: \n%s", expected, actual)
	}
}

func TestContext2Apply_targetedWithTaintedInState(t *testing.T) {
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	p.ApplyFn = testApplyFn
	ctx := testContext2(t, &ContextOpts{
		Module: testModule(t, "apply-tainted-targets"),
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		Targets: []string{"aws_instance.iambeingadded"},
		State: &State{
			Modules: []*ModuleState{
				&ModuleState{
					Path: rootModulePath,
					Resources: map[string]*ResourceState{
						"aws_instance.ifailedprovisioners": &ResourceState{
							Tainted: []*InstanceState{
								&InstanceState{
									ID: "ifailedprovisioners",
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

	// Write / Read plan to simulate running it through a Plan file
	var buf bytes.Buffer
	if err := WritePlan(plan, &buf); err != nil {
		t.Fatalf("err: %s", err)
	}

	planFromFile, err := ReadPlan(&buf)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	ctx = planFromFile.Context(&ContextOpts{
		Module: testModule(t, "apply-tainted-targets"),
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(`
aws_instance.iambeingadded:
  ID = foo
aws_instance.ifailedprovisioners: (1 tainted)
  ID = <not created>
  Tainted ID 1 = ifailedprovisioners
		`)
	if actual != expected {
		t.Fatalf("expected state: \n%s\ngot: \n%s", expected, actual)
	}
}

// Higher level test exposing the bug this covers in
// TestResource_ignoreChangesRequired
func TestContext2Apply_ignoreChangesCreate(t *testing.T) {
	m := testModule(t, "apply-ignore-changes-create")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	if p, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	} else {
		t.Logf(p.String())
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	mod := state.RootModule()
	if len(mod.Resources) != 1 {
		t.Fatalf("bad: %s", state)
	}

	actual := strings.TrimSpace(state.String())
	// Expect no changes from original state
	expected := strings.TrimSpace(`
aws_instance.foo:
  ID = foo
  required_field = set
  type = aws_instance
`)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}
