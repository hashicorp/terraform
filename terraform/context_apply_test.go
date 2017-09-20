package terraform

import (
	"bytes"
	"fmt"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/config/module"
)

func TestContext2Apply_basic(t *testing.T) {
	m := testModule(t, "apply-good")
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

func TestContext2Apply_escape(t *testing.T) {
	m := testModule(t, "apply-escape")
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

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	checkStateString(t, state, `
aws_instance.bar:
  ID = foo
  foo = "bar"
  type = aws_instance
`)
}

func TestContext2Apply_resourceCountOneList(t *testing.T) {
	m := testModule(t, "apply-resource-count-one-list")
	p := testProvider("null")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"null": testProviderFuncFixed(p),
			},
		),
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(`null_resource.foo:
  ID = foo

Outputs:

test = [foo]`)
	if actual != expected {
		t.Fatalf("expected: \n%s\n\ngot: \n%s\n", expected, actual)
	}
}
func TestContext2Apply_resourceCountZeroList(t *testing.T) {
	m := testModule(t, "apply-resource-count-zero-list")
	p := testProvider("null")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"null": testProviderFuncFixed(p),
			},
		),
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(`<no state>
Outputs:

test = []`)
	if actual != expected {
		t.Fatalf("expected: \n%s\n\ngot: \n%s\n", expected, actual)
	}
}

func TestContext2Apply_resourceDependsOnModule(t *testing.T) {
	m := testModule(t, "apply-resource-depends-on-module")
	p := testProvider("aws")
	p.DiffFn = testDiffFn

	{
		// Wait for the dependency, sleep, and verify the graph never
		// called a child.
		var called int32
		var checked bool
		p.ApplyFn = func(
			info *InstanceInfo,
			is *InstanceState,
			id *InstanceDiff) (*InstanceState, error) {
			if info.HumanId() == "module.child.aws_instance.child" {
				checked = true

				// Sleep to allow parallel execution
				time.Sleep(50 * time.Millisecond)

				// Verify that called is 0 (dep not called)
				if atomic.LoadInt32(&called) != 0 {
					return nil, fmt.Errorf("aws_instance.a should not be called")
				}
			}

			atomic.AddInt32(&called, 1)
			return testApplyFn(info, is, id)
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

		state, err := ctx.Apply()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if !checked {
			t.Fatal("should check")
		}

		checkStateString(t, state, testTerraformApplyResourceDependsOnModuleStr)
	}
}

// Test that without a config, the Dependencies in the state are enough
// to maintain proper ordering.
func TestContext2Apply_resourceDependsOnModuleStateOnly(t *testing.T) {
	m := testModule(t, "apply-resource-depends-on-module-empty")
	p := testProvider("aws")
	p.DiffFn = testDiffFn

	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.a": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
						},
						Dependencies: []string{"module.child"},
					},
				},
			},
			&ModuleState{
				Path: []string{"root", "child"},
				Resources: map[string]*ResourceState{
					"aws_instance.child": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
						},
					},
				},
			},
		},
	}

	{
		// Wait for the dependency, sleep, and verify the graph never
		// called a child.
		var called int32
		var checked bool
		p.ApplyFn = func(
			info *InstanceInfo,
			is *InstanceState,
			id *InstanceDiff) (*InstanceState, error) {
			if info.HumanId() == "aws_instance.a" {
				checked = true

				// Sleep to allow parallel execution
				time.Sleep(50 * time.Millisecond)

				// Verify that called is 0 (dep not called)
				if atomic.LoadInt32(&called) != 0 {
					return nil, fmt.Errorf("module child should not be called")
				}
			}

			atomic.AddInt32(&called, 1)
			return testApplyFn(info, is, id)
		}

		ctx := testContext2(t, &ContextOpts{
			Module: m,
			ProviderResolver: ResourceProviderResolverFixed(
				map[string]ResourceProviderFactory{
					"aws": testProviderFuncFixed(p),
				},
			),
			State: state,
		})

		if _, err := ctx.Plan(); err != nil {
			t.Fatalf("err: %s", err)
		}

		state, err := ctx.Apply()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if !checked {
			t.Fatal("should check")
		}

		checkStateString(t, state, `
<no state>
module.child:
  <no state>
		`)
	}
}

func TestContext2Apply_resourceDependsOnModuleDestroy(t *testing.T) {
	m := testModule(t, "apply-resource-depends-on-module")
	p := testProvider("aws")
	p.DiffFn = testDiffFn

	var globalState *State
	{
		p.ApplyFn = testApplyFn
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

		state, err := ctx.Apply()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		globalState = state
	}

	{
		// Wait for the dependency, sleep, and verify the graph never
		// called a child.
		var called int32
		var checked bool
		p.ApplyFn = func(
			info *InstanceInfo,
			is *InstanceState,
			id *InstanceDiff) (*InstanceState, error) {
			if info.HumanId() == "aws_instance.a" {
				checked = true

				// Sleep to allow parallel execution
				time.Sleep(50 * time.Millisecond)

				// Verify that called is 0 (dep not called)
				if atomic.LoadInt32(&called) != 0 {
					return nil, fmt.Errorf("module child should not be called")
				}
			}

			atomic.AddInt32(&called, 1)
			return testApplyFn(info, is, id)
		}

		ctx := testContext2(t, &ContextOpts{
			Module: m,
			ProviderResolver: ResourceProviderResolverFixed(
				map[string]ResourceProviderFactory{
					"aws": testProviderFuncFixed(p),
				},
			),
			State:   globalState,
			Destroy: true,
		})

		if _, err := ctx.Plan(); err != nil {
			t.Fatalf("err: %s", err)
		}

		state, err := ctx.Apply()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if !checked {
			t.Fatal("should check")
		}

		checkStateString(t, state, `
<no state>
module.child:
  <no state>
		`)
	}
}

func TestContext2Apply_resourceDependsOnModuleGrandchild(t *testing.T) {
	m := testModule(t, "apply-resource-depends-on-module-deep")
	p := testProvider("aws")
	p.DiffFn = testDiffFn

	{
		// Wait for the dependency, sleep, and verify the graph never
		// called a child.
		var called int32
		var checked bool
		p.ApplyFn = func(
			info *InstanceInfo,
			is *InstanceState,
			id *InstanceDiff) (*InstanceState, error) {
			if info.HumanId() == "module.child.grandchild.aws_instance.c" {
				checked = true

				// Sleep to allow parallel execution
				time.Sleep(50 * time.Millisecond)

				// Verify that called is 0 (dep not called)
				if atomic.LoadInt32(&called) != 0 {
					return nil, fmt.Errorf("aws_instance.a should not be called")
				}
			}

			atomic.AddInt32(&called, 1)
			return testApplyFn(info, is, id)
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

		state, err := ctx.Apply()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if !checked {
			t.Fatal("should check")
		}

		checkStateString(t, state, testTerraformApplyResourceDependsOnModuleDeepStr)
	}
}

func TestContext2Apply_resourceDependsOnModuleInModule(t *testing.T) {
	m := testModule(t, "apply-resource-depends-on-module-in-module")
	p := testProvider("aws")
	p.DiffFn = testDiffFn

	{
		// Wait for the dependency, sleep, and verify the graph never
		// called a child.
		var called int32
		var checked bool
		p.ApplyFn = func(
			info *InstanceInfo,
			is *InstanceState,
			id *InstanceDiff) (*InstanceState, error) {
			if info.HumanId() == "module.child.grandchild.aws_instance.c" {
				checked = true

				// Sleep to allow parallel execution
				time.Sleep(50 * time.Millisecond)

				// Verify that called is 0 (dep not called)
				if atomic.LoadInt32(&called) != 0 {
					return nil, fmt.Errorf("nothing else should not be called")
				}
			}

			atomic.AddInt32(&called, 1)
			return testApplyFn(info, is, id)
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

		state, err := ctx.Apply()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if !checked {
			t.Fatal("should check")
		}

		checkStateString(t, state, testTerraformApplyResourceDependsOnModuleInModuleStr)
	}
}

func TestContext2Apply_mapVarBetweenModules(t *testing.T) {
	m := testModule(t, "apply-map-var-through-module")
	p := testProvider("null")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"null": testProviderFuncFixed(p),
			},
		),
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(`<no state>
Outputs:

amis_from_module = {eu-west-1:ami-789012 eu-west-2:ami-989484 us-west-1:ami-123456 us-west-2:ami-456789 }

module.test:
  null_resource.noop:
    ID = foo

  Outputs:

  amis_out = {eu-west-1:ami-789012 eu-west-2:ami-989484 us-west-1:ami-123456 us-west-2:ami-456789 }`)
	if actual != expected {
		t.Fatalf("expected: \n%s\n\ngot: \n%s\n", expected, actual)
	}
}

func TestContext2Apply_refCount(t *testing.T) {
	m := testModule(t, "apply-ref-count")
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
	expected := strings.TrimSpace(testTerraformApplyRefCountStr)
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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

// Two providers that are configured should both be configured prior to apply
func TestContext2Apply_providerAliasConfigure(t *testing.T) {
	m := testModule(t, "apply-provider-alias-configure")

	p2 := testProvider("another")
	p2.ApplyFn = testApplyFn
	p2.DiffFn = testDiffFn

	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"another": testProviderFuncFixed(p2),
			},
		),
	})

	if p, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	} else {
		t.Logf(p.String())
	}

	// Configure to record calls AFTER Plan above
	var configCount int32
	p2.ConfigureFn = func(c *ResourceConfig) error {
		atomic.AddInt32(&configCount, 1)

		foo, ok := c.Get("foo")
		if !ok {
			return fmt.Errorf("foo is not found")
		}

		if foo != "bar" {
			return fmt.Errorf("foo: %#v", foo)
		}

		return nil
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if configCount != 2 {
		t.Fatalf("provider config expected 2 calls, got: %d", configCount)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyProviderAliasConfigStr)
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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

// Higher level test at TestResource_dataSourceListApplyPanic
func TestContext2Apply_computedAttrRefTypeMismatch(t *testing.T) {
	m := testModule(t, "apply-computed-attr-ref-type-mismatch")
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

	_, err := ctx.Apply()
	if err == nil {
		t.Fatalf("Expected err, got none!")
	}

	expected := "Expected ami to be string"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected:\n\n%s\n\nto contain:\n\n%s", err, expected)
	}
}

func TestContext2Apply_emptyModule(t *testing.T) {
	m := testModule(t, "apply-empty-module")
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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

// This tests that when a CBD resource depends on a non-CBD resource,
// we can still properly apply changes that require new for both.
func TestContext2Apply_createBeforeDestroy_dependsNonCBD(t *testing.T) {
	m := testModule(t, "apply-cbd-depends-non-cbd")
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

					"aws_instance.foo": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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

	checkStateString(t, state, `
aws_instance.bar:
  ID = foo
  require_new = yes
  type = aws_instance
  value = foo
aws_instance.foo:
  ID = foo
  require_new = yes
  type = aws_instance
	`)
}

func TestContext2Apply_createBeforeDestroy_hook(t *testing.T) {
	h := new(MockHook)
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

	var actual []string
	var actualLock sync.Mutex
	h.PostApplyFn = func(n *InstanceInfo, s *InstanceState, e error) (HookAction, error) {
		actualLock.Lock()
		defer actualLock.Unlock()
		actual = append(actual, n.Id)
		return HookActionContinue, nil
	}

	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Hooks:  []Hook{h},
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: state,
	})

	if p, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	} else {
		t.Logf(p.String())
	}

	if _, err := ctx.Apply(); err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := []string{"aws_instance.bar", "aws_instance.bar (deposed #0)"}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

// Test that we can perform an apply with CBD in a count with deposed instances.
func TestContext2Apply_createBeforeDestroy_deposedCount(t *testing.T) {
	m := testModule(t, "apply-cbd-count")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.bar.0": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID:      "bar",
							Tainted: true,
						},

						Deposed: []*InstanceState{
							&InstanceState{
								ID: "foo",
							},
						},
					},
					"aws_instance.bar.1": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID:      "bar",
							Tainted: true,
						},

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
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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

	checkStateString(t, state, `
aws_instance.bar.0:
  ID = foo
  foo = bar
  type = aws_instance
aws_instance.bar.1:
  ID = foo
  foo = bar
  type = aws_instance
	`)
}

// Test that when we have a deposed instance but a good primary, we still
// destroy the deposed instance.
func TestContext2Apply_createBeforeDestroy_deposedOnly(t *testing.T) {
	m := testModule(t, "apply-cbd-deposed-only")
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

						Deposed: []*InstanceState{
							&InstanceState{
								ID: "foo",
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

	checkStateString(t, state, `
aws_instance.bar:
  ID = bar
	`)
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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

// Test that the destroy operation uses depends_on as a source of ordering.
func TestContext2Apply_destroyDependsOn(t *testing.T) {
	// It is possible for this to be racy, so we loop a number of times
	// just to check.
	for i := 0; i < 10; i++ {
		testContext2Apply_destroyDependsOn(t)
	}
}

func testContext2Apply_destroyDependsOn(t *testing.T) {
	m := testModule(t, "apply-destroy-depends-on")
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
							ID:         "foo",
							Attributes: map[string]string{},
						},
					},

					"aws_instance.bar": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID:         "bar",
							Attributes: map[string]string{},
						},
					},
				},
			},
		},
	}

	// Record the order we see Apply
	var actual []string
	var actualLock sync.Mutex
	p.ApplyFn = func(
		info *InstanceInfo, _ *InstanceState, _ *InstanceDiff) (*InstanceState, error) {
		actualLock.Lock()
		defer actualLock.Unlock()
		actual = append(actual, info.Id)
		return nil, nil
	}

	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State:       state,
		Destroy:     true,
		Parallelism: 1, // To check ordering
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	if _, err := ctx.Apply(); err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := []string{"aws_instance.foo", "aws_instance.bar"}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

// Test that destroy ordering is correct with dependencies only
// in the state.
func TestContext2Apply_destroyDependsOnStateOnly(t *testing.T) {
	// It is possible for this to be racy, so we loop a number of times
	// just to check.
	for i := 0; i < 10; i++ {
		testContext2Apply_destroyDependsOnStateOnly(t)
	}
}

func testContext2Apply_destroyDependsOnStateOnly(t *testing.T) {
	m := testModule(t, "empty")
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
							ID:         "foo",
							Attributes: map[string]string{},
						},
					},

					"aws_instance.bar": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID:         "bar",
							Attributes: map[string]string{},
						},
						Dependencies: []string{"aws_instance.foo"},
					},
				},
			},
		},
	}

	// Record the order we see Apply
	var actual []string
	var actualLock sync.Mutex
	p.ApplyFn = func(
		info *InstanceInfo, _ *InstanceState, _ *InstanceDiff) (*InstanceState, error) {
		actualLock.Lock()
		defer actualLock.Unlock()
		actual = append(actual, info.Id)
		return nil, nil
	}

	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State:       state,
		Destroy:     true,
		Parallelism: 1, // To check ordering
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	if _, err := ctx.Apply(); err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := []string{"aws_instance.bar", "aws_instance.foo"}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

// Test that destroy ordering is correct with dependencies only
// in the state within a module (GH-11749)
func TestContext2Apply_destroyDependsOnStateOnlyModule(t *testing.T) {
	// It is possible for this to be racy, so we loop a number of times
	// just to check.
	for i := 0; i < 10; i++ {
		testContext2Apply_destroyDependsOnStateOnlyModule(t)
	}
}

func testContext2Apply_destroyDependsOnStateOnlyModule(t *testing.T) {
	m := testModule(t, "empty")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: []string{"root", "child"},
				Resources: map[string]*ResourceState{
					"aws_instance.foo": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID:         "foo",
							Attributes: map[string]string{},
						},
					},

					"aws_instance.bar": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID:         "bar",
							Attributes: map[string]string{},
						},
						Dependencies: []string{"aws_instance.foo"},
					},
				},
			},
		},
	}

	// Record the order we see Apply
	var actual []string
	var actualLock sync.Mutex
	p.ApplyFn = func(
		info *InstanceInfo, _ *InstanceState, _ *InstanceDiff) (*InstanceState, error) {
		actualLock.Lock()
		defer actualLock.Unlock()
		actual = append(actual, info.Id)
		return nil, nil
	}

	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State:       state,
		Destroy:     true,
		Parallelism: 1, // To check ordering
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	if _, err := ctx.Apply(); err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := []string{"aws_instance.bar", "aws_instance.foo"}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestContext2Apply_dataBasic(t *testing.T) {
	m := testModule(t, "apply-data-basic")
	p := testProvider("null")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	p.ReadDataApplyReturn = &InstanceState{ID: "yo"}

	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"null": testProviderFuncFixed(p),
			},
		),
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

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyDataBasicStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContext2Apply_destroyData(t *testing.T) {
	m := testModule(t, "apply-destroy-data-resource")
	p := testProvider("null")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"data.null_data_source.testing": &ResourceState{
						Type: "null_data_source",
						Primary: &InstanceState{
							ID: "-",
							Attributes: map[string]string{
								"inputs.#":    "1",
								"inputs.test": "yes",
							},
						},
					},
				},
			},
		},
	}
	hook := &testHook{}
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"null": testProviderFuncFixed(p),
			},
		),
		State:   state,
		Destroy: true,
		Hooks:   []Hook{hook},
	})

	if p, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	} else {
		t.Logf(p.String())
	}

	newState, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if got := len(newState.Modules); got != 1 {
		t.Fatalf("state has %d modules after destroy; want 1", got)
	}

	if got := len(newState.Modules[0].Resources); got != 0 {
		t.Fatalf("state has %d resources after destroy; want 0", got)
	}

	wantHookCalls := []*testHookCall{
		{"PreDiff", "data.null_data_source.testing"},
		{"PostDiff", "data.null_data_source.testing"},
		{"PostStateUpdate", ""},
	}
	if !reflect.DeepEqual(hook.Calls, wantHookCalls) {
		t.Errorf("wrong hook calls\ngot: %swant: %s", spew.Sdump(hook.Calls), spew.Sdump(wantHookCalls))
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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

func TestContext2Apply_destroyModuleVarProviderConfig(t *testing.T) {
	m := testModule(t, "apply-destroy-mod-var-provider-config")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: []string{"root", "child"},
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
	}
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State:   state,
		Destroy: true,
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	_, err := ctx.Apply()
	if err != nil {
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
				Path: []string{"root", "child"},
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
		Module:           m,
		ProviderResolver: ResourceProviderResolverFixed(providers),
		State:            state,
		Destroy:          true,
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	p.ApplyFn = func(*InstanceInfo, *InstanceState, *InstanceDiff) (*InstanceState, error) {
		if !stopped {
			stopped = true
			go ctx.Stop()

			for {
				if ctx.sh.Stopped() {
					break
				}
				time.Sleep(10 * time.Millisecond)
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
	var applyErr error
	stateCh := make(chan *State)
	go func() {
		state, err := ctx.Apply()
		if err != nil {
			applyErr = err
		}

		stateCh <- state
	}()

	state := <-stateCh
	if applyErr != nil {
		t.Fatalf("err: %s", applyErr)
	}

	mod := state.RootModule()
	if len(mod.Resources) != 1 {
		t.Fatalf("bad: %s", state.String())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyCancelStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}

	if !p.StopCalled {
		t.Fatal("stop should be called")
	}
}

func TestContext2Apply_cancelBlock(t *testing.T) {
	m := testModule(t, "apply-cancel-block")
	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	applyCh := make(chan struct{})
	p.DiffFn = testDiffFn
	p.ApplyFn = func(*InstanceInfo, *InstanceState, *InstanceDiff) (*InstanceState, error) {
		close(applyCh)

		for !ctx.sh.Stopped() {
			// Wait for stop to be called. We call Gosched here so that
			// the other goroutines can always be scheduled to set Stopped.
			runtime.Gosched()
		}

		// Sleep
		time.Sleep(100 * time.Millisecond)

		return &InstanceState{
			ID: "foo",
		}, nil
	}

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Start the Apply in a goroutine
	var applyErr error
	stateCh := make(chan *State)
	go func() {
		state, err := ctx.Apply()
		if err != nil {
			applyErr = err
		}

		stateCh <- state
	}()

	stopDone := make(chan struct{})
	go func() {
		defer close(stopDone)
		<-applyCh
		ctx.Stop()
	}()

	// Make sure that stop blocks
	select {
	case <-stopDone:
		t.Fatal("stop should block")
	case <-time.After(10 * time.Millisecond):
	}

	// Wait for stop
	select {
	case <-stopDone:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("stop should be done")
	}

	// Wait for apply to complete
	state := <-stateCh
	if applyErr != nil {
		t.Fatalf("err: %s", applyErr)
	}

	checkStateString(t, state, `
aws_instance.foo:
  ID = foo
	`)
}

func TestContext2Apply_cancelProvisioner(t *testing.T) {
	m := testModule(t, "apply-cancel-provisioner")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
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
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	prStopped := make(chan struct{})
	pr.ApplyFn = func(rs *InstanceState, c *ResourceConfig) error {
		// Start the stop process
		go ctx.Stop()

		<-prStopped
		return nil
	}
	pr.StopFn = func() error {
		close(prStopped)
		return nil
	}

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Start the Apply in a goroutine
	var applyErr error
	stateCh := make(chan *State)
	go func() {
		state, err := ctx.Apply()
		if err != nil {
			applyErr = err
		}

		stateCh <- state
	}()

	// Wait for completion
	state := <-stateCh
	if applyErr != nil {
		t.Fatalf("err: %s", applyErr)
	}

	checkStateString(t, state, `
aws_instance.foo: (tainted)
  ID = foo
  num = 2
  type = aws_instance
	`)

	if !pr.StopCalled {
		t.Fatal("stop should be called")
	}
}

func TestContext2Apply_compute(t *testing.T) {
	m := testModule(t, "apply-compute")
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

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	ctx.variables = map[string]interface{}{"value": "1"}

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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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

func TestContext2Apply_countDecreaseToOneX(t *testing.T) {
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
						Primary: &InstanceState{
							ID: "bar",
							Attributes: map[string]string{
								"foo":  "foo",
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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

func TestContext2Apply_countVariableRef(t *testing.T) {
	m := testModule(t, "apply-count-variable-ref")
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

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyCountVariableRefStr)
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Variables: map[string]interface{}{
			"images": []map[string]interface{}{
				map[string]interface{}{
					"us-west-2": "overridden",
				},
			},
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

func TestContext2Apply_moduleBasic(t *testing.T) {
	m := testModule(t, "apply-module")
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
		t.Fatalf("bad, expected:\n%s\n\nactual:\n%s", expected, actual)
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
				Outputs: map[string]*OutputState{
					"a_output": &OutputState{
						Type:      "string",
						Sensitive: false,
						Value:     "a",
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

func TestContext2Apply_moduleInheritAlias(t *testing.T) {
	m := testModule(t, "apply-module-provider-inherit-alias")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	p.ConfigureFn = func(c *ResourceConfig) error {
		if _, ok := c.Get("child"); !ok {
			return nil
		}

		if _, ok := c.Get("root"); ok {
			return fmt.Errorf("child should not get root")
		}

		return nil
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

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	checkStateString(t, state, `
<no state>
module.child:
  aws_instance.foo:
    ID = foo
    provider = aws.eu
	`)
}

func TestContext2Apply_moduleOrphanInheritAlias(t *testing.T) {
	m := testModule(t, "apply-module-provider-inherit-alias-orphan")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	called := false
	p.ConfigureFn = func(c *ResourceConfig) error {
		called = true

		if _, ok := c.Get("child"); !ok {
			return nil
		}

		if _, ok := c.Get("root"); ok {
			return fmt.Errorf("child should not get root")
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
						Provider: "aws.eu",
					},
				},
			},
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Module: m,
		State:  state,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !called {
		t.Fatal("must call configure")
	}

	checkStateString(t, state, `
module.child:
  <no state>
  `)
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	if _, err := ctx.Apply(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestContext2Apply_moduleOrphanGrandchildProvider(t *testing.T) {
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

	// Create a state with an orphan module that is nested (grandchild)
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: []string{"root", "parent", "child"},
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws":  testProviderFuncFixed(p),
				"test": testProviderFuncFixed(pTest),
			},
		),
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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

// Tests that variables used as module vars that reference data that
// already exists in the state and requires no diff works properly. This
// fixes an issue faced where module variables were pruned because they were
// accessing "non-existent" resources (they existed, just not in the graph
// cause they weren't in the diff).
func TestContext2Apply_moduleVarRefExisting(t *testing.T) {
	m := testModule(t, "apply-ref-existing")
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
	expected := strings.TrimSpace(testTerraformApplyModuleVarRefExistingStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContext2Apply_moduleVarResourceCount(t *testing.T) {
	m := testModule(t, "apply-module-var-resource-count")
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
		Variables: map[string]interface{}{
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Variables: map[string]interface{}{
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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

// Tests that a module can be targeted and everything is properly created.
// This adds to the plan test to also just verify that apply works.
func TestContext2Apply_moduleTarget(t *testing.T) {
	m := testModule(t, "plan-targeted-cross-module")
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
		Targets: []string{"module.B"},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	checkStateString(t, state, `
<no state>
module.A:
  aws_instance.foo:
    ID = foo
    foo = bar
    type = aws_instance

  Outputs:

  value = foo
module.B:
  aws_instance.bar:
    ID = foo
    foo = foo
    type = aws_instance
	`)
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
				"do":  testProviderFuncFixed(pDO),
			},
		),
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

func TestContext2Apply_multiProviderDestroy(t *testing.T) {
	m := testModule(t, "apply-multi-provider-destroy")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	p2 := testProvider("do")
	p2.ApplyFn = testApplyFn
	p2.DiffFn = testDiffFn

	var state *State

	// First, create the instances
	{
		ctx := testContext2(t, &ContextOpts{
			Module: m,
			ProviderResolver: ResourceProviderResolverFixed(
				map[string]ResourceProviderFactory{
					"aws":   testProviderFuncFixed(p),
					"vault": testProviderFuncFixed(p2),
				},
			),
		})

		if _, err := ctx.Plan(); err != nil {
			t.Fatalf("err: %s", err)
		}

		s, err := ctx.Apply()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		state = s
	}

	// Destroy them
	{
		// Verify that aws_instance.bar is destroyed first
		var checked bool
		var called int32
		var lock sync.Mutex
		applyFn := func(
			info *InstanceInfo,
			is *InstanceState,
			id *InstanceDiff) (*InstanceState, error) {
			lock.Lock()
			defer lock.Unlock()

			if info.HumanId() == "aws_instance.bar" {
				checked = true

				// Sleep to allow parallel execution
				time.Sleep(50 * time.Millisecond)

				// Verify that called is 0 (dep not called)
				if atomic.LoadInt32(&called) != 0 {
					return nil, fmt.Errorf("nothing else should be called")
				}
			}

			atomic.AddInt32(&called, 1)
			return testApplyFn(info, is, id)
		}

		// Set the apply functions
		p.ApplyFn = applyFn
		p2.ApplyFn = applyFn

		ctx := testContext2(t, &ContextOpts{
			Destroy: true,
			State:   state,
			Module:  m,
			ProviderResolver: ResourceProviderResolverFixed(
				map[string]ResourceProviderFactory{
					"aws":   testProviderFuncFixed(p),
					"vault": testProviderFuncFixed(p2),
				},
			),
		})

		if _, err := ctx.Plan(); err != nil {
			t.Fatalf("err: %s", err)
		}

		s, err := ctx.Apply()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if !checked {
			t.Fatal("should be checked")
		}

		state = s
	}

	checkStateString(t, state, `<no state>`)
}

// This is like the multiProviderDestroy test except it tests that
// dependent resources within a child module that inherit provider
// configuration are still destroyed first.
func TestContext2Apply_multiProviderDestroyChild(t *testing.T) {
	m := testModule(t, "apply-multi-provider-destroy-child")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	p2 := testProvider("do")
	p2.ApplyFn = testApplyFn
	p2.DiffFn = testDiffFn

	var state *State

	// First, create the instances
	{
		ctx := testContext2(t, &ContextOpts{
			Module: m,
			ProviderResolver: ResourceProviderResolverFixed(
				map[string]ResourceProviderFactory{
					"aws":   testProviderFuncFixed(p),
					"vault": testProviderFuncFixed(p2),
				},
			),
		})

		if _, err := ctx.Plan(); err != nil {
			t.Fatalf("err: %s", err)
		}

		s, err := ctx.Apply()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		state = s
	}

	// Destroy them
	{
		// Verify that aws_instance.bar is destroyed first
		var checked bool
		var called int32
		var lock sync.Mutex
		applyFn := func(
			info *InstanceInfo,
			is *InstanceState,
			id *InstanceDiff) (*InstanceState, error) {
			lock.Lock()
			defer lock.Unlock()

			if info.HumanId() == "module.child.aws_instance.bar" {
				checked = true

				// Sleep to allow parallel execution
				time.Sleep(50 * time.Millisecond)

				// Verify that called is 0 (dep not called)
				if atomic.LoadInt32(&called) != 0 {
					return nil, fmt.Errorf("nothing else should be called")
				}
			}

			atomic.AddInt32(&called, 1)
			return testApplyFn(info, is, id)
		}

		// Set the apply functions
		p.ApplyFn = applyFn
		p2.ApplyFn = applyFn

		ctx := testContext2(t, &ContextOpts{
			Destroy: true,
			State:   state,
			Module:  m,
			ProviderResolver: ResourceProviderResolverFixed(
				map[string]ResourceProviderFactory{
					"aws":   testProviderFuncFixed(p),
					"vault": testProviderFuncFixed(p2),
				},
			),
		})

		if _, err := ctx.Plan(); err != nil {
			t.Fatalf("err: %s", err)
		}

		s, err := ctx.Apply()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if !checked {
			t.Fatal("should be checked")
		}

		state = s
	}

	checkStateString(t, state, `
<no state>
module.child:
  <no state>
`)
}

func TestContext2Apply_multiVar(t *testing.T) {
	m := testModule(t, "apply-multi-var")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	// First, apply with a count of 3
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

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := state.RootModule().Outputs["output"]
	expected := "bar0,bar1,bar2"
	if actual == nil || actual.Value != expected {
		t.Fatalf("bad: \n%s", actual)
	}

	t.Logf("Initial state: %s", state.String())

	// Apply again, reduce the count to 1
	{
		ctx := testContext2(t, &ContextOpts{
			Module: m,
			State:  state,
			ProviderResolver: ResourceProviderResolverFixed(
				map[string]ResourceProviderFactory{
					"aws": testProviderFuncFixed(p),
				},
			),
			Variables: map[string]interface{}{
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

		t.Logf("End state: %s", state.String())

		actual := state.RootModule().Outputs["output"]
		if actual == nil {
			t.Fatal("missing output")
		}

		expected := "bar0"
		if actual.Value != expected {
			t.Fatalf("bad: \n%s", actual)
		}
	}
}

// This is a holistic test of multi-var (aka "splat variable") handling
// across several different Terraform subsystems. This is here because
// historically there were quirky differences in handling across different
// parts of Terraform and so here we want to assert the expected behavior and
// ensure that it remains consistent in future.
func TestContext2Apply_multiVarComprehensive(t *testing.T) {
	m := testModule(t, "apply-multi-var-comprehensive")
	p := testProvider("test")

	configs := map[string]*ResourceConfig{}

	p.ApplyFn = testApplyFn

	p.DiffFn = func(info *InstanceInfo, s *InstanceState, c *ResourceConfig) (*InstanceDiff, error) {
		configs[info.HumanId()] = c

		// Return a minimal diff to make sure this resource gets included in
		// the apply graph and thus the final state, but otherwise we're just
		// gathering data for assertions.
		return &InstanceDiff{
			Attributes: map[string]*ResourceAttrDiff{
				"id": &ResourceAttrDiff{
					NewComputed: true,
				},
				"name": &ResourceAttrDiff{
					New: info.HumanId(),
				},
			},
		}, nil
	}

	// First, apply with a count of 3
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"test": testProviderFuncFixed(p),
			},
		),
		Variables: map[string]interface{}{
			"count": "3",
		},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("error during plan: %s", err)
	}

	checkConfig := func(name string, want map[string]interface{}) {
		got := configs[name].Config
		if !reflect.DeepEqual(got, want) {
			t.Errorf(
				"wrong config for %s\ngot:  %s\nwant: %s",
				name, spew.Sdump(got), spew.Sdump(want),
			)
		}
	}

	checkConfig("test_thing.multi_count_var.0", map[string]interface{}{
		"source_id":   unknownValue(),
		"source_name": "test_thing.source.0",
	})
	checkConfig("test_thing.multi_count_var.2", map[string]interface{}{
		"source_id":   unknownValue(),
		"source_name": "test_thing.source.2",
	})
	checkConfig("test_thing.multi_count_derived.0", map[string]interface{}{
		"source_id":   unknownValue(),
		"source_name": "test_thing.source.0",
	})
	checkConfig("test_thing.multi_count_derived.2", map[string]interface{}{
		"source_id":   unknownValue(),
		"source_name": "test_thing.source.2",
	})
	checkConfig("test_thing.whole_splat", map[string]interface{}{
		"source_ids": unknownValue(),
		"source_names": []interface{}{
			"test_thing.source.0",
			"test_thing.source.1",
			"test_thing.source.2",
		},
		"source_ids_from_func": unknownValue(),
		"source_names_from_func": []interface{}{
			"test_thing.source.0",
			"test_thing.source.1",
			"test_thing.source.2",
		},

		// This one ends up being a list with a single unknown value at this
		// layer, but is fixed up inside helper/schema. There is a test for
		// this inside the "test" provider, since core tests can't exercise
		// helper/schema functionality.
		"source_ids_wrapped": []interface{}{unknownValue()},

		"source_names_wrapped": []interface{}{
			"test_thing.source.0",
			"test_thing.source.1",
			"test_thing.source.2",
		},
		"first_source_id":   unknownValue(),
		"first_source_name": "test_thing.source.0",
	})
	checkConfig("module.child.test_thing.whole_splat", map[string]interface{}{
		"source_ids": unknownValue(),
		"source_names": []interface{}{
			"test_thing.source.0",
			"test_thing.source.1",
			"test_thing.source.2",
		},

		// This one ends up being a list with a single unknown value at this
		// layer, but is fixed up inside helper/schema. There is a test for
		// this inside the "test" provider, since core tests can't exercise
		// helper/schema functionality.
		"source_ids_wrapped": []interface{}{unknownValue()},

		"source_names_wrapped": []interface{}{
			"test_thing.source.0",
			"test_thing.source.1",
			"test_thing.source.2",
		},
	})

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("error during apply: %s", err)
	}

	{
		want := map[string]interface{}{
			"source_ids": []interface{}{"foo", "foo", "foo"},
			"source_names": []interface{}{
				"test_thing.source.0",
				"test_thing.source.1",
				"test_thing.source.2",
			},
		}
		got := map[string]interface{}{}
		for k, s := range state.RootModule().Outputs {
			got[k] = s.Value
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf(
				"wrong outputs\ngot:  %s\nwant: %s",
				spew.Sdump(got), spew.Sdump(want),
			)
		}
	}
}

// Test that multi-var (splat) access is ordered by count, not by
// value.
func TestContext2Apply_multiVarOrder(t *testing.T) {
	m := testModule(t, "apply-multi-var-order")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	// First, apply with a count of 3
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

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	t.Logf("State: %s", state.String())

	actual := state.RootModule().Outputs["should-be-11"]
	expected := "index-11"
	if actual == nil || actual.Value != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

// Test that multi-var (splat) access is ordered by count, not by
// value, through interpolations.
func TestContext2Apply_multiVarOrderInterp(t *testing.T) {
	m := testModule(t, "apply-multi-var-order-interp")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	// First, apply with a count of 3
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

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	t.Logf("State: %s", state.String())

	actual := state.RootModule().Outputs["should-be-11"]
	expected := "baz-index-11"
	if actual == nil || actual.Value != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

// Based on GH-10440 where a graph edge wasn't properly being created
// between a modified resource and a count instance being destroyed.
func TestContext2Apply_multiVarCountDec(t *testing.T) {
	var s *State

	// First create resources. Nothing sneaky here.
	{
		m := testModule(t, "apply-multi-var-count-dec")
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
			Variables: map[string]interface{}{
				"count": "2",
			},
		})

		if _, err := ctx.Plan(); err != nil {
			t.Fatalf("err: %s", err)
		}

		state, err := ctx.Apply()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		t.Logf("Step 1 state: %s", state)

		s = state
	}

	// Decrease the count by 1 and verify that everything happens in the
	// right order.
	{
		m := testModule(t, "apply-multi-var-count-dec")
		p := testProvider("aws")
		p.ApplyFn = testApplyFn
		p.DiffFn = testDiffFn

		// Verify that aws_instance.bar is modified first and nothing
		// else happens at the same time.
		var checked bool
		var called int32
		var lock sync.Mutex
		p.ApplyFn = func(
			info *InstanceInfo,
			is *InstanceState,
			id *InstanceDiff) (*InstanceState, error) {
			lock.Lock()
			defer lock.Unlock()

			if info.HumanId() == "aws_instance.bar" {
				checked = true

				// Sleep to allow parallel execution
				time.Sleep(50 * time.Millisecond)

				// Verify that called is 0 (dep not called)
				if atomic.LoadInt32(&called) != 1 {
					return nil, fmt.Errorf("nothing else should be called")
				}
			}

			atomic.AddInt32(&called, 1)
			return testApplyFn(info, is, id)
		}

		ctx := testContext2(t, &ContextOpts{
			State:  s,
			Module: m,
			ProviderResolver: ResourceProviderResolverFixed(
				map[string]ResourceProviderFactory{
					"aws": testProviderFuncFixed(p),
				},
			),
			Variables: map[string]interface{}{
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

		if !checked {
			t.Fatal("apply never called")
		}

		t.Logf("Step 2 state: %s", state)

		s = state
	}
}

// Test that we can resolve a multi-var (splat) for the first resource
// created in a non-root module, which happens when the module state doesn't
// exist yet.
// https://github.com/hashicorp/terraform/issues/14438
func TestContext2Apply_multiVarMissingState(t *testing.T) {
	m := testModule(t, "apply-multi-var-missing-state")
	p := testProvider("test")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	// First, apply with a count of 3
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"test": testProviderFuncFixed(p),
			},
		),
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("plan failed: %s", err)
	}

	// Before the relevant bug was fixed, Terraform would panic during apply.
	if _, err := ctx.Apply(); err != nil {
		t.Fatalf("apply failed: %s", err)
	}

	// If we get here with no errors or panics then our test was successful.
}

func TestContext2Apply_nilDiff(t *testing.T) {
	m := testModule(t, "apply-good")
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

func TestContext2Apply_outputDependsOn(t *testing.T) {
	m := testModule(t, "apply-output-depends-on")
	p := testProvider("aws")
	p.DiffFn = testDiffFn

	{
		// Create a custom apply function that sleeps a bit (to allow parallel
		// graph execution) and then returns an error to force a partial state
		// return. We then verify the output is NOT there.
		p.ApplyFn = func(
			info *InstanceInfo,
			is *InstanceState,
			id *InstanceDiff) (*InstanceState, error) {

			// Sleep to allow parallel execution
			time.Sleep(50 * time.Millisecond)

			// Return error to force partial state
			return nil, fmt.Errorf("abcd")
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

		state, err := ctx.Apply()
		if err == nil || !strings.Contains(err.Error(), "abcd") {
			t.Fatalf("err: %s", err)
		}

		checkStateString(t, state, `<no state>`)
	}

	{
		// Create the standard apply function and verify we get the output
		p.ApplyFn = testApplyFn

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

		state, err := ctx.Apply()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		checkStateString(t, state, `
aws_instance.foo:
  ID = foo

Outputs:

value = result
		`)
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
				Outputs: map[string]*OutputState{
					"foo": &OutputState{
						Type:      "string",
						Sensitive: false,
						Value:     "bar",
					},
					"bar": &OutputState{
						Type:      "string",
						Sensitive: false,
						Value:     "baz",
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

func TestContext2Apply_outputOrphanModule(t *testing.T) {
	m := testModule(t, "apply-output-orphan-module")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: []string{"root", "child"},
				Outputs: map[string]*OutputState{
					"foo": &OutputState{
						Type:  "string",
						Value: "bar",
					},
					"bar": &OutputState{
						Type:  "string",
						Value: "baz",
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
	expected := strings.TrimSpace(testTerraformApplyOutputOrphanModuleStr)
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws":  testProviderFuncFixed(p),
				"test": testProviderFuncFixed(pTest),
			},
		),
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

func TestContext2Apply_providerConfigureDisabled(t *testing.T) {
	m := testModule(t, "apply-provider-configure-disabled")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	called := false
	p.ConfigureFn = func(c *ResourceConfig) error {
		called = true

		if _, ok := c.Get("value"); !ok {
			return fmt.Errorf("value is not found")
		}

		return nil
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

	if _, err := ctx.Apply(); err != nil {
		t.Fatalf("err: %s", err)
	}

	if !called {
		t.Fatal("configure never called")
	}
}

func TestContext2Apply_provisionerModule(t *testing.T) {
	m := testModule(t, "apply-provisioner-module")
	p := testProvider("aws")
	pr := testProvisioner()
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
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
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyProvisionerModuleStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}

	// Verify apply was invoked
	if !pr.ApplyCalled {
		t.Fatalf("provisioner not invoked")
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ResourceProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
		Variables: map[string]interface{}{
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ResourceProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
		Variables: map[string]interface{}{
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
		Module:           m,
		ProviderResolver: ResourceProviderResolverFixed(ps),
		State:            state,
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
		Module:           m,
		ProviderResolver: ResourceProviderResolverFixed(ps),
		State:            state,
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

// Verify that a normal provisioner with on_failure "continue" set won't
// taint the resource and continues executing.
func TestContext2Apply_provisionerFailContinue(t *testing.T) {
	m := testModule(t, "apply-provisioner-fail-continue")
	p := testProvider("aws")
	pr := testProvisioner()
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	pr.ApplyFn = func(rs *InstanceState, c *ResourceConfig) error {
		return fmt.Errorf("provisioner error")
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
  foo = bar
  type = aws_instance
  `)

	// Verify apply was invoked
	if !pr.ApplyCalled {
		t.Fatalf("provisioner not invoked")
	}
}

// Verify that a normal provisioner with on_failure "continue" records
// the error with the hook.
func TestContext2Apply_provisionerFailContinueHook(t *testing.T) {
	h := new(MockHook)
	m := testModule(t, "apply-provisioner-fail-continue")
	p := testProvider("aws")
	pr := testProvisioner()
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	pr.ApplyFn = func(rs *InstanceState, c *ResourceConfig) error {
		return fmt.Errorf("provisioner error")
	}

	ctx := testContext2(t, &ContextOpts{
		Module: m,
		Hooks:  []Hook{h},
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ResourceProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	if _, err := ctx.Apply(); err != nil {
		t.Fatalf("err: %s", err)
	}

	if !h.PostProvisionCalled {
		t.Fatal("PostProvision not called")
	}
	if h.PostProvisionErrorArg == nil {
		t.Fatal("should have error")
	}
}

func TestContext2Apply_provisionerDestroy(t *testing.T) {
	m := testModule(t, "apply-provisioner-destroy")
	p := testProvider("aws")
	pr := testProvisioner()
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	pr.ApplyFn = func(rs *InstanceState, c *ResourceConfig) error {
		val, ok := c.Config["foo"]
		if !ok || val != "destroy" {
			t.Fatalf("bad value for foo: %v %#v", val, c)
		}

		return nil
	}

	state := &State{
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
		Module:  m,
		State:   state,
		Destroy: true,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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

	checkStateString(t, state, `<no state>`)

	// Verify apply was invoked
	if !pr.ApplyCalled {
		t.Fatalf("provisioner not invoked")
	}
}

// Verify that on destroy provisioner failure, nothing happens to the instance
func TestContext2Apply_provisionerDestroyFail(t *testing.T) {
	m := testModule(t, "apply-provisioner-destroy")
	p := testProvider("aws")
	pr := testProvisioner()
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	pr.ApplyFn = func(rs *InstanceState, c *ResourceConfig) error {
		return fmt.Errorf("provisioner error")
	}

	state := &State{
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
		Module:  m,
		State:   state,
		Destroy: true,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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

	checkStateString(t, state, `
aws_instance.foo:
  ID = bar
	`)

	// Verify apply was invoked
	if !pr.ApplyCalled {
		t.Fatalf("provisioner not invoked")
	}
}

// Verify that on destroy provisioner failure with "continue" that
// we continue to the next provisioner.
func TestContext2Apply_provisionerDestroyFailContinue(t *testing.T) {
	m := testModule(t, "apply-provisioner-destroy-continue")
	p := testProvider("aws")
	pr := testProvisioner()
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	var l sync.Mutex
	var calls []string
	pr.ApplyFn = func(rs *InstanceState, c *ResourceConfig) error {
		val, ok := c.Config["foo"]
		if !ok {
			t.Fatalf("bad value for foo: %v %#v", val, c)
		}

		l.Lock()
		defer l.Unlock()
		calls = append(calls, val.(string))
		return fmt.Errorf("provisioner error")
	}

	state := &State{
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
		Module:  m,
		State:   state,
		Destroy: true,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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

	checkStateString(t, state, `<no state>`)

	// Verify apply was invoked
	if !pr.ApplyCalled {
		t.Fatalf("provisioner not invoked")
	}

	expected := []string{"one", "two"}
	if !reflect.DeepEqual(calls, expected) {
		t.Fatalf("bad: %#v", calls)
	}
}

// Verify that on destroy provisioner failure with "continue" that
// we continue to the next provisioner. But if the next provisioner defines
// to fail, then we fail after running it.
func TestContext2Apply_provisionerDestroyFailContinueFail(t *testing.T) {
	m := testModule(t, "apply-provisioner-destroy-fail")
	p := testProvider("aws")
	pr := testProvisioner()
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	var l sync.Mutex
	var calls []string
	pr.ApplyFn = func(rs *InstanceState, c *ResourceConfig) error {
		val, ok := c.Config["foo"]
		if !ok {
			t.Fatalf("bad value for foo: %v %#v", val, c)
		}

		l.Lock()
		defer l.Unlock()
		calls = append(calls, val.(string))
		return fmt.Errorf("provisioner error")
	}

	state := &State{
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
		Module:  m,
		State:   state,
		Destroy: true,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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

	checkStateString(t, state, `
aws_instance.foo:
  ID = bar
  `)

	// Verify apply was invoked
	if !pr.ApplyCalled {
		t.Fatalf("provisioner not invoked")
	}

	expected := []string{"one", "two"}
	if !reflect.DeepEqual(calls, expected) {
		t.Fatalf("bad: %#v", calls)
	}
}

// Verify destroy provisioners are not run for tainted instances.
func TestContext2Apply_provisionerDestroyTainted(t *testing.T) {
	m := testModule(t, "apply-provisioner-destroy")
	p := testProvider("aws")
	pr := testProvisioner()
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	destroyCalled := false
	pr.ApplyFn = func(rs *InstanceState, c *ResourceConfig) error {
		expected := "create"
		if rs.ID == "bar" {
			destroyCalled = true
			return nil
		}

		val, ok := c.Config["foo"]
		if !ok || val != expected {
			t.Fatalf("bad value for foo: %v %#v", val, c)
		}

		return nil
	}

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

	ctx := testContext2(t, &ContextOpts{
		Module: m,
		State:  state,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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

	checkStateString(t, state, `
aws_instance.foo:
  ID = foo
  foo = bar
  type = aws_instance
	`)

	// Verify apply was invoked
	if !pr.ApplyCalled {
		t.Fatalf("provisioner not invoked")
	}

	if destroyCalled {
		t.Fatal("destroy should not be called")
	}
}

func TestContext2Apply_provisionerDestroyModule(t *testing.T) {
	m := testModule(t, "apply-provisioner-destroy-module")
	p := testProvider("aws")
	pr := testProvisioner()
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	pr.ApplyFn = func(rs *InstanceState, c *ResourceConfig) error {
		val, ok := c.Config["foo"]
		if !ok || val != "value" {
			t.Fatalf("bad value for foo: %v %#v", val, c)
		}

		return nil
	}

	state := &State{
		Modules: []*ModuleState{
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
		Module:  m,
		State:   state,
		Destroy: true,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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

	checkStateString(t, state, `
module.child:
  <no state>`)

	// Verify apply was invoked
	if !pr.ApplyCalled {
		t.Fatalf("provisioner not invoked")
	}
}

func TestContext2Apply_provisionerDestroyRef(t *testing.T) {
	m := testModule(t, "apply-provisioner-destroy-ref")
	p := testProvider("aws")
	pr := testProvisioner()
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	pr.ApplyFn = func(rs *InstanceState, c *ResourceConfig) error {
		val, ok := c.Config["foo"]
		if !ok || val != "hello" {
			return fmt.Errorf("bad value for foo: %v %#v", val, c)
		}

		return nil
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
								"key": "hello",
							},
						},
					},

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
		Module:  m,
		State:   state,
		Destroy: true,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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

	checkStateString(t, state, `<no state>`)

	// Verify apply was invoked
	if !pr.ApplyCalled {
		t.Fatalf("provisioner not invoked")
	}
}

// Test that a destroy provisioner referencing an invalid key errors.
func TestContext2Apply_provisionerDestroyRefInvalid(t *testing.T) {
	m := testModule(t, "apply-provisioner-destroy-ref")
	p := testProvider("aws")
	pr := testProvisioner()
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	pr.ApplyFn = func(rs *InstanceState, c *ResourceConfig) error {
		return nil
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
						},
					},

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
		Module:  m,
		State:   state,
		Destroy: true,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ResourceProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	if _, err := ctx.Apply(); err == nil {
		t.Fatal("expected error")
	}
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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

func TestContext2Apply_provisionerMultiSelfRefSingle(t *testing.T) {
	var lock sync.Mutex
	order := make([]string, 0, 5)

	m := testModule(t, "apply-provisioner-multi-self-ref-single")
	p := testProvider("aws")
	pr := testProvisioner()
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	pr.ApplyFn = func(rs *InstanceState, c *ResourceConfig) error {
		lock.Lock()
		defer lock.Unlock()

		val, ok := c.Config["order"]
		if !ok {
			t.Fatalf("bad value for order: %v %#v", val, c)
		}

		order = append(order, val.(string))
		return nil
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
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyProvisionerMultiSelfRefSingleStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}

	// Verify apply was invoked
	if !pr.ApplyCalled {
		t.Fatalf("provisioner not invoked")
	}

	// Verify our result
	sort.Strings(order)
	expectedOrder := []string{"0", "1", "2"}
	if !reflect.DeepEqual(order, expectedOrder) {
		t.Fatalf("bad: %#v", order)
	}
}

func TestContext2Apply_provisionerMultiSelfRefCount(t *testing.T) {
	var lock sync.Mutex
	commands := make([]string, 0, 5)

	m := testModule(t, "apply-provisioner-multi-self-ref-count")
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ResourceProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	if _, err := ctx.Apply(); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Verify apply was invoked
	if !pr.ApplyCalled {
		t.Fatalf("provisioner not invoked")
	}

	// Verify our result
	sort.Strings(commands)
	expectedCommands := []string{"3", "3", "3"}
	if !reflect.DeepEqual(commands, expectedCommands) {
		t.Fatalf("bad: %#v", commands)
	}
}

func TestContext2Apply_provisionerExplicitSelfRef(t *testing.T) {
	m := testModule(t, "apply-provisioner-explicit-self-ref")
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

	var state *State
	{
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
		})

		_, err := ctx.Plan()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		state, err = ctx.Apply()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		// Verify apply was invoked
		if !pr.ApplyCalled {
			t.Fatalf("provisioner not invoked")
		}
	}

	{
		ctx := testContext2(t, &ContextOpts{
			Module:  m,
			Destroy: true,
			State:   state,
			ProviderResolver: ResourceProviderResolverFixed(
				map[string]ResourceProviderFactory{
					"aws": testProviderFuncFixed(p),
				},
			),
			Provisioners: map[string]ResourceProvisionerFactory{
				"shell": testProvisionerFuncFixed(pr),
			},
		})

		_, err := ctx.Plan()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		state, err = ctx.Apply()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		checkStateString(t, state, `<no state>`)
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ResourceProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
		Variables: map[string]interface{}{
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

func TestContext2Apply_destroyX(t *testing.T) {
	m := testModule(t, "apply-destroy")
	h := new(HookRecordApplyOrder)
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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

func TestContext2Apply_destroyOrder(t *testing.T) {
	m := testModule(t, "apply-destroy")
	h := new(HookRecordApplyOrder)
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
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

	// First plan and apply a create operation
	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	t.Logf("State 1: %s", state)

	// Next, plan and apply config-less to force a destroy with "apply"
	h.Active = true
	ctx = testContext2(t, &ContextOpts{
		State:  state,
		Module: module.NewEmptyTree(),
		Hooks:  []Hook{h},
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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

// https://github.com/hashicorp/terraform/issues/2767
func TestContext2Apply_destroyModulePrefix(t *testing.T) {
	m := testModule(t, "apply-destroy-module-resource-prefix")
	h := new(MockHook)
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
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

	// First plan and apply a create operation
	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Verify that we got the apply info correct
	if v := h.PreApplyInfo.HumanId(); v != "module.child.aws_instance.foo" {
		t.Fatalf("bad: %s", v)
	}

	// Next, plan and apply a destroy operation and reset the hook
	h = new(MockHook)
	ctx = testContext2(t, &ContextOpts{
		Destroy: true,
		State:   state,
		Module:  m,
		Hooks:   []Hook{h},
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err = ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Test that things were destroyed
	if v := h.PreApplyInfo.HumanId(); v != "module.child.aws_instance.foo" {
		t.Fatalf("bad: %s", v)
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
			ProviderResolver: ResourceProviderResolverFixed(
				map[string]ResourceProviderFactory{
					"aws": testProviderFuncFixed(p),
				},
			),
		})

		// First plan and apply a create operation
		if p, err := ctx.Plan(); err != nil {
			t.Fatalf("plan err: %s", err)
		} else {
			t.Logf("Step 1 plan: %s", p)
		}

		state, err = ctx.Apply()
		if err != nil {
			t.Fatalf("apply err: %s", err)
		}

		t.Logf("Step 1 state: %s", state)
	}

	h := new(HookRecordApplyOrder)
	h.Active = true

	{
		ctx := testContext2(t, &ContextOpts{
			Destroy: true,
			Module:  m,
			State:   state,
			Hooks:   []Hook{h},
			ProviderResolver: ResourceProviderResolverFixed(
				map[string]ResourceProviderFactory{
					"aws": testProviderFuncFixed(p),
				},
			),
			Variables: map[string]interface{}{
				"key_name": "foobarkey",
			},
		})

		// First plan and apply a create operation
		plan, err := ctx.Plan()
		if err != nil {
			t.Fatalf("destroy plan err: %s", err)
		}

		t.Logf("Step 2 plan: %s", plan)

		var buf bytes.Buffer
		if err := WritePlan(plan, &buf); err != nil {
			t.Fatalf("plan write err: %s", err)
		}

		planFromFile, err := ReadPlan(&buf)
		if err != nil {
			t.Fatalf("plan read err: %s", err)
		}

		ctx, err = planFromFile.Context(&ContextOpts{
			ProviderResolver: ResourceProviderResolverFixed(
				map[string]ResourceProviderFactory{
					"aws": testProviderFuncFixed(p),
				},
			),
		})
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		state, err = ctx.Apply()
		if err != nil {
			t.Fatalf("destroy apply err: %s", err)
		}

		t.Logf("Step 2 state: %s", state)
	}

	//Test that things were destroyed
	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(`
<no state>
module.child:
  <no state>
		`)
	if actual != expected {
		t.Fatalf("expected:\n\n%s\n\nactual:\n\n%s", expected, actual)
	}
}

func TestContext2Apply_destroyWithModuleVariableAndCount(t *testing.T) {
	m := testModule(t, "apply-destroy-mod-var-and-count")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	var state *State
	var err error
	{
		ctx := testContext2(t, &ContextOpts{
			Module: m,
			ProviderResolver: ResourceProviderResolverFixed(
				map[string]ResourceProviderFactory{
					"aws": testProviderFuncFixed(p),
				},
			),
		})

		// First plan and apply a create operation
		if _, err := ctx.Plan(); err != nil {
			t.Fatalf("plan err: %s", err)
		}

		state, err = ctx.Apply()
		if err != nil {
			t.Fatalf("apply err: %s", err)
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
			ProviderResolver: ResourceProviderResolverFixed(
				map[string]ResourceProviderFactory{
					"aws": testProviderFuncFixed(p),
				},
			),
		})

		// First plan and apply a create operation
		plan, err := ctx.Plan()
		if err != nil {
			t.Fatalf("destroy plan err: %s", err)
		}

		var buf bytes.Buffer
		if err := WritePlan(plan, &buf); err != nil {
			t.Fatalf("plan write err: %s", err)
		}

		planFromFile, err := ReadPlan(&buf)
		if err != nil {
			t.Fatalf("plan read err: %s", err)
		}

		ctx, err = planFromFile.Context(&ContextOpts{
			ProviderResolver: ResourceProviderResolverFixed(
				map[string]ResourceProviderFactory{
					"aws": testProviderFuncFixed(p),
				},
			),
		})
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		state, err = ctx.Apply()
		if err != nil {
			t.Fatalf("destroy apply err: %s", err)
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

func TestContext2Apply_destroyTargetWithModuleVariableAndCount(t *testing.T) {
	m := testModule(t, "apply-destroy-mod-var-and-count")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	var state *State
	var err error
	{
		ctx := testContext2(t, &ContextOpts{
			Module: m,
			ProviderResolver: ResourceProviderResolverFixed(
				map[string]ResourceProviderFactory{
					"aws": testProviderFuncFixed(p),
				},
			),
		})

		// First plan and apply a create operation
		if _, err := ctx.Plan(); err != nil {
			t.Fatalf("plan err: %s", err)
		}

		state, err = ctx.Apply()
		if err != nil {
			t.Fatalf("apply err: %s", err)
		}
	}

	{
		ctx := testContext2(t, &ContextOpts{
			Destroy: true,
			Module:  m,
			State:   state,
			ProviderResolver: ResourceProviderResolverFixed(
				map[string]ResourceProviderFactory{
					"aws": testProviderFuncFixed(p),
				},
			),
			Targets: []string{"module.child"},
		})

		_, err := ctx.Plan()
		if err != nil {
			t.Fatalf("plan err: %s", err)
		}

		// Destroy, targeting the module explicitly
		state, err = ctx.Apply()
		if err != nil {
			t.Fatalf("destroy apply err: %s", err)
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

func TestContext2Apply_destroyWithModuleVariableAndCountNested(t *testing.T) {
	m := testModule(t, "apply-destroy-mod-var-and-count-nested")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	var state *State
	var err error
	{
		ctx := testContext2(t, &ContextOpts{
			Module: m,
			ProviderResolver: ResourceProviderResolverFixed(
				map[string]ResourceProviderFactory{
					"aws": testProviderFuncFixed(p),
				},
			),
		})

		// First plan and apply a create operation
		if _, err := ctx.Plan(); err != nil {
			t.Fatalf("plan err: %s", err)
		}

		state, err = ctx.Apply()
		if err != nil {
			t.Fatalf("apply err: %s", err)
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
			ProviderResolver: ResourceProviderResolverFixed(
				map[string]ResourceProviderFactory{
					"aws": testProviderFuncFixed(p),
				},
			),
		})

		// First plan and apply a create operation
		plan, err := ctx.Plan()
		if err != nil {
			t.Fatalf("destroy plan err: %s", err)
		}

		var buf bytes.Buffer
		if err := WritePlan(plan, &buf); err != nil {
			t.Fatalf("plan write err: %s", err)
		}

		planFromFile, err := ReadPlan(&buf)
		if err != nil {
			t.Fatalf("plan read err: %s", err)
		}

		ctx, err = planFromFile.Context(&ContextOpts{
			ProviderResolver: ResourceProviderResolverFixed(
				map[string]ResourceProviderFactory{
					"aws": testProviderFuncFixed(p),
				},
			),
		})
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		state, err = ctx.Apply()
		if err != nil {
			t.Fatalf("destroy apply err: %s", err)
		}
	}

	//Test that things were destroyed
	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(`
<no state>
module.child:
  <no state>
module.child.child2:
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
						Primary: &InstanceState{
							ID: "bar",
							Attributes: map[string]string{
								"id": "bar",
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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

func TestContext2Apply_outputBasic(t *testing.T) {
	m := testModule(t, "apply-output")
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	_, err := ctx.Plan()
	if err == nil {
		t.Fatalf("err: %s", err)
	}
	if !strings.Contains(err.Error(), "is not a valid type") {
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p1),
			},
		),
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p2),
			},
		),
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
		t.Fatalf("expected: \n%s\n\nbad: \n%s", expected, actual)
	}
}

func TestContext2Apply_outputMulti(t *testing.T) {
	m := testModule(t, "apply-output-multi")
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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

func TestContext2Apply_taintX(t *testing.T) {
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
						Primary: &InstanceState{
							ID: "baz",
							Attributes: map[string]string{
								"num":  "2",
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
						Primary: &InstanceState{
							ID: "baz",
							Attributes: map[string]string{
								"num":  "2",
								"type": "aws_instance",
							},
							Tainted: true,
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
						Primary: &InstanceState{
							ID: "baz",
							Attributes: map[string]string{
								"num":  "2",
								"type": "aws_instance",
							},
							Tainted: true,
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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

func TestContext2Apply_targetedDestroyCountDeps(t *testing.T) {
	m := testModule(t, "apply-destroy-targeted-count")
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

	checkStateString(t, state, `<no state>`)
}

// https://github.com/hashicorp/terraform/issues/4462
func TestContext2Apply_targetedDestroyModule(t *testing.T) {
	m := testModule(t, "apply-targeted-module")
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Targets: []string{"aws_instance.foo"},
	})

	if p, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	} else {
		t.Logf("Diff: %s", p)
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

// GH-10911 untargeted outputs should not be in the graph, and therefore
// not execute.
func TestContext2Apply_targetedModuleUnrelatedOutputs(t *testing.T) {
	m := testModule(t, "apply-targeted-module-unrelated-outputs")
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
		Targets: []string{"module.child2"},
		State: &State{
			Modules: []*ModuleState{
				{
					Path:      []string{"root"},
					Outputs:   map[string]*OutputState{},
					Resources: map[string]*ResourceState{},
				},
				{
					Path: []string{"root", "child1"},
					Outputs: map[string]*OutputState{
						"instance_id": {
							Type:  "string",
							Value: "foo-bar-baz",
						},
					},
					Resources: map[string]*ResourceState{},
				},
				{
					Path:      []string{"root", "child2"},
					Outputs:   map[string]*OutputState{},
					Resources: map[string]*ResourceState{},
				},
			},
		},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// module.child1's instance_id output should be retained from state
	// module.child2's instance_id is updated because its dependency is updated
	// child2_id is updated because if its transitive dependency via module.child2
	checkStateString(t, state, `
<no state>
Outputs:

child2_id = foo

module.child1:
  <no state>
  Outputs:

  instance_id = foo-bar-baz
module.child2:
  aws_instance.foo:
    ID = foo

  Outputs:

  instance_id = foo
`)
}

func TestContext2Apply_targetedModuleResource(t *testing.T) {
	m := testModule(t, "apply-targeted-module-resource")
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
	if mod == nil || len(mod.Resources) != 1 {
		t.Fatalf("expected 1 resource, got: %#v", mod)
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Variables: map[string]interface{}{
			"foo":       "us-west-2",
			"test_list": []interface{}{"Hello", "World"},
			"test_map": map[string]interface{}{
				"Hello": "World",
				"Foo":   "Bar",
				"Baz":   "Foo",
			},
			"amis": []map[string]interface{}{
				map[string]interface{}{
					"us-east-1": "override",
				},
			},
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
		t.Fatalf("expected: %s\n got:\n%s", expected, actual)
	}
}

func TestContext2Apply_varsEnv(t *testing.T) {
	// Set the env var
	defer tempEnv(t, "TF_VAR_ami", "baz")()
	defer tempEnv(t, "TF_VAR_list", `["Hello", "World"]`)()
	defer tempEnv(t, "TF_VAR_map", `{"Hello" = "World", "Foo" = "Bar", "Baz" = "Foo"}`)()

	m := testModule(t, "apply-vars-env")
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: state,
	})

	if p, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	} else {
		t.Logf("plan: %s", p)
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
		t.Fatalf("bad: \n%s\n\n%s", actual, expected)
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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

// GH-7824
func TestContext2Apply_issue7824(t *testing.T) {
	p := testProvider("template")
	p.ResourcesReturn = append(p.ResourcesReturn, ResourceType{
		Name: "template_file",
	})

	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	// Apply cleanly step 0
	ctx := testContext2(t, &ContextOpts{
		Module: testModule(t, "issue-7824"),
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"template": testProviderFuncFixed(p),
			},
		),
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

	ctx, err = planFromFile.Context(&ContextOpts{
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"template": testProviderFuncFixed(p),
			},
		),
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	_, err = ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"template": testProviderFuncFixed(p),
			},
		),
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"template": testProviderFuncFixed(p),
			},
		),
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

	ctx, err = planFromFile.Context(&ContextOpts{
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"template": testProviderFuncFixed(p),
			},
		),
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

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
    template_file.parent.*
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Targets: []string{"aws_instance.iambeingadded"},
		State: &State{
			Modules: []*ModuleState{
				&ModuleState{
					Path: rootModulePath,
					Resources: map[string]*ResourceState{
						"aws_instance.ifailedprovisioners": &ResourceState{
							Type: "aws_instance",
							Primary: &InstanceState{
								ID:      "ifailedprovisioners",
								Tainted: true,
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

	ctx, err = planFromFile.Context(&ContextOpts{
		Module: testModule(t, "apply-tainted-targets"),
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(`
aws_instance.iambeingadded:
  ID = foo
aws_instance.ifailedprovisioners: (tainted)
  ID = ifailedprovisioners
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
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
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
		t.Fatalf("expected:\n%s\ngot:\n%s", expected, actual)
	}
}

func TestContext2Apply_ignoreChangesWithDep(t *testing.T) {
	m := testModule(t, "apply-ignore-changes-dep")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = func(i *InstanceInfo, s *InstanceState, c *ResourceConfig) (*InstanceDiff, error) {
		switch i.Type {
		case "aws_instance":
			newAmi, _ := c.Get("ami")
			return &InstanceDiff{
				Attributes: map[string]*ResourceAttrDiff{
					"ami": &ResourceAttrDiff{
						Old:         s.Attributes["ami"],
						New:         newAmi.(string),
						RequiresNew: true,
					},
				},
			}, nil
		case "aws_eip":
			return testDiffFn(i, s, c)
		default:
			t.Fatalf("Unexpected type: %s", i.Type)
			return nil, nil
		}
	}
	s := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.foo.0": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "i-abc123",
							Attributes: map[string]string{
								"ami": "ami-abcd1234",
								"id":  "i-abc123",
							},
						},
					},
					"aws_instance.foo.1": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "i-bcd234",
							Attributes: map[string]string{
								"ami": "ami-abcd1234",
								"id":  "i-bcd234",
							},
						},
					},
					"aws_eip.foo.0": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "eip-abc123",
							Attributes: map[string]string{
								"id":       "eip-abc123",
								"instance": "i-abc123",
							},
						},
					},
					"aws_eip.foo.1": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "eip-bcd234",
							Attributes: map[string]string{
								"id":       "eip-bcd234",
								"instance": "i-bcd234",
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

	if p, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	} else {
		t.Logf(p.String())
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(s.String())
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContext2Apply_ignoreChangesWildcard(t *testing.T) {
	m := testModule(t, "apply-ignore-changes-wildcard")
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
		t.Fatalf("expected:\n%s\ngot:\n%s", expected, actual)
	}
}

// https://github.com/hashicorp/terraform/issues/7378
func TestContext2Apply_destroyNestedModuleWithAttrsReferencingResource(t *testing.T) {
	m := testModule(t, "apply-destroy-nested-module-with-attrs")
	p := testProvider("null")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	var state *State
	var err error
	{
		ctx := testContext2(t, &ContextOpts{
			Module: m,
			ProviderResolver: ResourceProviderResolverFixed(
				map[string]ResourceProviderFactory{
					"null": testProviderFuncFixed(p),
				},
			),
		})

		// First plan and apply a create operation
		if _, err := ctx.Plan(); err != nil {
			t.Fatalf("plan err: %s", err)
		}

		state, err = ctx.Apply()
		if err != nil {
			t.Fatalf("apply err: %s", err)
		}
	}

	{
		ctx := testContext2(t, &ContextOpts{
			Destroy: true,
			Module:  m,
			State:   state,
			ProviderResolver: ResourceProviderResolverFixed(
				map[string]ResourceProviderFactory{
					"null": testProviderFuncFixed(p),
				},
			),
		})

		plan, err := ctx.Plan()
		if err != nil {
			t.Fatalf("destroy plan err: %s", err)
		}

		var buf bytes.Buffer
		if err := WritePlan(plan, &buf); err != nil {
			t.Fatalf("plan write err: %s", err)
		}

		planFromFile, err := ReadPlan(&buf)
		if err != nil {
			t.Fatalf("plan read err: %s", err)
		}

		ctx, err = planFromFile.Context(&ContextOpts{
			ProviderResolver: ResourceProviderResolverFixed(
				map[string]ResourceProviderFactory{
					"null": testProviderFuncFixed(p),
				},
			),
		})
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		state, err = ctx.Apply()
		if err != nil {
			t.Fatalf("destroy apply err: %s", err)
		}
	}

	//Test that things were destroyed
	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(`
<no state>
module.middle:
  <no state>
module.middle.bottom:
  <no state>
		`)
	if actual != expected {
		t.Fatalf("expected: \n%s\n\nbad: \n%s", expected, actual)
	}
}

// If a data source explicitly depends on another resource, it's because we need
// that resource to be applied first.
func TestContext2Apply_dataDependsOn(t *testing.T) {
	p := testProvider("null")
	m := testModule(t, "apply-data-depends-on")

	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"null": testProviderFuncFixed(p),
			},
		),
	})

	// the "provisioner" here writes to this variable, because the intent is to
	// create a dependency which can't be viewed through the graph, and depends
	// solely on the configuration providing "depends_on"
	provisionerOutput := ""

	p.ApplyFn = func(info *InstanceInfo, s *InstanceState, d *InstanceDiff) (*InstanceState, error) {
		// the side effect of the resource being applied
		provisionerOutput = "APPLIED"
		return testApplyFn(info, s, d)
	}

	p.DiffFn = testDiffFn
	p.ReadDataDiffFn = testDataDiffFn

	p.ReadDataApplyFn = func(*InstanceInfo, *InstanceDiff) (*InstanceState, error) {
		// Read the artifact created by our dependency being applied.
		// Without any "depends_on", this would be skipped as it's assumed the
		// initial diff during refresh was all that's needed.
		return &InstanceState{
			ID: "read",
			Attributes: map[string]string{
				"foo": provisionerOutput,
			},
		}, nil
	}

	_, err := ctx.Refresh()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	root := state.ModuleByPath(RootModulePath)
	actual := root.Resources["data.null_data_source.read"].Primary.Attributes["foo"]

	expected := "APPLIED"
	if actual != expected {
		t.Fatalf("bad:\n%s", strings.TrimSpace(state.String()))
	}
}

func TestContext2Apply_terraformEnv(t *testing.T) {
	m := testModule(t, "apply-terraform-env")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	ctx := testContext2(t, &ContextOpts{
		Meta:   &ContextMeta{Env: "foo"},
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

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := state.RootModule().Outputs["output"]
	expected := "foo"
	if actual == nil || actual.Value != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

// verify that multiple config references only create a single depends_on entry
func TestContext2Apply_multiRef(t *testing.T) {
	m := testModule(t, "apply-multi-ref")
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

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	deps := state.Modules[0].Resources["aws_instance.other"].Dependencies
	if len(deps) > 1 || deps[0] != "aws_instance.create" {
		t.Fatalf("expected 1 depends_on entry for aws_instance.create, got %q", deps)
	}
}

func TestContext2Apply_targetedModuleRecursive(t *testing.T) {
	m := testModule(t, "apply-targeted-module-recursive")
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
		Targets: []string{"module.child"},
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	mod := state.ModuleByPath([]string{"root", "child", "subchild"})
	if mod == nil {
		t.Fatalf("no subchild module found in the state!\n\n%#v", state)
	}
	if len(mod.Resources) != 1 {
		t.Fatalf("expected 1 resources, got: %#v", mod.Resources)
	}

	checkStateString(t, state, `
<no state>
module.child.subchild:
  aws_instance.foo:
    ID = foo
    num = 2
    type = aws_instance
	`)
}

func TestContext2Apply_localVal(t *testing.T) {
	m := testModule(t, "apply-local-val")
	ctx := testContext2(t, &ContextOpts{
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{},
		),
	})

	if _, err := ctx.Plan(); err != nil {
		t.Fatalf("error during plan: %s", err)
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("error during apply: %s", err)
	}

	got := strings.TrimSpace(state.String())
	want := strings.TrimSpace(`
<no state>
Outputs:

result_1 = hello
result_3 = hello world

module.child:
  <no state>
  Outputs:

  result = hello
`)
	if got != want {
		t.Fatalf("wrong final state\ngot:\n%s\nwant:\n%s", got, want)
	}
}
