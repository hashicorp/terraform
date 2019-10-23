package terraform

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-test/deep"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/configs/hcl2shim"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/provisioners"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

func TestContext2Apply_basic(t *testing.T) {
	m := testModule(t, "apply-good")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	mod := state.RootModule()
	if len(mod.Resources) < 2 {
		t.Fatalf("bad: %#v", mod.Resources)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Apply_unstable(t *testing.T) {
	// This tests behavior when the configuration contains an unstable value,
	// such as the result of uuid() or timestamp(), where each call produces
	// a different result.
	//
	// This is an important case to test because we need to ensure that
	// we don't re-call the function during the apply phase: the value should
	// be fixed during plan

	m := testModule(t, "apply-unstable")
	p := testProvider("test")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"test": testProviderFuncFixed(p),
			},
		),
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected error during Plan: %s", diags.Err())
	}

	addr := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test_resource",
		Name: "foo",
	}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)
	schema := p.GetSchemaReturn.ResourceTypes["test_resource"] // automatically available in mock
	rds := plan.Changes.ResourceInstance(addr)
	rd, err := rds.Decode(schema.ImpliedType())
	if err != nil {
		t.Fatal(err)
	}
	if rd.After.GetAttr("random").IsKnown() {
		t.Fatalf("Attribute 'random' has known value %#v; should be unknown in plan", rd.After.GetAttr("random"))
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("unexpected error during Apply: %s", diags.Err())
	}

	mod := state.Module(addr.Module)
	rss := state.ResourceInstance(addr)

	if len(mod.Resources) != 1 {
		t.Fatalf("wrong number of resources %d; want 1", len(mod.Resources))
	}

	rs, err := rss.Current.Decode(schema.ImpliedType())
	got := rs.Value.GetAttr("random")
	if !got.IsKnown() {
		t.Fatalf("random is still unknown after apply")
	}
	if got, want := len(got.AsString()), 36; got != want {
		t.Fatalf("random string has wrong length %d; want %d", got, want)
	}
}

func TestContext2Apply_escape(t *testing.T) {
	m := testModule(t, "apply-escape")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	checkStateString(t, state, `
aws_instance.bar:
  ID = foo
  provider = provider.aws
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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"null": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	assertNoDiagnostics(t, diags)

	got := strings.TrimSpace(state.String())
	want := strings.TrimSpace(`null_resource.foo.0:
  ID = foo
  provider = provider.null

Outputs:

test = [foo]`)
	if got != want {
		t.Fatalf("got:\n%s\n\nwant:\n%s\n", got, want)
	}
}
func TestContext2Apply_resourceCountZeroList(t *testing.T) {
	m := testModule(t, "apply-resource-count-zero-list")
	p := testProvider("null")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"null": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	got := strings.TrimSpace(state.String())
	want := strings.TrimSpace(`Outputs:

test = []`)
	if got != want {
		t.Fatalf("wrong state\n\ngot:\n%s\n\nwant:\n%s\n", got, want)
	}
}

func TestContext2Apply_resourceDependsOnModule(t *testing.T) {
	m := testModule(t, "apply-resource-depends-on-module")
	p := testProvider("aws")
	p.DiffFn = testDiffFn

	// verify the apply happens in the correct order
	var mu sync.Mutex
	var order []string

	p.ApplyFn = func(
		info *InstanceInfo,
		is *InstanceState,
		id *InstanceDiff) (*InstanceState, error) {

		if id.Attributes["ami"].New == "child" {

			// make the child slower than the parent
			time.Sleep(50 * time.Millisecond)

			mu.Lock()
			order = append(order, "child")
			mu.Unlock()
		} else {
			mu.Lock()
			order = append(order, "parent")
			mu.Unlock()
		}

		return testApplyFn(info, is, id)
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	if !reflect.DeepEqual(order, []string{"child", "parent"}) {
		t.Fatal("resources applied out of order")
	}

	checkStateString(t, state, testTerraformApplyResourceDependsOnModuleStr)
}

// Test that without a config, the Dependencies in the state are enough
// to maintain proper ordering.
func TestContext2Apply_resourceDependsOnModuleStateOnly(t *testing.T) {
	m := testModule(t, "apply-resource-depends-on-module-empty")
	p := testProvider("aws")
	p.DiffFn = testDiffFn

	state := MustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.a": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "parent",
						},
						Dependencies: []string{"module.child"},
						Provider:     "provider.aws",
					},
				},
			},
			&ModuleState{
				Path: []string{"root", "child"},
				Resources: map[string]*ResourceState{
					"aws_instance.child": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "child",
						},
						Provider: "provider.aws",
					},
				},
			},
		},
	})

	{
		// verify the apply happens in the correct order
		var mu sync.Mutex
		var order []string

		p.ApplyFn = func(
			info *InstanceInfo,
			is *InstanceState,
			id *InstanceDiff) (*InstanceState, error) {

			if is.ID == "parent" {
				// make the dep slower than the parent
				time.Sleep(50 * time.Millisecond)

				mu.Lock()
				order = append(order, "child")
				mu.Unlock()
			} else {
				mu.Lock()
				order = append(order, "parent")
				mu.Unlock()
			}

			return testApplyFn(info, is, id)
		}

		ctx := testContext2(t, &ContextOpts{
			Config: m,
			ProviderResolver: providers.ResolverFixed(
				map[string]providers.Factory{
					"aws": testProviderFuncFixed(p),
				},
			),
			State: state,
		})

		if _, diags := ctx.Plan(); diags.HasErrors() {
			t.Fatalf("diags: %s", diags.Err())
		}

		state, diags := ctx.Apply()
		assertNoErrors(t, diags)

		if !reflect.DeepEqual(order, []string{"child", "parent"}) {
			t.Fatal("resources applied out of order")
		}

		checkStateString(t, state, "<no state>")
	}
}

func TestContext2Apply_resourceDependsOnModuleDestroy(t *testing.T) {
	m := testModule(t, "apply-resource-depends-on-module")
	p := testProvider("aws")
	p.DiffFn = testDiffFn

	var globalState *states.State
	{
		p.ApplyFn = testApplyFn
		ctx := testContext2(t, &ContextOpts{
			Config: m,
			ProviderResolver: providers.ResolverFixed(
				map[string]providers.Factory{
					"aws": testProviderFuncFixed(p),
				},
			),
		})

		if _, diags := ctx.Plan(); diags.HasErrors() {
			t.Fatalf("diags: %s", diags.Err())
		}

		state, diags := ctx.Apply()
		if diags.HasErrors() {
			t.Fatalf("diags: %s", diags.Err())
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

			if is.Attributes["ami"] == "parent" {
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
			Config: m,
			ProviderResolver: providers.ResolverFixed(
				map[string]providers.Factory{
					"aws": testProviderFuncFixed(p),
				},
			),
			State:   globalState,
			Destroy: true,
		})

		if _, diags := ctx.Plan(); diags.HasErrors() {
			t.Fatalf("diags: %s", diags.Err())
		}

		state, diags := ctx.Apply()
		if diags.HasErrors() {
			t.Fatalf("diags: %s", diags.Err())
		}

		if !checked {
			t.Fatal("should check")
		}

		checkStateString(t, state, `<no state>`)
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

			if id.Attributes["ami"].New == "grandchild" {
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
			Config: m,
			ProviderResolver: providers.ResolverFixed(
				map[string]providers.Factory{
					"aws": testProviderFuncFixed(p),
				},
			),
		})

		if _, diags := ctx.Plan(); diags.HasErrors() {
			t.Fatalf("diags: %s", diags.Err())
		}

		state, diags := ctx.Apply()
		if diags.HasErrors() {
			t.Fatalf("diags: %s", diags.Err())
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
		p.ApplyFn = func(info *InstanceInfo, is *InstanceState, id *InstanceDiff) (*InstanceState, error) {
			if id.Attributes["ami"].New == "grandchild" {
				checked = true

				// Sleep to allow parallel execution
				time.Sleep(50 * time.Millisecond)

				// Verify that called is 0 (dep not called)
				if atomic.LoadInt32(&called) != 0 {
					return nil, fmt.Errorf("something else was applied before grandchild; grandchild should be first")
				}
			}

			atomic.AddInt32(&called, 1)
			return testApplyFn(info, is, id)
		}

		ctx := testContext2(t, &ContextOpts{
			Config: m,
			ProviderResolver: providers.ResolverFixed(
				map[string]providers.Factory{
					"aws": testProviderFuncFixed(p),
				},
			),
		})

		if _, diags := ctx.Plan(); diags.HasErrors() {
			t.Fatalf("diags: %s", diags.Err())
		}

		state, diags := ctx.Apply()
		if diags.HasErrors() {
			t.Fatalf("diags: %s", diags.Err())
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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"null": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(`<no state>
Outputs:

amis_from_module = {eu-west-1:ami-789012 eu-west-2:ami-989484 us-west-1:ami-123456 us-west-2:ami-456789 }

module.test:
  null_resource.noop:
    ID = foo
    provider = provider.null

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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	mod := state.RootModule()
	if len(mod.Resources) < 2 {
		t.Fatalf("bad: %#v", mod.Resources)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyRefCountStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Apply_providerAlias(t *testing.T) {
	m := testModule(t, "apply-provider-alias")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	mod := state.RootModule()
	if len(mod.Resources) < 2 {
		t.Fatalf("bad: %#v", mod.Resources)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyProviderAliasStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

// Two providers that are configured should both be configured prior to apply
func TestContext2Apply_providerAliasConfigure(t *testing.T) {
	m := testModule(t, "apply-provider-alias-configure")

	p2 := testProvider("another")
	p2.ApplyFn = testApplyFn
	p2.DiffFn = testDiffFn

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"another": testProviderFuncFixed(p2),
			},
		),
	})

	if p, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	} else {
		t.Logf(legacyDiffComparisonString(p.Changes))
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

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	if configCount != 2 {
		t.Fatalf("provider config expected 2 calls, got: %d", configCount)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyProviderAliasConfigStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(`
aws_instance.foo:
  ID = foo
  provider = provider.aws
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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
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
	state := MustShimLegacyState(&State{
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
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: state,
	})

	if p, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	} else {
		t.Logf(legacyDiffComparisonString(p.Changes))
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	mod := state.RootModule()
	if got, want := len(mod.Resources), 1; got != want {
		t.Logf("state:\n%s", state)
		t.Fatalf("wrong number of resources %d; want %d", got, want)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyCreateBeforeStr)
	if actual != expected {
		t.Fatalf("expected:\n%s\ngot:\n%s", expected, actual)
	}
}

func TestContext2Apply_createBeforeDestroyUpdate(t *testing.T) {
	m := testModule(t, "apply-good-create-before-update")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	state := MustShimLegacyState(&State{
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
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: state,
	})

	if p, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	} else {
		t.Logf(legacyDiffComparisonString(p.Changes))
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	mod := state.RootModule()
	if len(mod.Resources) != 1 {
		t.Fatalf("bad: %s", state)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyCreateBeforeUpdateStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

// This tests that when a CBD resource depends on a non-CBD resource,
// we can still properly apply changes that require new for both.
func TestContext2Apply_createBeforeDestroy_dependsNonCBD(t *testing.T) {
	m := testModule(t, "apply-cbd-depends-non-cbd")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	state := MustShimLegacyState(&State{
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
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: state,
	})

	if p, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	} else {
		t.Logf(legacyDiffComparisonString(p.Changes))
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	checkStateString(t, state, `
aws_instance.bar:
  ID = foo
  provider = provider.aws
  require_new = yes
  type = aws_instance
  value = foo

  Dependencies:
    aws_instance.foo
aws_instance.foo:
  ID = foo
  provider = provider.aws
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
	state := MustShimLegacyState(&State{
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
						Provider: "provider.aws",
					},
				},
			},
		},
	})

	var actual []cty.Value
	var actualLock sync.Mutex
	h.PostApplyFn = func(addr addrs.AbsResourceInstance, gen states.Generation, sv cty.Value, e error) (HookAction, error) {
		actualLock.Lock()

		defer actualLock.Unlock()
		actual = append(actual, sv)
		return HookActionContinue, nil
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Hooks:  []Hook{h},
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: state,
	})

	if p, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	} else {
		t.Logf(legacyDiffComparisonString(p.Changes))
	}

	if _, diags := ctx.Apply(); diags.HasErrors() {
		t.Fatalf("apply errors: %s", diags.Err())
	}

	expected := []cty.Value{
		cty.ObjectVal(map[string]cty.Value{
			"id":          cty.StringVal("foo"),
			"require_new": cty.StringVal("xyz"),
			"type":        cty.StringVal("aws_instance"),
		}),
		cty.NullVal(cty.DynamicPseudoType),
	}

	cmpOpt := cmp.Transformer("ctyshim", hcl2shim.ConfigValueFromHCL2)
	if !cmp.Equal(actual, expected, cmpOpt) {
		t.Fatalf("wrong state snapshot sequence\n%s", cmp.Diff(expected, actual, cmpOpt))
	}
}

// Test that we can perform an apply with CBD in a count with deposed instances.
func TestContext2Apply_createBeforeDestroy_deposedCount(t *testing.T) {
	m := testModule(t, "apply-cbd-count")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	state := MustShimLegacyState(&State{
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
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: state,
	})

	if p, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	} else {
		t.Logf(legacyDiffComparisonString(p.Changes))
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	checkStateString(t, state, `
aws_instance.bar.0:
  ID = foo
  provider = provider.aws
  foo = bar
  type = aws_instance
aws_instance.bar.1:
  ID = foo
  provider = provider.aws
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

	state := MustShimLegacyState(&State{
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
						Provider: "provider.aws",
					},
				},
			},
		},
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: state,
	})

	if p, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	} else {
		t.Logf(legacyDiffComparisonString(p.Changes))
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	checkStateString(t, state, `
aws_instance.bar:
  ID = bar
  provider = provider.aws
	`)
}

func TestContext2Apply_destroyComputed(t *testing.T) {
	m := testModule(t, "apply-destroy-computed")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	state := MustShimLegacyState(&State{
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
						Provider: "provider.aws",
					},
				},
			},
		},
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State:   state,
		Destroy: true,
	})

	if p, diags := ctx.Plan(); diags.HasErrors() {
		logDiagnostics(t, diags)
		t.Fatal("plan failed")
	} else {
		t.Logf("plan:\n\n%s", legacyDiffComparisonString(p.Changes))
	}

	if _, diags := ctx.Apply(); diags.HasErrors() {
		logDiagnostics(t, diags)
		t.Fatal("apply failed")
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
	state := MustShimLegacyState(&State{
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
	})

	// Record the order we see Apply
	var actual []string
	var actualLock sync.Mutex
	p.ApplyFn = func(
		_ *InstanceInfo, is *InstanceState, _ *InstanceDiff) (*InstanceState, error) {
		actualLock.Lock()
		defer actualLock.Unlock()
		actual = append(actual, is.ID)
		return nil, nil
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State:       state,
		Destroy:     true,
		Parallelism: 1, // To check ordering
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	if _, diags := ctx.Apply(); diags.HasErrors() {
		t.Fatalf("apply errors: %s", diags.Err())
	}

	expected := []string{"foo", "bar"}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("wrong order\ngot:  %#v\nwant: %#v", actual, expected)
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
	state := MustShimLegacyState(&State{
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
						Provider: "provider.aws",
					},

					"aws_instance.bar": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID:         "bar",
							Attributes: map[string]string{},
						},
						Dependencies: []string{"aws_instance.foo"},
						Provider:     "provider.aws",
					},
				},
			},
		},
	})

	// Record the order we see Apply
	var actual []string
	var actualLock sync.Mutex
	p.ApplyFn = func(
		_ *InstanceInfo, is *InstanceState, _ *InstanceDiff) (*InstanceState, error) {
		actualLock.Lock()
		defer actualLock.Unlock()
		actual = append(actual, is.ID)
		return nil, nil
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State:       state,
		Destroy:     true,
		Parallelism: 1, // To check ordering
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	if _, diags := ctx.Apply(); diags.HasErrors() {
		t.Fatalf("apply errors: %s", diags.Err())
	}

	expected := []string{"bar", "foo"}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("wrong order\ngot:  %#v\nwant: %#v", actual, expected)
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
	state := MustShimLegacyState(&State{
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
						Provider: "provider.aws",
					},

					"aws_instance.bar": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID:         "bar",
							Attributes: map[string]string{},
						},
						Dependencies: []string{"aws_instance.foo"},
						Provider:     "provider.aws",
					},
				},
			},
		},
	})

	// Record the order we see Apply
	var actual []string
	var actualLock sync.Mutex
	p.ApplyFn = func(
		_ *InstanceInfo, is *InstanceState, _ *InstanceDiff) (*InstanceState, error) {
		actualLock.Lock()
		defer actualLock.Unlock()
		actual = append(actual, is.ID)
		return nil, nil
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State:       state,
		Destroy:     true,
		Parallelism: 1, // To check ordering
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	if _, diags := ctx.Apply(); diags.HasErrors() {
		t.Fatalf("apply errors: %s", diags.Err())
	}

	expected := []string{"bar", "foo"}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("wrong order\ngot:  %#v\nwant: %#v", actual, expected)
	}
}

func TestContext2Apply_dataBasic(t *testing.T) {
	m := testModule(t, "apply-data-basic")
	p := testProvider("null")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	p.ReadDataSourceResponse = providers.ReadDataSourceResponse{
		State: cty.ObjectVal(map[string]cty.Value{
			"id":  cty.StringVal("yo"),
			"foo": cty.NullVal(cty.String),
		}),
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"null": testProviderFuncFixed(p),
			},
		),
	})

	if p, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	} else {
		t.Logf(legacyDiffComparisonString(p.Changes))
	}

	state, diags := ctx.Apply()
	assertNoErrors(t, diags)

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyDataBasicStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Apply_destroyData(t *testing.T) {
	m := testModule(t, "apply-destroy-data-resource")
	p := testProvider("null")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	state := MustShimLegacyState(&State{
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
	})
	hook := &testHook{}
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"null": testProviderFuncFixed(p),
			},
		),
		State:   state,
		Destroy: true,
		Hooks:   []Hook{hook},
	})

	if p, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	} else {
		t.Logf(legacyDiffComparisonString(p.Changes))
	}

	newState, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	if got := len(newState.Modules); got != 1 {
		t.Fatalf("state has %d modules after destroy; want 1", got)
	}

	if got := len(newState.RootModule().Resources); got != 0 {
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
	state := MustShimLegacyState(&State{
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
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State:   state,
		Destroy: true,
	})

	if p, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	} else {
		t.Logf(legacyDiffComparisonString(p.Changes))
	}

	if _, diags := ctx.Apply(); diags.HasErrors() {
		t.Fatalf("apply errors: %s", diags.Err())
	}
}

func TestContext2Apply_destroyModuleVarProviderConfig(t *testing.T) {
	m := testModule(t, "apply-destroy-mod-var-provider-config")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	state := MustShimLegacyState(&State{
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
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State:   state,
		Destroy: true,
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	_, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}
}

// https://github.com/hashicorp/terraform/issues/2892
func TestContext2Apply_destroyCrossProviders(t *testing.T) {
	m := testModule(t, "apply-destroy-cross-providers")

	p_aws := testProvider("aws")
	p_aws.ApplyFn = testApplyFn
	p_aws.DiffFn = testDiffFn
	p_aws.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
				},
			},
			"aws_vpc": {
				Attributes: map[string]*configschema.Attribute{
					"value": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
		},
	}

	providers := map[string]providers.Factory{
		"aws": testProviderFuncFixed(p_aws),
	}

	// Bug only appears from time to time,
	// so we run this test multiple times
	// to check for the race-condition

	// FIXME: this test flaps now, so run it more times
	for i := 0; i <= 100; i++ {
		ctx := getContextForApply_destroyCrossProviders(t, m, providers)

		if _, diags := ctx.Plan(); diags.HasErrors() {
			logDiagnostics(t, diags)
			t.Fatal("plan failed")
		}

		if _, diags := ctx.Apply(); diags.HasErrors() {
			logDiagnostics(t, diags)
			t.Fatal("apply failed")
		}
	}
}

func getContextForApply_destroyCrossProviders(t *testing.T, m *configs.Config, providerFactories map[string]providers.Factory) *Context {
	state := MustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.shared": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "remote-2652591293",
							Attributes: map[string]string{
								"id": "test",
							},
						},
						Provider: "provider.aws",
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
						Provider: "provider.aws",
					},
				},
			},
		},
	})
	ctx := testContext2(t, &ContextOpts{
		Config:           m,
		ProviderResolver: providers.ResolverFixed(providerFactories),
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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyMinimalStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Apply_badDiff(t *testing.T) {
	m := testModule(t, "apply-good")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	p.DiffFn = func(*InstanceInfo, *InstanceState, *ResourceConfig) (*InstanceDiff, error) {
		return &InstanceDiff{
			Attributes: map[string]*ResourceAttrDiff{
				"newp": &ResourceAttrDiff{
					Old:         "",
					New:         "",
					NewComputed: true,
				},
			},
		}, nil
	}

	if _, diags := ctx.Apply(); diags == nil {
		t.Fatal("should error")
	}
}

func TestContext2Apply_cancel(t *testing.T) {
	stopped := false

	m := testModule(t, "apply-cancel")
	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
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
				"value": "2",
			},
		}, nil
	}
	p.DiffFn = func(info *InstanceInfo, s *InstanceState, rc *ResourceConfig) (*InstanceDiff, error) {
		d := &InstanceDiff{
			Attributes: map[string]*ResourceAttrDiff{},
		}
		if new, ok := rc.Get("value"); ok {
			d.Attributes["value"] = &ResourceAttrDiff{
				New: new.(string),
			}
		}
		if new, ok := rc.Get("foo"); ok {
			d.Attributes["foo"] = &ResourceAttrDiff{
				New: new.(string),
			}
		}
		return d, nil
	}

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	// Start the Apply in a goroutine
	var applyDiags tfdiags.Diagnostics
	stateCh := make(chan *states.State)
	go func() {
		state, diags := ctx.Apply()
		applyDiags = diags

		stateCh <- state
	}()

	state := <-stateCh
	if applyDiags.HasErrors() {
		t.Fatalf("unexpected errors: %s", applyDiags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyCancelStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}

	if !p.StopCalled {
		t.Fatal("stop should be called")
	}
}

func TestContext2Apply_cancelBlock(t *testing.T) {
	m := testModule(t, "apply-cancel-block")
	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	applyCh := make(chan struct{})
	p.DiffFn = func(*InstanceInfo, *InstanceState, *ResourceConfig) (*InstanceDiff, error) {
		return &InstanceDiff{
			Attributes: map[string]*ResourceAttrDiff{
				"id": &ResourceAttrDiff{
					New: "foo",
				},
				"num": &ResourceAttrDiff{
					New: "2",
				},
			},
		}, nil
	}
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
			Attributes: map[string]string{
				"num": "2",
			},
		}, nil
	}

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	// Start the Apply in a goroutine
	var applyDiags tfdiags.Diagnostics
	stateCh := make(chan *states.State)
	go func() {
		state, diags := ctx.Apply()
		applyDiags = diags

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
	if applyDiags.HasErrors() {
		t.Fatalf("unexpected error: %s", applyDiags.Err())
	}

	checkStateString(t, state, `
aws_instance.foo:
  ID = foo
  provider = provider.aws
  num = 2
	`)
}

// for_each values cannot be used in the provisioner during destroy.
// There may be a way to handle this, but for now make sure we print an error
// rather than crashing with an invalid config.
func TestContext2Apply_provisionerDestroyForEach(t *testing.T) {
	m := testModule(t, "apply-provisioner-each")
	p := testProvider("aws")
	pr := testProvisioner()
	p.DiffFn = testDiffFn
	p.ApplyFn = testApplyFn

	s := &states.State{
		Modules: map[string]*states.Module{
			"": &states.Module{
				Resources: map[string]*states.Resource{
					"aws_instance.bar": &states.Resource{
						Addr:     addrs.Resource{Mode: 77, Type: "aws_instance", Name: "bar"},
						EachMode: states.EachMap,
						Instances: map[addrs.InstanceKey]*states.ResourceInstance{
							addrs.StringKey("a"): &states.ResourceInstance{
								Current: &states.ResourceInstanceObjectSrc{
									AttrsJSON: []byte(`{"foo":"bar","id":"foo"}`),
								},
							},
							addrs.StringKey("b"): &states.ResourceInstance{
								Current: &states.ResourceInstanceObjectSrc{
									AttrsJSON: []byte(`{"foo":"bar","id":"foo"}`),
								},
							},
						},
						ProviderConfig: addrs.AbsProviderConfig{
							Module:         addrs.ModuleInstance(nil),
							ProviderConfig: addrs.ProviderConfig{Type: "aws", Alias: ""},
						},
					},
				},
			},
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
		State:   s,
		Destroy: true,
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	_, diags := ctx.Apply()
	if diags == nil {
		t.Fatal("should error")
	}
	if !strings.Contains(diags.Err().Error(), "each.value is unknown and cannot be used in this context") {
		t.Fatal("unexpected error:", diags.Err())
	}
}

func TestContext2Apply_cancelProvisioner(t *testing.T) {
	m := testModule(t, "apply-cancel-provisioner")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	pr := testProvisioner()
	pr.GetSchemaResponse = provisioners.GetSchemaResponse{
		Provisioner: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"foo": {
					Type:     cty.String,
					Optional: true,
				},
			},
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ProvisionerFactory{
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

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	// Start the Apply in a goroutine
	var applyDiags tfdiags.Diagnostics
	stateCh := make(chan *states.State)
	go func() {
		state, diags := ctx.Apply()
		applyDiags = diags

		stateCh <- state
	}()

	// Wait for completion
	state := <-stateCh
	assertNoErrors(t, applyDiags)

	checkStateString(t, state, `
aws_instance.foo: (tainted)
  ID = foo
  provider = provider.aws
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
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"num": {
						Type:     cty.Number,
						Optional: true,
					},
					"compute": {
						Type:     cty.String,
						Optional: true,
					},
					"compute_value": {
						Type:     cty.String,
						Optional: true,
					},
					"foo": {
						Type:     cty.String,
						Optional: true,
					},
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"type": {
						Type:     cty.String,
						Computed: true,
					},
					"value": { // Populated from compute_value because compute = "value" in the config fixture
						Type:     cty.String,
						Computed: true,
					},
				},
			},
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	ctx.variables = InputValues{
		"value": &InputValue{
			Value:      cty.NumberIntVal(1),
			SourceType: ValueFromCaller,
		},
	}

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyComputeStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Apply_countDecrease(t *testing.T) {
	m := testModule(t, "apply-count-dec")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	p.ApplyFn = testApplyFn
	s := MustShimLegacyState(&State{
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
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		logDiagnostics(t, diags)
		t.Fatal("plan failed")
	}

	state, diags := ctx.Apply()
	assertNoErrors(t, diags)

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyCountDecStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Apply_countDecreaseToOneX(t *testing.T) {
	m := testModule(t, "apply-count-dec-one")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	s := MustShimLegacyState(&State{
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
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyCountDecToOneStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
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
	s := MustShimLegacyState(&State{
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
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	if p, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	} else {
		got := strings.TrimSpace(legacyPlanComparisonString(ctx.State(), p.Changes))
		want := strings.TrimSpace(testTerraformApplyCountDecToOneCorruptedPlanStr)
		if got != want {
			t.Fatalf("wrong plan result\ngot:\n%s\nwant:\n%s", got, want)
		}
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyCountDecToOneCorruptedStr)
	if actual != expected {
		t.Fatalf("wrong final state\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Apply_countTainted(t *testing.T) {
	m := testModule(t, "apply-count-tainted")
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	p.ApplyFn = testApplyFn
	s := MustShimLegacyState(&State{
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
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	{
		plan, diags := ctx.Plan()
		assertNoErrors(t, diags)
		got := strings.TrimSpace(legacyDiffComparisonString(plan.Changes))
		want := strings.TrimSpace(`
DESTROY/CREATE: aws_instance.foo[0]
  foo:  "foo" => "foo"
  id:   "bar" => "<computed>"
  type: "aws_instance" => "aws_instance"
CREATE: aws_instance.foo[1]
  foo:  "" => "foo"
  id:   "" => "<computed>"
  type: "" => "aws_instance"
`)
		if got != want {
			t.Fatalf("wrong plan\n\ngot:\n%s\n\nwant:\n%s", got, want)
		}
	}

	state, diags := ctx.Apply()
	assertNoErrors(t, diags)

	got := strings.TrimSpace(state.String())
	want := strings.TrimSpace(`
aws_instance.foo.0:
  ID = foo
  provider = provider.aws
  foo = foo
  type = aws_instance
aws_instance.foo.1:
  ID = foo
  provider = provider.aws
  foo = foo
  type = aws_instance
`)
	if got != want {
		t.Fatalf("wrong final state\n\ngot:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestContext2Apply_countVariable(t *testing.T) {
	m := testModule(t, "apply-count-variable")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyCountVariableStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Apply_countVariableRef(t *testing.T) {
	m := testModule(t, "apply-count-variable-ref")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyCountVariableRefStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Apply_provisionerInterpCount(t *testing.T) {
	// This test ensures that a provisioner can interpolate a resource count
	// even though the provisioner expression is evaluated during the plan
	// walk. https://github.com/hashicorp/terraform/issues/16840

	m, snap := testModuleWithSnapshot(t, "apply-provisioner-interp-count")

	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	pr := testProvisioner()

	providerResolver := providers.ResolverFixed(
		map[string]providers.Factory{
			"aws": testProviderFuncFixed(p),
		},
	)
	provisioners := map[string]ProvisionerFactory{
		"local-exec": testProvisionerFuncFixed(pr),
	}
	ctx := testContext2(t, &ContextOpts{
		Config:           m,
		ProviderResolver: providerResolver,
		Provisioners:     provisioners,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("plan failed unexpectedly: %s", diags.Err())
	}

	state := ctx.State()

	// We'll marshal and unmarshal the plan here, to ensure that we have
	// a clean new context as would be created if we separately ran
	// terraform plan -out=tfplan && terraform apply tfplan
	ctxOpts, err := contextOptsForPlanViaFile(snap, state, plan)
	if err != nil {
		t.Fatal(err)
	}
	ctxOpts.ProviderResolver = providerResolver
	ctxOpts.Provisioners = provisioners
	ctx, diags = NewContext(ctxOpts)
	if diags.HasErrors() {
		t.Fatalf("failed to create context for plan: %s", diags.Err())
	}

	// Applying the plan should now succeed
	_, diags = ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("apply failed unexpectedly: %s", diags.Err())
	}

	// Verify apply was invoked
	if !pr.ProvisionResourceCalled {
		t.Fatalf("provisioner was not called")
	}
}

func TestContext2Apply_foreachVariable(t *testing.T) {
	m := testModule(t, "plan-for-each-unknown-value")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Variables: InputValues{
			"foo": &InputValue{
				Value: cty.StringVal("hello"),
			},
		},
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyForEachVariableStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Apply_moduleBasic(t *testing.T) {
	m := testModule(t, "apply-module")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
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

		if is.ID == "b" {
			// Pause briefly to make any race conditions more visible, since
			// missing edges here can cause undeterministic ordering.
			time.Sleep(100 * time.Millisecond)
		}

		orderLock.Lock()
		defer orderLock.Unlock()

		order = append(order, is.ID)
		return nil, nil
	}

	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"id":    {Type: cty.String, Required: true},
					"blah":  {Type: cty.String, Optional: true},
					"value": {Type: cty.String, Optional: true},
				},
			},
		},
	}

	state := MustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.b": resourceState("aws_instance", "b"),
				},
			},

			&ModuleState{
				Path: []string{"root", "child"},
				Resources: map[string]*ResourceState{
					"aws_instance.a": resourceState("aws_instance", "a"),
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
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State:   state,
		Destroy: true,
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	expected := []string{"b", "a"}
	if !reflect.DeepEqual(order, expected) {
		t.Errorf("wrong order\ngot: %#v\nwant: %#v", order, expected)
	}

	{
		actual := strings.TrimSpace(state.String())
		expected := strings.TrimSpace(testTerraformApplyModuleDestroyOrderStr)
		if actual != expected {
			t.Errorf("wrong final state\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
		}
	}
}

func TestContext2Apply_moduleInheritAlias(t *testing.T) {
	m := testModule(t, "apply-module-provider-inherit-alias")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	p.ConfigureFn = func(c *ResourceConfig) error {
		if _, ok := c.Get("value"); !ok {
			return nil
		}

		if _, ok := c.Get("root"); ok {
			return fmt.Errorf("child should not get root")
		}

		return nil
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	checkStateString(t, state, `
<no state>
module.child:
  aws_instance.foo:
    ID = foo
    provider = provider.aws.eu
	`)
}

func TestContext2Apply_orphanResource(t *testing.T) {
	// This is a two-step test:
	// 1. Apply a configuration with resources that have count set.
	//    This should place the empty resource object in the state to record
	//    that each exists, and record any instances.
	// 2. Apply an empty configuration against the same state, which should
	//    then clean up both the instances and the containing resource objects.
	p := testProvider("test")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_thing": {},
		},
	}

	// Step 1: create the resources and instances
	m := testModule(t, "apply-orphan-resource")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"test": testProviderFuncFixed(p),
			},
		),
	})
	_, diags := ctx.Plan()
	assertNoErrors(t, diags)
	state, diags := ctx.Apply()
	assertNoErrors(t, diags)

	// At this point both resources should be recorded in the state, along
	// with the single instance associated with test_thing.one.
	want := states.BuildState(func(s *states.SyncState) {
		providerAddr := addrs.ProviderConfig{
			Type: "test",
		}.Absolute(addrs.RootModuleInstance)
		zeroAddr := addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_thing",
			Name: "zero",
		}.Absolute(addrs.RootModuleInstance)
		oneAddr := addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_thing",
			Name: "one",
		}.Absolute(addrs.RootModuleInstance)
		s.SetResourceMeta(zeroAddr, states.EachList, providerAddr)
		s.SetResourceMeta(oneAddr, states.EachList, providerAddr)
		s.SetResourceInstanceCurrent(oneAddr.Instance(addrs.IntKey(0)), &states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{}`),
		}, providerAddr)
	})
	if !cmp.Equal(state, want) {
		t.Fatalf("wrong state after step 1\n%s", cmp.Diff(want, state))
	}

	// Step 2: update with an empty config, to destroy everything
	m = testModule(t, "empty")
	ctx = testContext2(t, &ContextOpts{
		Config: m,
		State:  state,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"test": testProviderFuncFixed(p),
			},
		),
	})
	_, diags = ctx.Plan()
	assertNoErrors(t, diags)
	state, diags = ctx.Apply()
	assertNoErrors(t, diags)

	// The state should now be _totally_ empty, with just an empty root module
	// (since that always exists) and no resources at all.
	want = states.NewState()
	if !cmp.Equal(state, want) {
		t.Fatalf("wrong state after step 2\ngot: %swant: %s", spew.Sdump(state), spew.Sdump(want))
	}

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
	state := MustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: []string{"root", "child"},
				Resources: map[string]*ResourceState{
					"aws_instance.bar": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
						},
						Provider: "provider.aws.eu",
					},
				},
			},
		},
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		State:  state,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	if !called {
		t.Fatal("must call configure")
	}

	checkStateString(t, state, "<no state>")
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
	state := MustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: []string{"root", "child"},
				Resources: map[string]*ResourceState{
					"aws_instance.bar": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
						},
						Provider: "provider.aws",
					},
				},
			},
		},
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		State:  state,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	if _, diags := ctx.Apply(); diags.HasErrors() {
		t.Fatalf("apply errors: %s", diags.Err())
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
	state := MustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: []string{"root", "parent", "child"},
				Resources: map[string]*ResourceState{
					"aws_instance.bar": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
						},
						Provider: "provider.aws",
					},
				},
			},
		},
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		State:  state,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	if _, diags := ctx.Apply(); diags.HasErrors() {
		t.Fatalf("apply errors: %s", diags.Err())
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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	if _, diags := ctx.Apply(); diags.HasErrors() {
		t.Fatalf("apply errors: %s", diags.Err())
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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws":  testProviderFuncFixed(p),
				"test": testProviderFuncFixed(pTest),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyModuleOnlyProviderStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Apply_moduleProviderAlias(t *testing.T) {
	m := testModule(t, "apply-module-provider-alias")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyModuleProviderAliasStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Apply_moduleProviderAliasTargets(t *testing.T) {
	m := testModule(t, "apply-module-provider-alias")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Targets: []addrs.Targetable{
			addrs.AbsResource{
				Module: addrs.RootModuleInstance,
				Resource: addrs.Resource{
					Mode: addrs.ManagedResourceMode,
					Type: "nonexistent",
					Name: "thing",
				},
			},
		},
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(`
<no state>
	`)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Apply_moduleProviderCloseNested(t *testing.T) {
	m := testModule(t, "apply-module-provider-close-nested")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: MustShimLegacyState(&State{
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
		}),
		Destroy: true,
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	if _, diags := ctx.Apply(); diags.HasErrors() {
		t.Fatalf("apply errors: %s", diags.Err())
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

	state := MustShimLegacyState(&State{
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
						Provider: "provider.aws",
					},
				},
			},
		},
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: state,
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyModuleVarRefExistingStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Apply_moduleVarResourceCount(t *testing.T) {
	m := testModule(t, "apply-module-var-resource-count")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Variables: InputValues{
			"num": &InputValue{
				Value:      cty.NumberIntVal(2),
				SourceType: ValueFromCaller,
			},
		},
		Destroy: true,
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	if _, diags := ctx.Apply(); diags.HasErrors() {
		t.Fatalf("apply errors: %s", diags.Err())
	}

	ctx = testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Variables: InputValues{
			"num": &InputValue{
				Value:      cty.NumberIntVal(5),
				SourceType: ValueFromCaller,
			},
		},
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	if _, diags := ctx.Apply(); diags.HasErrors() {
		t.Fatalf("apply errors: %s", diags.Err())
	}
}

// GH-819
func TestContext2Apply_moduleBool(t *testing.T) {
	m := testModule(t, "apply-module-bool")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyModuleBoolStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Child("B", addrs.NoKey),
		},
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	checkStateString(t, state, `
<no state>
module.A:
  aws_instance.foo:
    ID = foo
    provider = provider.aws
    foo = bar
    type = aws_instance

  Outputs:

  value = foo
module.B:
  aws_instance.bar:
    ID = foo
    provider = provider.aws
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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
				"do":  testProviderFuncFixed(pDO),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	mod := state.RootModule()
	if len(mod.Resources) < 2 {
		t.Fatalf("bad: %#v", mod.Resources)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyMultiProviderStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Apply_multiProviderDestroy(t *testing.T) {
	m := testModule(t, "apply-multi-provider-destroy")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	p.GetSchemaReturn = &ProviderSchema{
		Provider: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"addr": {Type: cty.String, Optional: true},
			},
		},
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
	}

	p2 := testProvider("vault")
	p2.ApplyFn = testApplyFn
	p2.DiffFn = testDiffFn
	p2.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"vault_instance": {
				Attributes: map[string]*configschema.Attribute{
					"id": {Type: cty.String, Computed: true},
				},
			},
		},
	}

	var state *states.State

	// First, create the instances
	{
		ctx := testContext2(t, &ContextOpts{
			Config: m,
			ProviderResolver: providers.ResolverFixed(
				map[string]providers.Factory{
					"aws":   testProviderFuncFixed(p),
					"vault": testProviderFuncFixed(p2),
				},
			),
		})

		if _, diags := ctx.Plan(); diags.HasErrors() {
			t.Fatalf("errors during create plan: %s", diags.Err())
		}

		s, diags := ctx.Apply()
		if diags.HasErrors() {
			t.Fatalf("errors during create apply: %s", diags.Err())
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

			if info.Type == "aws_instance" {
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
			Config:  m,
			ProviderResolver: providers.ResolverFixed(
				map[string]providers.Factory{
					"aws":   testProviderFuncFixed(p),
					"vault": testProviderFuncFixed(p2),
				},
			),
		})

		if _, diags := ctx.Plan(); diags.HasErrors() {
			t.Fatalf("errors during destroy plan: %s", diags.Err())
		}

		s, diags := ctx.Apply()
		if diags.HasErrors() {
			t.Fatalf("errors during destroy apply: %s", diags.Err())
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
	p.GetSchemaReturn = &ProviderSchema{
		Provider: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"value": {Type: cty.String, Optional: true},
			},
		},
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
	}

	p2 := testProvider("vault")
	p2.ApplyFn = testApplyFn
	p2.DiffFn = testDiffFn
	p2.GetSchemaReturn = &ProviderSchema{
		Provider: &configschema.Block{},
		ResourceTypes: map[string]*configschema.Block{
			"vault_instance": {
				Attributes: map[string]*configschema.Attribute{
					"id": {Type: cty.String, Computed: true},
				},
			},
		},
	}

	var state *states.State

	// First, create the instances
	{
		ctx := testContext2(t, &ContextOpts{
			Config: m,
			ProviderResolver: providers.ResolverFixed(
				map[string]providers.Factory{
					"aws":   testProviderFuncFixed(p),
					"vault": testProviderFuncFixed(p2),
				},
			),
		})

		if _, diags := ctx.Plan(); diags.HasErrors() {
			t.Fatalf("diags: %s", diags.Err())
		}

		s, diags := ctx.Apply()
		if diags.HasErrors() {
			t.Fatalf("diags: %s", diags.Err())
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

			if info.Type == "aws_instance" {
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
			Config:  m,
			ProviderResolver: providers.ResolverFixed(
				map[string]providers.Factory{
					"aws":   testProviderFuncFixed(p),
					"vault": testProviderFuncFixed(p2),
				},
			),
		})

		if _, diags := ctx.Plan(); diags.HasErrors() {
			t.Fatalf("diags: %s", diags.Err())
		}

		s, diags := ctx.Apply()
		if diags.HasErrors() {
			t.Fatalf("diags: %s", diags.Err())
		}

		if !checked {
			t.Fatal("should be checked")
		}

		state = s
	}

	checkStateString(t, state, `
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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Variables: InputValues{
			"num": &InputValue{
				Value:      cty.NumberIntVal(3),
				SourceType: ValueFromCaller,
			},
		},
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	actual := state.RootModule().OutputValues["output"]
	expected := cty.StringVal("bar0,bar1,bar2")
	if actual == nil || actual.Value != expected {
		t.Fatalf("wrong value\ngot:  %#v\nwant: %#v", actual.Value, expected)
	}

	t.Logf("Initial state: %s", state.String())

	// Apply again, reduce the count to 1
	{
		ctx := testContext2(t, &ContextOpts{
			Config: m,
			State:  state,
			ProviderResolver: providers.ResolverFixed(
				map[string]providers.Factory{
					"aws": testProviderFuncFixed(p),
				},
			),
			Variables: InputValues{
				"num": &InputValue{
					Value:      cty.NumberIntVal(1),
					SourceType: ValueFromCaller,
				},
			},
		})

		if _, diags := ctx.Plan(); diags.HasErrors() {
			t.Fatalf("diags: %s", diags.Err())
		}

		state, diags := ctx.Apply()
		if diags.HasErrors() {
			t.Fatalf("diags: %s", diags.Err())
		}

		t.Logf("End state: %s", state.String())

		actual := state.RootModule().OutputValues["output"]
		if actual == nil {
			t.Fatal("missing output")
		}

		expected := cty.StringVal("bar0")
		if actual.Value != expected {
			t.Fatalf("wrong value\ngot:  %#v\nwant: %#v", actual.Value, expected)
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
	var configsLock sync.Mutex

	p.ApplyFn = testApplyFn

	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		proposed := req.ProposedNewState
		configsLock.Lock()
		defer configsLock.Unlock()
		key := proposed.GetAttr("key").AsString()
		// This test was originally written using the legacy p.DiffFn interface,
		// and so the assertions below expect an old-style ResourceConfig, which
		// we'll construct via our shim for now to avoid rewriting all of the
		// assertions.
		configs[key] = NewResourceConfigShimmed(req.Config, p.GetSchemaReturn.ResourceTypes["test_thing"])

		retVals := make(map[string]cty.Value)
		for it := proposed.ElementIterator(); it.Next(); {
			idxVal, val := it.Element()
			idx := idxVal.AsString()

			switch idx {
			case "id":
				retVals[idx] = cty.UnknownVal(cty.String)
			case "name":
				retVals[idx] = cty.StringVal(key)
			default:
				retVals[idx] = val
			}
		}

		return providers.PlanResourceChangeResponse{
			PlannedState: cty.ObjectVal(retVals),
		}
	}

	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_thing": {
				Attributes: map[string]*configschema.Attribute{
					"key": {Type: cty.String, Required: true},

					"source_id":              {Type: cty.String, Optional: true},
					"source_name":            {Type: cty.String, Optional: true},
					"first_source_id":        {Type: cty.String, Optional: true},
					"first_source_name":      {Type: cty.String, Optional: true},
					"source_ids":             {Type: cty.List(cty.String), Optional: true},
					"source_names":           {Type: cty.List(cty.String), Optional: true},
					"source_ids_from_func":   {Type: cty.List(cty.String), Optional: true},
					"source_names_from_func": {Type: cty.List(cty.String), Optional: true},
					"source_ids_wrapped":     {Type: cty.List(cty.List(cty.String)), Optional: true},
					"source_names_wrapped":   {Type: cty.List(cty.List(cty.String)), Optional: true},

					"id":   {Type: cty.String, Computed: true},
					"name": {Type: cty.String, Computed: true},
				},
			},
		},
	}

	// First, apply with a count of 3
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"test": testProviderFuncFixed(p),
			},
		),
		Variables: InputValues{
			"num": &InputValue{
				Value:      cty.NumberIntVal(3),
				SourceType: ValueFromCaller,
			},
		},
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		logDiagnostics(t, diags)
		t.Fatalf("errors during plan")
	}

	checkConfig := func(key string, want map[string]interface{}) {
		configsLock.Lock()
		defer configsLock.Unlock()

		if _, ok := configs[key]; !ok {
			t.Errorf("no config recorded for %s; expected a configuration", key)
			return
		}
		got := configs[key].Config
		t.Run("config for "+key, func(t *testing.T) {
			want["key"] = key // to avoid doing this for every example
			for _, problem := range deep.Equal(got, want) {
				t.Errorf(problem)
			}
		})
	}

	checkConfig("multi_count_var.0", map[string]interface{}{
		"source_id":   hcl2shim.UnknownVariableValue,
		"source_name": "source.0",
	})
	checkConfig("multi_count_var.2", map[string]interface{}{
		"source_id":   hcl2shim.UnknownVariableValue,
		"source_name": "source.2",
	})
	checkConfig("multi_count_derived.0", map[string]interface{}{
		"source_id":   hcl2shim.UnknownVariableValue,
		"source_name": "source.0",
	})
	checkConfig("multi_count_derived.2", map[string]interface{}{
		"source_id":   hcl2shim.UnknownVariableValue,
		"source_name": "source.2",
	})
	checkConfig("whole_splat", map[string]interface{}{
		"source_ids": []interface{}{
			hcl2shim.UnknownVariableValue,
			hcl2shim.UnknownVariableValue,
			hcl2shim.UnknownVariableValue,
		},
		"source_names": []interface{}{
			"source.0",
			"source.1",
			"source.2",
		},
		"source_ids_from_func": hcl2shim.UnknownVariableValue,
		"source_names_from_func": []interface{}{
			"source.0",
			"source.1",
			"source.2",
		},

		"source_ids_wrapped": []interface{}{
			[]interface{}{
				hcl2shim.UnknownVariableValue,
				hcl2shim.UnknownVariableValue,
				hcl2shim.UnknownVariableValue,
			},
		},
		"source_names_wrapped": []interface{}{
			[]interface{}{
				"source.0",
				"source.1",
				"source.2",
			},
		},

		"first_source_id":   hcl2shim.UnknownVariableValue,
		"first_source_name": "source.0",
	})
	checkConfig("child.whole_splat", map[string]interface{}{
		"source_ids": []interface{}{
			hcl2shim.UnknownVariableValue,
			hcl2shim.UnknownVariableValue,
			hcl2shim.UnknownVariableValue,
		},
		"source_names": []interface{}{
			"source.0",
			"source.1",
			"source.2",
		},

		"source_ids_wrapped": []interface{}{
			[]interface{}{
				hcl2shim.UnknownVariableValue,
				hcl2shim.UnknownVariableValue,
				hcl2shim.UnknownVariableValue,
			},
		},
		"source_names_wrapped": []interface{}{
			[]interface{}{
				"source.0",
				"source.1",
				"source.2",
			},
		},
	})

	t.Run("apply", func(t *testing.T) {
		state, diags := ctx.Apply()
		if diags.HasErrors() {
			t.Fatalf("error during apply: %s", diags.Err())
		}

		want := map[string]interface{}{
			"source_ids": []interface{}{"foo", "foo", "foo"},
			"source_names": []interface{}{
				"source.0",
				"source.1",
				"source.2",
			},
		}
		got := map[string]interface{}{}
		for k, s := range state.RootModule().OutputValues {
			got[k] = hcl2shim.ConfigValueFromHCL2(s.Value)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf(
				"wrong outputs\ngot:  %s\nwant: %s",
				spew.Sdump(got), spew.Sdump(want),
			)
		}
	})
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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	t.Logf("State: %s", state.String())

	actual := state.RootModule().OutputValues["should-be-11"]
	expected := cty.StringVal("index-11")
	if actual == nil || actual.Value != expected {
		t.Fatalf("wrong value\ngot:  %#v\nwant: %#v", actual.Value, expected)
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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	t.Logf("State: %s", state.String())

	actual := state.RootModule().OutputValues["should-be-11"]
	expected := cty.StringVal("baz-index-11")
	if actual == nil || actual.Value != expected {
		t.Fatalf("wrong value\ngot:  %#v\nwant: %#v", actual.Value, expected)
	}
}

// Based on GH-10440 where a graph edge wasn't properly being created
// between a modified resource and a count instance being destroyed.
func TestContext2Apply_multiVarCountDec(t *testing.T) {
	var s *states.State

	// First create resources. Nothing sneaky here.
	{
		m := testModule(t, "apply-multi-var-count-dec")
		p := testProvider("aws")
		p.ApplyFn = testApplyFn
		p.DiffFn = testDiffFn
		ctx := testContext2(t, &ContextOpts{
			Config: m,
			ProviderResolver: providers.ResolverFixed(
				map[string]providers.Factory{
					"aws": testProviderFuncFixed(p),
				},
			),
			Variables: InputValues{
				"num": &InputValue{
					Value:      cty.NumberIntVal(2),
					SourceType: ValueFromCaller,
				},
			},
		})

		log.Print("\n========\nStep 1 Plan\n========")
		if _, diags := ctx.Plan(); diags.HasErrors() {
			t.Fatalf("diags: %s", diags.Err())
		}

		log.Print("\n========\nStep 1 Apply\n========")
		state, diags := ctx.Apply()
		if diags.HasErrors() {
			t.Fatalf("diags: %s", diags.Err())
		}

		t.Logf("Step 1 state:\n%s", state)

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

			if id != nil && id.Attributes != nil && id.Attributes["ami"] != nil && id.Attributes["ami"].New == "special" {
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
			Config: m,
			ProviderResolver: providers.ResolverFixed(
				map[string]providers.Factory{
					"aws": testProviderFuncFixed(p),
				},
			),
			Variables: InputValues{
				"num": &InputValue{
					Value:      cty.NumberIntVal(1),
					SourceType: ValueFromCaller,
				},
			},
		})

		log.Print("\n========\nStep 2 Plan\n========")
		plan, diags := ctx.Plan()
		if diags.HasErrors() {
			t.Fatalf("plan errors: %s", diags.Err())
		}

		t.Logf("Step 2 plan:\n%s", legacyDiffComparisonString(plan.Changes))

		log.Print("\n========\nStep 2 Apply\n========")
		state, diags := ctx.Apply()
		if diags.HasErrors() {
			t.Fatalf("apply errors: %s", diags.Err())
		}

		if !checked {
			t.Error("apply never called")
		}

		t.Logf("Step 2 state:\n%s", state)

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
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_thing": {
				Attributes: map[string]*configschema.Attribute{
					"a_ids": {Type: cty.String, Optional: true},
					"id":    {Type: cty.String, Computed: true},
				},
			},
		},
	}

	// First, apply with a count of 3
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"test": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan failed: %s", diags.Err())
	}

	// Before the relevant bug was fixed, Tdiagsaform would panic during apply.
	if _, diags := ctx.Apply(); diags.HasErrors() {
		t.Fatalf("apply failed: %s", diags.Err())
	}

	// If we get here with no errors or panics then our test was successful.
}

func TestContext2Apply_nilDiff(t *testing.T) {
	m := testModule(t, "apply-good")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	p.DiffFn = func(*InstanceInfo, *InstanceState, *ResourceConfig) (*InstanceDiff, error) {
		return nil, nil
	}

	if _, diags := ctx.Apply(); diags == nil {
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
			Config: m,
			ProviderResolver: providers.ResolverFixed(
				map[string]providers.Factory{
					"aws": testProviderFuncFixed(p),
				},
			),
		})

		if _, diags := ctx.Plan(); diags.HasErrors() {
			t.Fatalf("diags: %s", diags.Err())
		}

		state, diags := ctx.Apply()
		if !diags.HasErrors() || !strings.Contains(diags.Err().Error(), "abcd") {
			t.Fatalf("err: %s", diags.Err())
		}

		checkStateString(t, state, `<no state>`)
	}

	{
		// Create the standard apply function and verify we get the output
		p.ApplyFn = testApplyFn

		ctx := testContext2(t, &ContextOpts{
			Config: m,
			ProviderResolver: providers.ResolverFixed(
				map[string]providers.Factory{
					"aws": testProviderFuncFixed(p),
				},
			),
		})

		if _, diags := ctx.Plan(); diags.HasErrors() {
			t.Fatalf("diags: %s", diags.Err())
		}

		state, diags := ctx.Apply()
		if diags.HasErrors() {
			t.Fatalf("diags: %s", diags.Err())
		}

		checkStateString(t, state, `
aws_instance.foo:
  ID = foo
  provider = provider.aws

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

	state := MustShimLegacyState(&State{
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
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: state,
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyOutputOrphanStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Apply_outputOrphanModule(t *testing.T) {
	m := testModule(t, "apply-output-orphan-module")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	state := MustShimLegacyState(&State{
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
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: state.DeepCopy(),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyOutputOrphanModuleStr)
	if actual != expected {
		t.Fatalf("expected:\n%s\n\ngot:\n%s", expected, actual)
	}

	// now apply with no module in the config, which should remove the
	// remaining output
	ctx = testContext2(t, &ContextOpts{
		Config: configs.NewEmptyConfig(),
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: state.DeepCopy(),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags = ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	if !state.Empty() {
		t.Fatalf("wrong final state %s\nwant empty state", spew.Sdump(state))
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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
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

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	if _, diags := ctx.Apply(); diags.HasErrors() {
		t.Fatalf("apply errors: %s", diags.Err())
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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	if _, diags := ctx.Apply(); diags.HasErrors() {
		t.Fatalf("apply errors: %s", diags.Err())
	}

	if !called {
		t.Fatal("configure never called")
	}
}

func TestContext2Apply_provisionerModule(t *testing.T) {
	m := testModule(t, "apply-provisioner-module")

	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	pr := testProvisioner()
	pr.GetSchemaResponse = provisioners.GetSchemaResponse{
		Provisioner: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"foo": {Type: cty.String, Optional: true},
			},
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyProvisionerModuleStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}

	// Verify apply was invoked
	if !pr.ProvisionResourceCalled {
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
		val, ok := c.Config["command"]
		if !ok || val != "computed_value" {
			t.Fatalf("bad value for foo: %v %#v", val, c)
		}

		return nil
	}
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
		Variables: InputValues{
			"value": &InputValue{
				Value:      cty.NumberIntVal(1),
				SourceType: ValueFromCaller,
			},
		},
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyProvisionerStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}

	// Verify apply was invoked
	if !pr.ProvisionResourceCalled {
		t.Fatalf("provisioner not invoked")
	}
}

func TestContext2Apply_provisionerCreateFail(t *testing.T) {
	m := testModule(t, "apply-provisioner-fail-create")
	p := testProvider("aws")
	pr := testProvisioner()
	p.DiffFn = testDiffFn

	p.ApplyFn = func(info *InstanceInfo, is *InstanceState, id *InstanceDiff) (*InstanceState, error) {
		is.ID = "foo"
		return is, fmt.Errorf("error")
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags == nil {
		t.Fatal("should error")
	}

	got := strings.TrimSpace(state.String())
	want := strings.TrimSpace(testTerraformApplyProvisionerFailCreateStr)
	if got != want {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", got, want)
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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags == nil {
		t.Fatal("should error")
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyProvisionerFailCreateNoIdStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags == nil {
		t.Fatal("should error")
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyProvisionerFailStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
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

	state := MustShimLegacyState(&State{
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
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
		State: state,
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags == nil {
		t.Fatal("should error")
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyProvisionerFailCreateBeforeDestroyStr)
	if actual != expected {
		t.Fatalf("expected:\n%s\n:got\n%s", expected, actual)
	}
}

func TestContext2Apply_error_createBeforeDestroy(t *testing.T) {
	m := testModule(t, "apply-error-create-before")
	p := testProvider("aws")
	state := MustShimLegacyState(&State{
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
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: state,
	})
	p.ApplyFn = func(info *InstanceInfo, is *InstanceState, id *InstanceDiff) (*InstanceState, error) {
		return nil, fmt.Errorf("error")
	}
	p.DiffFn = testDiffFn

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags == nil {
		t.Fatal("should have error")
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyErrorCreateBeforeDestroyStr)
	if actual != expected {
		t.Fatalf("wrong final state\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Apply_errorDestroy_createBeforeDestroy(t *testing.T) {
	m := testModule(t, "apply-error-create-before")
	p := testProvider("aws")
	state := MustShimLegacyState(&State{
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
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
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
			Attributes: map[string]string{
				"type":        "aws_instance",
				"require_new": "xyz",
			},
		}
		return is, nil
	}
	p.DiffFn = testDiffFn

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags == nil {
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
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"require_new": {Type: cty.String, Optional: true},
					"id":          {Type: cty.String, Computed: true},
				},
			},
		},
	}
	ps := map[string]providers.Factory{"aws": testProviderFuncFixed(p)}
	state := MustShimLegacyState(&State{
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
	})

	p.DiffFn = func(info *InstanceInfo, s *InstanceState, rc *ResourceConfig) (*InstanceDiff, error) {
		if rc == nil {
			return &InstanceDiff{
				Destroy: true,
			}, nil
		}

		rn, _ := rc.Get("require_new")
		return &InstanceDiff{
			Attributes: map[string]*ResourceAttrDiff{
				"id": {
					New:         hcl2shim.UnknownVariableValue,
					NewComputed: true,
					RequiresNew: true,
				},
				"require_new": {
					Old:         s.Attributes["require_new"],
					New:         rn.(string),
					RequiresNew: true,
				},
			},
		}, nil
	}

	ctx := testContext2(t, &ContextOpts{
		Config:           m,
		ProviderResolver: providers.ResolverFixed(ps),
		State:            state,
	})
	createdInstanceId := "bar"
	// Create works
	createFunc := func(is *InstanceState, id *InstanceDiff) (*InstanceState, error) {
		return &InstanceState{
			ID: createdInstanceId,
			Attributes: map[string]string{
				"require_new": id.Attributes["require_new"].New,
			},
		}, nil
	}
	// Destroy starts broken
	destroyFunc := func(is *InstanceState) (*InstanceState, error) {
		return is, fmt.Errorf("destroy failed")
	}
	p.ApplyFn = func(info *InstanceInfo, is *InstanceState, id *InstanceDiff) (*InstanceState, error) {
		if id.Destroy {
			return destroyFunc(is)
		} else {
			return createFunc(is, id)
		}
	}

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	// Destroy is broken, so even though CBD successfully replaces the instance,
	// we'll have to save the Deposed instance to destroy later
	state, diags := ctx.Apply()
	if diags == nil {
		t.Fatal("should have error")
	}

	checkStateString(t, state, `
aws_instance.web: (1 deposed)
  ID = bar
  provider = provider.aws
  require_new = yes
  Deposed ID 1 = foo
	`)

	createdInstanceId = "baz"
	ctx = testContext2(t, &ContextOpts{
		Config:           m,
		ProviderResolver: providers.ResolverFixed(ps),
		State:            state,
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	// We're replacing the primary instance once again. Destroy is _still_
	// broken, so the Deposed list gets longer
	state, diags = ctx.Apply()
	if diags == nil {
		t.Fatal("should have error")
	}

	// For this one we can't rely on checkStateString because its result is
	// not deterministic when multiple deposed objects are present. Instead,
	// we will probe the state object directly.
	{
		is := state.RootModule().Resources["aws_instance.web"].Instances[addrs.NoKey]
		t.Logf("aws_instance.web is %s", spew.Sdump(is))
		if is.Current == nil {
			t.Fatalf("no current object for aws_instance web; should have one")
		}
		if !bytes.Contains(is.Current.AttrsJSON, []byte("baz")) {
			t.Fatalf("incorrect current object attrs %s; want id=baz", is.Current.AttrsJSON)
		}
		if got, want := len(is.Deposed), 2; got != want {
			t.Fatalf("wrong number of deposed instances %d; want %d", got, want)
		}
		var foos, bars int
		for _, obj := range is.Deposed {
			if bytes.Contains(obj.AttrsJSON, []byte("foo")) {
				foos++
			}
			if bytes.Contains(obj.AttrsJSON, []byte("bar")) {
				bars++
			}
		}
		if got, want := foos, 1; got != want {
			t.Fatalf("wrong number of deposed instances with id=foo %d; want %d", got, want)
		}
		if got, want := bars, 1; got != want {
			t.Fatalf("wrong number of deposed instances with id=bar %d; want %d", got, want)
		}
	}

	// Destroy partially fixed!
	destroyFunc = func(is *InstanceState) (*InstanceState, error) {
		if is.ID == "foo" || is.ID == "baz" {
			return nil, nil
		} else {
			return is, fmt.Errorf("destroy partially failed")
		}
	}

	createdInstanceId = "qux"
	ctx = testContext2(t, &ContextOpts{
		Config:           m,
		ProviderResolver: providers.ResolverFixed(ps),
		State:            state,
	})
	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}
	state, diags = ctx.Apply()
	// Expect error because 1/2 of Deposed destroys failed
	if diags == nil {
		t.Fatal("should have error")
	}

	// foo and baz are now gone, bar sticks around
	checkStateString(t, state, `
aws_instance.web: (1 deposed)
  ID = qux
  provider = provider.aws
  require_new = yes
  Deposed ID 1 = bar
	`)

	// Destroy working fully!
	destroyFunc = func(is *InstanceState) (*InstanceState, error) {
		return nil, nil
	}

	createdInstanceId = "quux"
	ctx = testContext2(t, &ContextOpts{
		Config:           m,
		ProviderResolver: providers.ResolverFixed(ps),
		State:            state,
	})
	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}
	state, diags = ctx.Apply()
	if diags.HasErrors() {
		t.Fatal("should not have error:", diags.Err())
	}

	// And finally the state is clean
	checkStateString(t, state, `
aws_instance.web:
  ID = quux
  provider = provider.aws
  require_new = yes
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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	checkStateString(t, state, `
aws_instance.foo:
  ID = foo
  provider = provider.aws
  foo = bar
  type = aws_instance
  `)

	// Verify apply was invoked
	if !pr.ProvisionResourceCalled {
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
		Config: m,
		Hooks:  []Hook{h},
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	if _, diags := ctx.Apply(); diags.HasErrors() {
		t.Fatalf("apply errors: %s", diags.Err())
	}

	if !h.PostProvisionInstanceStepCalled {
		t.Fatal("PostProvisionInstanceStep not called")
	}
	if h.PostProvisionInstanceStepErrorArg == nil {
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
		val, ok := c.Config["command"]
		if !ok || val != "destroy" {
			t.Fatalf("bad value for foo: %v %#v", val, c)
		}

		return nil
	}

	state := MustShimLegacyState(&State{
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
		Config:  m,
		State:   state,
		Destroy: true,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	checkStateString(t, state, `<no state>`)

	// Verify apply was invoked
	if !pr.ProvisionResourceCalled {
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

	state := MustShimLegacyState(&State{
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
		Config:  m,
		State:   state,
		Destroy: true,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags == nil {
		t.Fatal("should error")
	}

	checkStateString(t, state, `
aws_instance.foo:
  ID = bar
  provider = provider.aws
	`)

	// Verify apply was invoked
	if !pr.ProvisionResourceCalled {
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
		val, ok := c.Config["command"]
		if !ok {
			t.Fatalf("bad value for foo: %v %#v", val, c)
		}

		l.Lock()
		defer l.Unlock()
		calls = append(calls, val.(string))
		return fmt.Errorf("provisioner error")
	}

	state := MustShimLegacyState(&State{
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
		Config:  m,
		State:   state,
		Destroy: true,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	checkStateString(t, state, `<no state>`)

	// Verify apply was invoked
	if !pr.ProvisionResourceCalled {
		t.Fatalf("provisioner not invoked")
	}

	expected := []string{"one", "two"}
	if !reflect.DeepEqual(calls, expected) {
		t.Fatalf("wrong commands\ngot:  %#v\nwant: %#v", calls, expected)
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
		val, ok := c.Config["command"]
		if !ok {
			t.Fatalf("bad value for foo: %v %#v", val, c)
		}

		l.Lock()
		defer l.Unlock()
		calls = append(calls, val.(string))
		return fmt.Errorf("provisioner error")
	}

	state := MustShimLegacyState(&State{
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
		Config:  m,
		State:   state,
		Destroy: true,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags == nil {
		t.Fatal("apply succeeded; wanted error from second provisioner")
	}

	checkStateString(t, state, `
aws_instance.foo:
  ID = bar
  provider = provider.aws
  `)

	// Verify apply was invoked
	if !pr.ProvisionResourceCalled {
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

		val, ok := c.Config["command"]
		if !ok || val != expected {
			t.Fatalf("bad value for command: %v %#v", val, c)
		}

		return nil
	}

	state := MustShimLegacyState(&State{
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
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		State:  state,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	checkStateString(t, state, `
aws_instance.foo:
  ID = foo
  provider = provider.aws
  foo = bar
  type = aws_instance
	`)

	// Verify apply was invoked
	if !pr.ProvisionResourceCalled {
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
		val, ok := c.Config["command"]
		if !ok || val != "value" {
			t.Fatalf("bad value for foo: %v %#v", val, c)
		}

		return nil
	}

	state := MustShimLegacyState(&State{
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
	})

	ctx := testContext2(t, &ContextOpts{
		Config:  m,
		State:   state,
		Destroy: true,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	checkStateString(t, state, `<no state>`)

	// Verify apply was invoked
	if !pr.ProvisionResourceCalled {
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
		val, ok := c.Config["command"]
		if !ok || val != "hello" {
			return fmt.Errorf("bad value for command: %v %#v", val, c)
		}

		return nil
	}

	state := MustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.bar": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
							Attributes: map[string]string{
								"value": "hello",
							},
						},
						Provider: "provider.aws",
					},

					"aws_instance.foo": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
						},
						Provider: "provider.aws",
					},
				},
			},
		},
	})

	ctx := testContext2(t, &ContextOpts{
		Config:  m,
		State:   state,
		Destroy: true,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	checkStateString(t, state, `<no state>`)

	// Verify apply was invoked
	if !pr.ProvisionResourceCalled {
		t.Fatalf("provisioner not invoked")
	}
}

// Test that a destroy provisioner referencing an invalid key errors.
func TestContext2Apply_provisionerDestroyRefInvalid(t *testing.T) {
	m := testModule(t, "apply-provisioner-destroy-ref-invalid")
	p := testProvider("aws")
	pr := testProvisioner()
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	pr.ApplyFn = func(rs *InstanceState, c *ResourceConfig) error {
		return nil
	}

	state := MustShimLegacyState(&State{
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
	})

	ctx := testContext2(t, &ContextOpts{
		Config:  m,
		State:   state,
		Destroy: true,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	// this was an apply test, but this is now caught in Validation
	if diags := ctx.Validate(); !diags.HasErrors() {
		t.Fatal("expected error")
	}
}

func TestContext2Apply_provisionerResourceRef(t *testing.T) {
	m := testModule(t, "apply-provisioner-resource-ref")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	pr := testProvisioner()
	pr.ApplyFn = func(rs *InstanceState, c *ResourceConfig) error {
		val, ok := c.Config["command"]
		if !ok || val != "2" {
			t.Fatalf("bad value for foo: %v %#v", val, c)
		}

		return nil
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyProvisionerResourceRefStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}

	// Verify apply was invoked
	if !pr.ProvisionResourceCalled {
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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyProvisionerSelfRefStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}

	// Verify apply was invoked
	if !pr.ProvisionResourceCalled {
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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyProvisionerMultiSelfRefStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}

	// Verify apply was invoked
	if !pr.ProvisionResourceCalled {
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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyProvisionerMultiSelfRefSingleStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}

	// Verify apply was invoked
	if !pr.ProvisionResourceCalled {
		t.Fatalf("provisioner not invoked")
	}

	// Verify our result
	sort.Strings(order)
	expectedOrder := []string{"0", "1", "2"}
	if !reflect.DeepEqual(order, expectedOrder) {
		t.Fatalf("bad: %#v", order)
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

	var state *states.State
	{
		ctx := testContext2(t, &ContextOpts{
			Config: m,
			ProviderResolver: providers.ResolverFixed(
				map[string]providers.Factory{
					"aws": testProviderFuncFixed(p),
				},
			),
			Provisioners: map[string]ProvisionerFactory{
				"shell": testProvisionerFuncFixed(pr),
			},
		})

		_, diags := ctx.Plan()
		if diags.HasErrors() {
			t.Fatalf("diags: %s", diags.Err())
		}

		state, diags = ctx.Apply()
		if diags.HasErrors() {
			t.Fatalf("diags: %s", diags.Err())
		}

		// Verify apply was invoked
		if !pr.ProvisionResourceCalled {
			t.Fatalf("provisioner not invoked")
		}
	}

	{
		ctx := testContext2(t, &ContextOpts{
			Config:  m,
			Destroy: true,
			State:   state,
			ProviderResolver: providers.ResolverFixed(
				map[string]providers.Factory{
					"aws": testProviderFuncFixed(p),
				},
			),
			Provisioners: map[string]ProvisionerFactory{
				"shell": testProvisionerFuncFixed(pr),
			},
		})

		_, diags := ctx.Plan()
		if diags.HasErrors() {
			t.Fatalf("diags: %s", diags.Err())
		}

		state, diags = ctx.Apply()
		if diags.HasErrors() {
			t.Fatalf("diags: %s", diags.Err())
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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		logDiagnostics(t, diags)
		t.Fatal("plan failed")
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		logDiagnostics(t, diags)
		t.Fatal("apply failed")
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyProvisionerDiffStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}

	// Verify apply was invoked
	if !pr.ProvisionResourceCalled {
		t.Fatalf("provisioner was not called on first apply")
	}
	pr.ProvisionResourceCalled = false

	// Change the state to force a diff
	mod := state.RootModule()
	obj := mod.Resources["aws_instance.bar"].Instances[addrs.NoKey].Current
	var attrs map[string]interface{}
	err := json.Unmarshal(obj.AttrsJSON, &attrs)
	if err != nil {
		t.Fatal(err)
	}
	attrs["foo"] = "baz"
	obj.AttrsJSON, err = json.Marshal(attrs)
	if err != nil {
		t.Fatal(err)
	}

	// Re-create context with state
	ctx = testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
		State: state,
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		logDiagnostics(t, diags)
		t.Fatal("plan failed")
	}

	state2, diags := ctx.Apply()
	if diags.HasErrors() {
		logDiagnostics(t, diags)
		t.Fatal("apply failed")
	}

	actual = strings.TrimSpace(state2.String())
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}

	// Verify apply was NOT invoked
	if pr.ProvisionResourceCalled {
		t.Fatalf("provisioner was called on second apply; should not have been")
	}
}

func TestContext2Apply_outputDiffVars(t *testing.T) {
	m := testModule(t, "apply-good")
	p := testProvider("aws")
	s := MustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.baz": &ResourceState{ // This one is not in config, so should be destroyed
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
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
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
	p.DiffFn = func(info *InstanceInfo, s *InstanceState, rc *ResourceConfig) (*InstanceDiff, error) {
		d := &InstanceDiff{
			Attributes: map[string]*ResourceAttrDiff{},
		}
		if new, ok := rc.Get("value"); ok {
			d.Attributes["value"] = &ResourceAttrDiff{
				New: new.(string),
			}
		}
		if new, ok := rc.Get("foo"); ok {
			d.Attributes["foo"] = &ResourceAttrDiff{
				New: new.(string),
			}
		} else if rc.IsComputed("foo") {
			d.Attributes["foo"] = &ResourceAttrDiff{
				NewComputed: true,
				Type:        DiffAttrOutput, // This doesn't actually really do anything anymore, but this test originally set it.
			}
		}
		if new, ok := rc.Get("num"); ok {
			d.Attributes["num"] = &ResourceAttrDiff{
				New: fmt.Sprintf("%#v", new),
			}
		}
		return d, nil
	}

	if _, diags := ctx.Plan(); diags.HasErrors() {
		logDiagnostics(t, diags)
		t.Fatal("plan failed")
	}
	if _, diags := ctx.Apply(); diags.HasErrors() {
		logDiagnostics(t, diags)
		t.Fatal("apply failed")
	}
}

func TestContext2Apply_destroyX(t *testing.T) {
	m := testModule(t, "apply-destroy")
	h := new(HookRecordApplyOrder)
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Hooks:  []Hook{h},
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	// First plan and apply a create operation
	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	// Next, plan and apply a destroy operation
	h.Active = true
	ctx = testContext2(t, &ContextOpts{
		Destroy: true,
		State:   state,
		Config:  m,
		Hooks:   []Hook{h},
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags = ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	// Test that things were destroyed
	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyDestroyStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
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
		Config: m,
		Hooks:  []Hook{h},
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	// First plan and apply a create operation
	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	t.Logf("State 1: %s", state)

	// Next, plan and apply a destroy
	h.Active = true
	ctx = testContext2(t, &ContextOpts{
		Destroy: true,
		State:   state,
		Config:  m,
		Hooks:   []Hook{h},
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags = ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	// Test that things were destroyed
	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyDestroyStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
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
		Config: m,
		Hooks:  []Hook{h},
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	// First plan and apply a create operation
	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	// Verify that we got the apply info correct
	if v := h.PreApplyAddr.String(); v != "module.child.aws_instance.foo" {
		t.Fatalf("bad: %s", v)
	}

	// Next, plan and apply a destroy operation and reset the hook
	h = new(MockHook)
	ctx = testContext2(t, &ContextOpts{
		Destroy: true,
		State:   state,
		Config:  m,
		Hooks:   []Hook{h},
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags = ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	// Test that things were destroyed
	if v := h.PreApplyAddr.String(); v != "module.child.aws_instance.foo" {
		t.Fatalf("bad: %s", v)
	}
}

func TestContext2Apply_destroyNestedModule(t *testing.T) {
	m := testModule(t, "apply-destroy-nested-module")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	s := MustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: []string{"root", "child", "subchild"},
				Resources: map[string]*ResourceState{
					"aws_instance.bar": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
						},
						Provider: "provider.aws",
					},
				},
			},
		},
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	// First plan and apply a create operation
	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	// Test that things were destroyed
	actual := strings.TrimSpace(state.String())
	if actual != "<no state>" {
		t.Fatalf("expected no state, got: %s", actual)
	}
}

func TestContext2Apply_destroyDeeplyNestedModule(t *testing.T) {
	m := testModule(t, "apply-destroy-deeply-nested-module")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	s := MustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: []string{"root", "child", "subchild", "subsubchild"},
				Resources: map[string]*ResourceState{
					"aws_instance.bar": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
						},
						Provider: "provider.aws",
					},
				},
			},
		},
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	// First plan and apply a create operation
	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	// Test that things were destroyed
	if !state.Empty() {
		t.Fatalf("wrong final state %s\nwant empty state", spew.Sdump(state))
	}
}

// https://github.com/hashicorp/terraform/issues/5440
func TestContext2Apply_destroyModuleWithAttrsReferencingResource(t *testing.T) {
	m, snap := testModuleWithSnapshot(t, "apply-destroy-module-with-attrs")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	var state *states.State
	{
		ctx := testContext2(t, &ContextOpts{
			Config: m,
			ProviderResolver: providers.ResolverFixed(
				map[string]providers.Factory{
					"aws": testProviderFuncFixed(p),
				},
			),
		})

		// First plan and apply a create operation
		if p, diags := ctx.Plan(); diags.HasErrors() {
			t.Fatalf("plan diags: %s", diags.Err())
		} else {
			t.Logf("Step 1 plan: %s", legacyDiffComparisonString(p.Changes))
		}

		var diags tfdiags.Diagnostics
		state, diags = ctx.Apply()
		if diags.HasErrors() {
			t.Fatalf("apply errs: %s", diags.Err())
		}

		t.Logf("Step 1 state: %s", state)
	}

	h := new(HookRecordApplyOrder)
	h.Active = true

	{
		ctx := testContext2(t, &ContextOpts{
			Destroy: true,
			Config:  m,
			State:   state,
			Hooks:   []Hook{h},
			ProviderResolver: providers.ResolverFixed(
				map[string]providers.Factory{
					"aws": testProviderFuncFixed(p),
				},
			),
		})

		// First plan and apply a create operation
		plan, diags := ctx.Plan()
		if diags.HasErrors() {
			t.Fatalf("destroy plan err: %s", diags.Err())
		}

		t.Logf("Step 2 plan: %s", legacyDiffComparisonString(plan.Changes))

		ctxOpts, err := contextOptsForPlanViaFile(snap, state, plan)
		if err != nil {
			t.Fatalf("failed to round-trip through planfile: %s", err)
		}

		ctxOpts.ProviderResolver = providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		)
		ctx, diags = NewContext(ctxOpts)
		if diags.HasErrors() {
			t.Fatalf("err: %s", diags.Err())
		}

		state, diags = ctx.Apply()
		if diags.HasErrors() {
			t.Fatalf("destroy apply err: %s", diags.Err())
		}

		t.Logf("Step 2 state: %s", state)
	}

	//Test that things were destroyed
	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(`<no state>`)
	if actual != expected {
		t.Fatalf("expected:\n\n%s\n\nactual:\n\n%s", expected, actual)
	}
}

func TestContext2Apply_destroyWithModuleVariableAndCount(t *testing.T) {
	m, snap := testModuleWithSnapshot(t, "apply-destroy-mod-var-and-count")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	var state *states.State
	var diags tfdiags.Diagnostics
	{
		ctx := testContext2(t, &ContextOpts{
			Config: m,
			ProviderResolver: providers.ResolverFixed(
				map[string]providers.Factory{
					"aws": testProviderFuncFixed(p),
				},
			),
		})

		// First plan and apply a create operation
		if _, diags := ctx.Plan(); diags.HasErrors() {
			t.Fatalf("plan err: %s", diags.Err())
		}

		state, diags = ctx.Apply()
		if diags.HasErrors() {
			t.Fatalf("apply err: %s", diags.Err())
		}
	}

	h := new(HookRecordApplyOrder)
	h.Active = true

	{
		ctx := testContext2(t, &ContextOpts{
			Destroy: true,
			Config:  m,
			State:   state,
			Hooks:   []Hook{h},
			ProviderResolver: providers.ResolverFixed(
				map[string]providers.Factory{
					"aws": testProviderFuncFixed(p),
				},
			),
		})

		// First plan and apply a create operation
		plan, diags := ctx.Plan()
		if diags.HasErrors() {
			t.Fatalf("destroy plan err: %s", diags.Err())
		}

		ctxOpts, err := contextOptsForPlanViaFile(snap, state, plan)
		if err != nil {
			t.Fatalf("failed to round-trip through planfile: %s", err)
		}

		ctxOpts.ProviderResolver = providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		)
		ctx, diags = NewContext(ctxOpts)
		if diags.HasErrors() {
			t.Fatalf("err: %s", diags.Err())
		}

		state, diags = ctx.Apply()
		if diags.HasErrors() {
			t.Fatalf("destroy apply err: %s", diags.Err())
		}
	}

	//Test that things were destroyed
	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(`
<no state>
module.child:
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

	var state *states.State
	var diags tfdiags.Diagnostics
	{
		ctx := testContext2(t, &ContextOpts{
			Config: m,
			ProviderResolver: providers.ResolverFixed(
				map[string]providers.Factory{
					"aws": testProviderFuncFixed(p),
				},
			),
		})

		// First plan and apply a create operation
		if _, diags := ctx.Plan(); diags.HasErrors() {
			t.Fatalf("plan err: %s", diags.Err())
		}

		state, diags = ctx.Apply()
		if diags.HasErrors() {
			t.Fatalf("apply err: %s", diags.Err())
		}
	}

	{
		ctx := testContext2(t, &ContextOpts{
			Destroy: true,
			Config:  m,
			State:   state,
			ProviderResolver: providers.ResolverFixed(
				map[string]providers.Factory{
					"aws": testProviderFuncFixed(p),
				},
			),
			Targets: []addrs.Targetable{
				addrs.RootModuleInstance.Child("child", addrs.NoKey),
			},
		})

		_, diags := ctx.Plan()
		if diags.HasErrors() {
			t.Fatalf("plan err: %s", diags)
		}
		if len(diags) != 1 {
			// Should have one warning that -target is in effect.
			t.Fatalf("got %d diagnostics in plan; want 1", len(diags))
		}
		if got, want := diags[0].Severity(), tfdiags.Warning; got != want {
			t.Errorf("wrong diagnostic severity %#v; want %#v", got, want)
		}
		if got, want := diags[0].Description().Summary, "Resource targeting is in effect"; got != want {
			t.Errorf("wrong diagnostic summary %#v; want %#v", got, want)
		}

		// Destroy, targeting the module explicitly
		state, diags = ctx.Apply()
		if diags.HasErrors() {
			t.Fatalf("destroy apply err: %s", diags)
		}
		if len(diags) != 1 {
			t.Fatalf("got %d diagnostics; want 1", len(diags))
		}
		if got, want := diags[0].Severity(), tfdiags.Warning; got != want {
			t.Errorf("wrong diagnostic severity %#v; want %#v", got, want)
		}
		if got, want := diags[0].Description().Summary, "Applied changes may be incomplete"; got != want {
			t.Errorf("wrong diagnostic summary %#v; want %#v", got, want)
		}
	}

	//Test that things were destroyed
	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(`<no state>`)
	if actual != expected {
		t.Fatalf("expected: \n%s\n\nbad: \n%s", expected, actual)
	}
}

func TestContext2Apply_destroyWithModuleVariableAndCountNested(t *testing.T) {
	m, snap := testModuleWithSnapshot(t, "apply-destroy-mod-var-and-count-nested")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	var state *states.State
	var diags tfdiags.Diagnostics
	{
		ctx := testContext2(t, &ContextOpts{
			Config: m,
			ProviderResolver: providers.ResolverFixed(
				map[string]providers.Factory{
					"aws": testProviderFuncFixed(p),
				},
			),
		})

		// First plan and apply a create operation
		if _, diags := ctx.Plan(); diags.HasErrors() {
			t.Fatalf("plan err: %s", diags.Err())
		}

		state, diags = ctx.Apply()
		if diags.HasErrors() {
			t.Fatalf("apply err: %s", diags.Err())
		}
	}

	h := new(HookRecordApplyOrder)
	h.Active = true

	{
		ctx := testContext2(t, &ContextOpts{
			Destroy: true,
			Config:  m,
			State:   state,
			Hooks:   []Hook{h},
			ProviderResolver: providers.ResolverFixed(
				map[string]providers.Factory{
					"aws": testProviderFuncFixed(p),
				},
			),
		})

		// First plan and apply a create operation
		plan, diags := ctx.Plan()
		if diags.HasErrors() {
			t.Fatalf("destroy plan err: %s", diags.Err())
		}

		ctxOpts, err := contextOptsForPlanViaFile(snap, state, plan)
		if err != nil {
			t.Fatalf("failed to round-trip through planfile: %s", err)
		}

		ctxOpts.ProviderResolver = providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		)
		ctx, diags = NewContext(ctxOpts)
		if diags.HasErrors() {
			t.Fatalf("err: %s", diags.Err())
		}

		state, diags = ctx.Apply()
		if diags.HasErrors() {
			t.Fatalf("destroy apply err: %s", diags.Err())
		}
	}

	//Test that things were destroyed
	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(`
<no state>
module.child.child2:
		`)
	if actual != expected {
		t.Fatalf("expected: \n%s\n\nbad: \n%s", expected, actual)
	}
}

func TestContext2Apply_destroyOutputs(t *testing.T) {
	m := testModule(t, "apply-destroy-outputs")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	// First plan and apply a create operation
	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()

	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	// Next, plan and apply a destroy operation
	ctx = testContext2(t, &ContextOpts{
		Destroy: true,
		State:   state,
		Config:  m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags = ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	mod := state.RootModule()
	if len(mod.Resources) > 0 {
		t.Fatalf("expected no resources, got: %#v", mod)
	}

	// destroying again should produce no errors
	ctx = testContext2(t, &ContextOpts{
		Destroy: true,
		State:   state,
		Config:  m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})
	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	if _, diags := ctx.Apply(); diags.HasErrors() {
		t.Fatal(diags.Err())
	}
}

func TestContext2Apply_destroyOrphan(t *testing.T) {
	m := testModule(t, "apply-error")
	p := testProvider("aws")
	s := MustShimLegacyState(&State{
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
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
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
	p.DiffFn = func(info *InstanceInfo, s *InstanceState, rc *ResourceConfig) (*InstanceDiff, error) {
		d := &InstanceDiff{
			Attributes: map[string]*ResourceAttrDiff{},
		}
		if new, ok := rc.Get("value"); ok {
			d.Attributes["value"] = &ResourceAttrDiff{
				New: new.(string),
			}
		}
		if new, ok := rc.Get("foo"); ok {
			d.Attributes["foo"] = &ResourceAttrDiff{
				New: new.(string),
			}
		}
		return d, nil
	}

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
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

	s := MustShimLegacyState(&State{
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
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
		State:   s,
		Destroy: true,
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	if called {
		t.Fatal("provisioner should not be called")
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace("<no state>")
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Apply_error(t *testing.T) {
	errored := false

	m := testModule(t, "apply-error")
	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
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
				"value": "2",
			},
		}, nil
	}
	p.DiffFn = func(info *InstanceInfo, s *InstanceState, rc *ResourceConfig) (*InstanceDiff, error) {
		d := &InstanceDiff{
			Attributes: map[string]*ResourceAttrDiff{},
		}
		if new, ok := rc.Get("value"); ok {
			d.Attributes["value"] = &ResourceAttrDiff{
				New: new.(string),
			}
		}
		if new, ok := rc.Get("foo"); ok {
			d.Attributes["foo"] = &ResourceAttrDiff{
				New: new.(string),
			}
		}
		return d, nil
	}

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags == nil {
		t.Fatal("should have error")
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyErrorStr)
	if actual != expected {
		t.Fatalf("expected:\n%s\n\ngot:\n%s", expected, actual)
	}
}

func TestContext2Apply_errorDestroy(t *testing.T) {
	m := testModule(t, "empty")
	p := testProvider("test")

	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_thing": {
				Attributes: map[string]*configschema.Attribute{
					"id": {Type: cty.String, Optional: true},
				},
			},
		},
	}
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		// Should actually be called for this test, because Terraform Core
		// constructs the plan for a destroy operation itself.
		return providers.PlanResourceChangeResponse{
			PlannedState: req.ProposedNewState,
		}
	}
	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
		// The apply (in this case, a destroy) always fails, so we can verify
		// that the object stays in the state after a destroy fails even though
		// we aren't returning a new state object here.
		return providers.ApplyResourceChangeResponse{
			Diagnostics: tfdiags.Diagnostics(nil).Append(fmt.Errorf("failed")),
		}
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		State: states.BuildState(func(ss *states.SyncState) {
			ss.SetResourceInstanceCurrent(
				addrs.Resource{
					Mode: addrs.ManagedResourceMode,
					Type: "test_thing",
					Name: "foo",
				}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
				&states.ResourceInstanceObjectSrc{
					Status:    states.ObjectReady,
					AttrsJSON: []byte(`{"id":"baz"}`),
				},
				addrs.ProviderConfig{
					Type: "test",
				}.Absolute(addrs.RootModuleInstance),
			)
		}),
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"test": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags == nil {
		t.Fatal("should have error")
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(`
test_thing.foo:
  ID = baz
  provider = provider.test
`) // test_thing.foo is still here, even though provider returned no new state along with its error
	if actual != expected {
		t.Fatalf("expected:\n%s\n\ngot:\n%s", expected, actual)
	}
}

func TestContext2Apply_errorCreateInvalidNew(t *testing.T) {
	m := testModule(t, "apply-error")

	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"value": {Type: cty.String, Optional: true},
					"foo":   {Type: cty.String, Optional: true},
				},
			},
		},
	}
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		return providers.PlanResourceChangeResponse{
			PlannedState: req.ProposedNewState,
		}
	}
	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
		// We're intentionally returning an inconsistent new state here
		// because we want to test that Terraform ignores the inconsistency
		// when accompanied by another error.
		return providers.ApplyResourceChangeResponse{
			NewState: cty.ObjectVal(map[string]cty.Value{
				"value": cty.StringVal("wrong wrong wrong wrong"),
				"foo":   cty.StringVal("absolutely brimming over with wrongability"),
			}),
			Diagnostics: tfdiags.Diagnostics(nil).Append(fmt.Errorf("forced error")),
		}
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags == nil {
		t.Fatal("should have error")
	}
	if got, want := len(diags), 1; got != want {
		// There should be no additional diagnostics generated by Terraform's own eval logic,
		// because the provider's own error supersedes them.
		t.Errorf("wrong number of diagnostics %d; want %d\n%s", got, want, diags.Err())
	}
	if got, want := diags.Err().Error(), "forced error"; !strings.Contains(got, want) {
		t.Errorf("returned error does not contain %q, but it should\n%s", want, diags.Err())
	}
	if got, want := len(state.RootModule().Resources), 2; got != want {
		t.Errorf("%d resources in state before prune; should have %d\n%s", got, want, spew.Sdump(state))
	}
	state.PruneResourceHusks() // aws_instance.bar with no instances gets left behind when we bail out, but that's okay
	if got, want := len(state.RootModule().Resources), 1; got != want {
		t.Errorf("%d resources in state after prune; should have only one (aws_instance.foo, tainted)\n%s", got, spew.Sdump(state))
	}
}

func TestContext2Apply_errorUpdateNullNew(t *testing.T) {
	m := testModule(t, "apply-error")

	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"value": {Type: cty.String, Optional: true},
					"foo":   {Type: cty.String, Optional: true},
				},
			},
		},
	}
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		return providers.PlanResourceChangeResponse{
			PlannedState: req.ProposedNewState,
		}
	}
	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
		// We're intentionally returning no NewState here because we want to
		// test that Terraform retains the prior state, rather than treating
		// the returned null as "no state" (object deleted).
		return providers.ApplyResourceChangeResponse{
			Diagnostics: tfdiags.Diagnostics(nil).Append(fmt.Errorf("forced error")),
		}
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		State: states.BuildState(func(ss *states.SyncState) {
			ss.SetResourceInstanceCurrent(
				addrs.Resource{
					Mode: addrs.ManagedResourceMode,
					Type: "aws_instance",
					Name: "foo",
				}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
				&states.ResourceInstanceObjectSrc{
					Status:    states.ObjectReady,
					AttrsJSON: []byte(`{"value":"old"}`),
				},
				addrs.ProviderConfig{
					Type: "aws",
				}.Absolute(addrs.RootModuleInstance),
			)
		}),
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags == nil {
		t.Fatal("should have error")
	}
	if got, want := len(diags), 1; got != want {
		// There should be no additional diagnostics generated by Terraform's own eval logic,
		// because the provider's own error supersedes them.
		t.Errorf("wrong number of diagnostics %d; want %d\n%s", got, want, diags.Err())
	}
	if got, want := diags.Err().Error(), "forced error"; !strings.Contains(got, want) {
		t.Errorf("returned error does not contain %q, but it should\n%s", want, diags.Err())
	}
	state.PruneResourceHusks()
	if got, want := len(state.RootModule().Resources), 1; got != want {
		t.Fatalf("%d resources in state; should have only one (aws_instance.foo, unmodified)\n%s", got, spew.Sdump(state))
	}

	is := state.ResourceInstance(addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "aws_instance",
		Name: "foo",
	}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance))
	if is == nil {
		t.Fatalf("aws_instance.foo is not in the state after apply")
	}
	if got, want := is.Current.AttrsJSON, []byte(`"old"`); !bytes.Contains(got, want) {
		t.Fatalf("incorrect attributes for aws_instance.foo\ngot: %s\nwant: JSON containing %s\n\n%s", got, want, spew.Sdump(is))
	}
}

func TestContext2Apply_errorPartial(t *testing.T) {
	errored := false

	m := testModule(t, "apply-error")
	p := testProvider("aws")
	s := MustShimLegacyState(&State{
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
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
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
				"value": "2",
			},
		}, nil
	}
	p.DiffFn = func(info *InstanceInfo, s *InstanceState, rc *ResourceConfig) (*InstanceDiff, error) {
		d := &InstanceDiff{
			Attributes: map[string]*ResourceAttrDiff{},
		}
		if new, ok := rc.Get("value"); ok {
			d.Attributes["value"] = &ResourceAttrDiff{
				New: new.(string),
			}
		}
		if new, ok := rc.Get("foo"); ok {
			d.Attributes["foo"] = &ResourceAttrDiff{
				New: new.(string),
			}
		}
		return d, nil
	}

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags == nil {
		t.Fatal("should have error")
	}

	mod := state.RootModule()
	if len(mod.Resources) != 2 {
		t.Fatalf("bad: %#v", mod.Resources)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyErrorPartialStr)
	if actual != expected {
		t.Fatalf("expected:\n%s\n\ngot:\n%s", expected, actual)
	}
}

func TestContext2Apply_hook(t *testing.T) {
	m := testModule(t, "apply-good")
	h := new(MockHook)
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Hooks:  []Hook{h},
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	if _, diags := ctx.Apply(); diags.HasErrors() {
		t.Fatalf("apply errors: %s", diags.Err())
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

	state := MustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.bar": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
						},
						Provider: "provider.aws",
					},
				},
			},
		},
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		State:  state,
		Hooks:  []Hook{h},
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	if _, diags := ctx.Apply(); diags.HasErrors() {
		t.Fatalf("apply errors: %s", diags.Err())
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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	p.ApplyFn = func(info *InstanceInfo, s *InstanceState, d *InstanceDiff) (*InstanceState, error) {
		result := s.MergeDiff(d)
		result.ID = "foo"
		result.Attributes = map[string]string{
			"id":  "bar",
			"num": "42",
		}

		return result, nil
	}
	p.DiffFn = func(*InstanceInfo, *InstanceState, *ResourceConfig) (*InstanceDiff, error) {
		return &InstanceDiff{
			Attributes: map[string]*ResourceAttrDiff{
				"num": &ResourceAttrDiff{
					New: "42",
				},
			},
		}, nil
	}

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("apply errors: %s", diags.Err())
	}

	mod := state.RootModule()
	rs, ok := mod.Resources["aws_instance.foo"]
	if !ok {
		t.Fatal("not in state")
	}
	var attrs map[string]interface{}
	err := json.Unmarshal(rs.Instances[addrs.NoKey].Current.AttrsJSON, &attrs)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := attrs["id"], "foo"; got != want {
		t.Fatalf("wrong id\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestContext2Apply_outputBasic(t *testing.T) {
	m := testModule(t, "apply-output")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyOutputStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Apply_outputAdd(t *testing.T) {
	m1 := testModule(t, "apply-output-add-before")
	p1 := testProvider("aws")
	p1.ApplyFn = testApplyFn
	p1.DiffFn = testDiffFn
	ctx1 := testContext2(t, &ContextOpts{
		Config: m1,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p1),
			},
		),
	})

	if _, diags := ctx1.Plan(); diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	state1, diags := ctx1.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	m2 := testModule(t, "apply-output-add-after")
	p2 := testProvider("aws")
	p2.ApplyFn = testApplyFn
	p2.DiffFn = testDiffFn
	ctx2 := testContext2(t, &ContextOpts{
		Config: m2,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p2),
			},
		),
		State: state1,
	})

	if _, diags := ctx2.Plan(); diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	state2, diags := ctx2.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	actual := strings.TrimSpace(state2.String())
	expected := strings.TrimSpace(testTerraformApplyOutputAddStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Apply_outputList(t *testing.T) {
	m := testModule(t, "apply-output-list")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyOutputMultiStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Apply_outputMultiIndex(t *testing.T) {
	m := testModule(t, "apply-output-multi-index")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyOutputMultiIndexStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
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
	s := MustShimLegacyState(&State{
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
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	if p, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	} else {
		t.Logf("plan: %s", legacyDiffComparisonString(p.Changes))
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
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
	s := MustShimLegacyState(&State{
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
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	if p, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	} else {
		t.Logf("plan: %s", legacyDiffComparisonString(p.Changes))
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
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
	s := MustShimLegacyState(&State{
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
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	if p, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	} else {
		t.Logf("plan: %s", legacyDiffComparisonString(p.Changes))
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
		},
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	mod := state.RootModule()
	if len(mod.Resources) != 1 {
		t.Fatalf("expected 1 resource, got: %#v", mod.Resources)
	}

	checkStateString(t, state, `
aws_instance.foo:
  ID = foo
  provider = provider.aws
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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
		},
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	checkStateString(t, state, `
aws_instance.foo.0:
  ID = foo
  provider = provider.aws
aws_instance.foo.1:
  ID = foo
  provider = provider.aws
aws_instance.foo.2:
  ID = foo
  provider = provider.aws
	`)
}

func TestContext2Apply_targetedCountIndex(t *testing.T) {
	m := testModule(t, "apply-targeted-count")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.ResourceInstance(
				addrs.ManagedResourceMode, "aws_instance", "foo", addrs.IntKey(1),
			),
		},
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	checkStateString(t, state, `
aws_instance.foo.1:
  ID = foo
  provider = provider.aws
	`)
}

func TestContext2Apply_targetedDestroy(t *testing.T) {
	m := testModule(t, "apply-targeted")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: MustShimLegacyState(&State{
			Modules: []*ModuleState{
				&ModuleState{
					Path: rootModulePath,
					Resources: map[string]*ResourceState{
						"aws_instance.foo": resourceState("aws_instance", "i-bcd345"),
						"aws_instance.bar": resourceState("aws_instance", "i-abc123"),
					},
				},
			},
		}),
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
		},
		Destroy: true,
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	mod := state.RootModule()
	if len(mod.Resources) != 1 {
		t.Fatalf("expected 1 resource, got: %#v", mod.Resources)
	}

	checkStateString(t, state, `
aws_instance.bar:
  ID = i-abc123
  provider = provider.aws
	`)
}

func TestContext2Apply_destroyProvisionerWithLocals(t *testing.T) {
	m := testModule(t, "apply-provisioner-destroy-locals")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	pr := testProvisioner()
	pr.ApplyFn = func(_ *InstanceState, rc *ResourceConfig) error {
		cmd, ok := rc.Get("command")
		if !ok || cmd != "local" {
			return fmt.Errorf("provisioner got %v:%s", ok, cmd)
		}
		return nil
	}
	pr.GetSchemaResponse = provisioners.GetSchemaResponse{
		Provisioner: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"command": {
					Type:     cty.String,
					Required: true,
				},
				"when": {
					Type:     cty.String,
					Optional: true,
				},
			},
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
		State: MustShimLegacyState(&State{
			Modules: []*ModuleState{
				&ModuleState{
					Path: []string{"root"},
					Resources: map[string]*ResourceState{
						"aws_instance.foo": resourceState("aws_instance", "1234"),
					},
				},
			},
		}),
		Destroy: true,
		// the test works without targeting, but this also tests that the local
		// node isn't inadvertently pruned because of the wrong evaluation
		// order.
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
		},
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	if _, diags := ctx.Apply(); diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	if !pr.ProvisionResourceCalled {
		t.Fatal("provisioner not called")
	}
}

// this also tests a local value in the config referencing a resource that
// wasn't in the state during destroy.
func TestContext2Apply_destroyProvisionerWithMultipleLocals(t *testing.T) {
	m := testModule(t, "apply-provisioner-destroy-multiple-locals")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	pr := testProvisioner()
	pr.GetSchemaResponse = provisioners.GetSchemaResponse{
		Provisioner: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"id": {
					Type:     cty.String,
					Required: true,
				},
				"command": {
					Type:     cty.String,
					Required: true,
				},
				"when": {
					Type:     cty.String,
					Optional: true,
				},
			},
		},
	}

	pr.ApplyFn = func(is *InstanceState, rc *ResourceConfig) error {
		cmd, ok := rc.Get("command")
		if !ok {
			return errors.New("no command in provisioner")
		}
		id, ok := rc.Get("id")
		if !ok {
			return errors.New("no id in provisioner")
		}

		switch id {
		case "1234":
			if cmd != "local" {
				return fmt.Errorf("provisioner %q got:%q", is.ID, cmd)
			}
		case "3456":
			if cmd != "1234" {
				return fmt.Errorf("provisioner %q got:%q", is.ID, cmd)
			}
		default:
			t.Fatal("unknown instance")
		}
		return nil
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
		State: MustShimLegacyState(&State{
			Modules: []*ModuleState{
				&ModuleState{
					Path: []string{"root"},
					Resources: map[string]*ResourceState{
						"aws_instance.foo": resourceState("aws_instance", "1234"),
						"aws_instance.bar": resourceState("aws_instance", "3456"),
					},
				},
			},
		}),
		Destroy: true,
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	if _, diags := ctx.Apply(); diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	if !pr.ProvisionResourceCalled {
		t.Fatal("provisioner not called")
	}
}

func TestContext2Apply_destroyProvisionerWithOutput(t *testing.T) {
	m := testModule(t, "apply-provisioner-destroy-outputs")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	pr := testProvisioner()
	pr.ApplyFn = func(is *InstanceState, rc *ResourceConfig) error {
		cmd, ok := rc.Get("command")
		if !ok || cmd != "3" {
			return fmt.Errorf("provisioner for %s got %v:%s", is.ID, ok, cmd)
		}
		return nil
	}
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
		State: MustShimLegacyState(&State{
			Modules: []*ModuleState{
				&ModuleState{
					Path: []string{"root"},
					Resources: map[string]*ResourceState{
						"aws_instance.foo": resourceState("aws_instance", "1"),
					},
					Outputs: map[string]*OutputState{
						"value": {
							Type:  "string",
							Value: "3",
						},
					},
				},
				&ModuleState{
					Path: []string{"root", "mod"},
					Resources: map[string]*ResourceState{
						"aws_instance.baz": resourceState("aws_instance", "3"),
					},
					// state needs to be properly initialized
					Outputs: map[string]*OutputState{},
				},
				&ModuleState{
					Path: []string{"root", "mod2"},
					Resources: map[string]*ResourceState{
						"aws_instance.bar": resourceState("aws_instance", "2"),
					},
				},
			},
		}),
		Destroy: true,

		// targeting the source of the value used by all resources should still
		// destroy them all.
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Child("mod", addrs.NoKey).Resource(
				addrs.ManagedResourceMode, "aws_instance", "baz",
			),
		},
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}
	if !pr.ProvisionResourceCalled {
		t.Fatal("provisioner not called")
	}

	// confirm all outputs were removed too
	for _, mod := range state.Modules {
		if len(mod.OutputValues) > 0 {
			t.Fatalf("output left in module state: %#v\n", mod)
		}
	}
}

func TestContext2Apply_targetedDestroyCountDeps(t *testing.T) {
	m := testModule(t, "apply-destroy-targeted-count")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: MustShimLegacyState(&State{
			Modules: []*ModuleState{
				&ModuleState{
					Path: rootModulePath,
					Resources: map[string]*ResourceState{
						"aws_instance.foo": resourceState("aws_instance", "i-bcd345"),
						"aws_instance.bar": resourceState("aws_instance", "i-abc123"),
					},
				},
			},
		}),
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
		},
		Destroy: true,
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: MustShimLegacyState(&State{
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
		}),
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Child("child", addrs.NoKey).Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
		},
		Destroy: true,
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	checkStateString(t, state, `
aws_instance.bar:
  ID = i-abc123
  provider = provider.aws
aws_instance.foo:
  ID = i-bcd345
  provider = provider.aws

module.child:
  aws_instance.bar:
    ID = i-abc123
    provider = provider.aws
	`)
}

func TestContext2Apply_targetedDestroyCountIndex(t *testing.T) {
	m := testModule(t, "apply-targeted-count")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: MustShimLegacyState(&State{
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
		}),
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.ResourceInstance(
				addrs.ManagedResourceMode, "aws_instance", "foo", addrs.IntKey(2),
			),
			addrs.RootModuleInstance.ResourceInstance(
				addrs.ManagedResourceMode, "aws_instance", "bar", addrs.IntKey(1),
			),
		},
		Destroy: true,
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	checkStateString(t, state, `
aws_instance.bar.0:
  ID = i-abc123
  provider = provider.aws
aws_instance.bar.2:
  ID = i-abc123
  provider = provider.aws
aws_instance.foo.0:
  ID = i-bcd345
  provider = provider.aws
aws_instance.foo.1:
  ID = i-bcd345
  provider = provider.aws
	`)
}

func TestContext2Apply_targetedModule(t *testing.T) {
	m := testModule(t, "apply-targeted-module")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Child("child", addrs.NoKey),
		},
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	mod := state.Module(addrs.RootModuleInstance.Child("child", addrs.NoKey))
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
    provider = provider.aws
    num = 2
    type = aws_instance
  aws_instance.foo:
    ID = foo
    provider = provider.aws
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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
		},
	})

	if p, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	} else {
		t.Logf("Diff: %s", legacyDiffComparisonString(p.Changes))
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	checkStateString(t, state, `
aws_instance.foo:
  ID = foo
  provider = provider.aws
  foo = foo
  type = aws_instance

  Dependencies:
    module.child

module.child:
  aws_instance.mod:
    ID = foo
    provider = provider.aws

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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Child("child2", addrs.NoKey),
		},
		State: MustShimLegacyState(&State{
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
		}),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	// - module.child1's instance_id output is dropped because we don't preserve
	//   non-root module outputs between runs (they can be recalculated from config)
	// - module.child2's instance_id is updated because its dependency is updated
	// - child2_id is updated because if its transitive dependency via module.child2
	checkStateString(t, state, `
<no state>
Outputs:

child2_id = foo

module.child2:
  aws_instance.foo:
    ID = foo
    provider = provider.aws

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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Child("child", addrs.NoKey).Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
		},
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	mod := state.Module(addrs.RootModuleInstance.Child("child", addrs.NoKey))
	if mod == nil || len(mod.Resources) != 1 {
		t.Fatalf("expected 1 resource, got: %#v", mod)
	}

	checkStateString(t, state, `
<no state>
module.child:
  aws_instance.foo:
    ID = foo
    provider = provider.aws
    num = 2
    type = aws_instance
	`)
}

func TestContext2Apply_targetedResourceOrphanModule(t *testing.T) {
	m := testModule(t, "apply-targeted-resource-orphan-module")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	// Create a state with an orphan module
	state := MustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: []string{"root", "child"},
				Resources: map[string]*ResourceState{
					"aws_instance.bar": &ResourceState{
						Type:     "aws_instance",
						Primary:  &InstanceState{},
						Provider: "provider.aws",
					},
				},
			},
		},
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: state,
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
		},
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	if _, diags := ctx.Apply(); diags.HasErrors() {
		t.Fatalf("apply errors: %s", diags.Err())
	}
}

func TestContext2Apply_unknownAttribute(t *testing.T) {
	m := testModule(t, "apply-unknown")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if !diags.HasErrors() {
		t.Error("should error, because attribute 'unknown' is still unknown after apply")
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyUnknownAttrStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Apply_unknownAttributeInterpolate(t *testing.T) {
	m := testModule(t, "apply-unknown-interpolate")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags == nil {
		t.Fatal("should error")
	}
}

func TestContext2Apply_vars(t *testing.T) {
	fixture := contextFixtureApplyVars(t)
	opts := fixture.ContextOpts()
	opts.Variables = InputValues{
		"foo": &InputValue{
			Value:      cty.StringVal("us-east-1"),
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
			Value: cty.MapVal(map[string]cty.Value{
				"us-east-1": cty.StringVal("override"),
			}),
			SourceType: ValueFromCaller,
		},
	}
	ctx := testContext2(t, opts)

	diags := ctx.Validate()
	if len(diags) != 0 {
		t.Fatalf("bad: %s", diags.ErrWithWarnings())
	}

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("err: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("err: %s", diags.Err())
	}

	got := strings.TrimSpace(state.String())
	want := strings.TrimSpace(testTerraformApplyVarsStr)
	if got != want {
		t.Errorf("wrong result\n\ngot:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestContext2Apply_varsEnv(t *testing.T) {
	fixture := contextFixtureApplyVarsEnv(t)
	opts := fixture.ContextOpts()
	opts.Variables = InputValues{
		"string": &InputValue{
			Value:      cty.StringVal("baz"),
			SourceType: ValueFromEnvVar,
		},
		"list": &InputValue{
			Value: cty.ListVal([]cty.Value{
				cty.StringVal("Hello"),
				cty.StringVal("World"),
			}),
			SourceType: ValueFromEnvVar,
		},
		"map": &InputValue{
			Value: cty.MapVal(map[string]cty.Value{
				"Hello": cty.StringVal("World"),
				"Foo":   cty.StringVal("Bar"),
				"Baz":   cty.StringVal("Foo"),
			}),
			SourceType: ValueFromEnvVar,
		},
	}
	ctx := testContext2(t, opts)

	diags := ctx.Validate()
	if len(diags) != 0 {
		t.Fatalf("bad: %s", diags.ErrWithWarnings())
	}

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("err: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("err: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testTerraformApplyVarsEnvStr)
	if actual != expected {
		t.Errorf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Apply_createBefore_depends(t *testing.T) {
	m := testModule(t, "apply-depends-create-before")
	h := new(HookRecordApplyOrder)
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	state := MustShimLegacyState(&State{
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
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Hooks:  []Hook{h},
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: state,
	})

	if p, diags := ctx.Plan(); diags.HasErrors() {
		logDiagnostics(t, diags)
		t.Fatal("plan failed")
	} else {
		t.Logf("plan:\n%s", legacyDiffComparisonString(p.Changes))
	}

	h.Active = true
	state, diags := ctx.Apply()
	if diags.HasErrors() {
		logDiagnostics(t, diags)
		t.Fatal("apply failed")
	}

	mod := state.RootModule()
	if len(mod.Resources) < 2 {
		t.Logf("state after apply:\n%s", state.String())
		t.Fatalf("only %d resources in root module; want at least 2", len(mod.Resources))
	}

	got := strings.TrimSpace(state.String())
	want := strings.TrimSpace(testTerraformApplyDependsCreateBeforeStr)
	if got != want {
		t.Fatalf("wrong final state\ngot:\n%s\n\nwant:\n%s", got, want)
	}

	// Test that things were managed _in the right order_
	order := h.States
	diffs := h.Diffs
	if !order[0].IsNull() || diffs[0].Action == plans.Delete {
		t.Fatalf("should create new instance first: %#v", order)
	}

	if order[1].GetAttr("id").AsString() != "baz" {
		t.Fatalf("update must happen after create: %#v", order)
	}

	if order[2].GetAttr("id").AsString() != "bar" || diffs[2].Action != plans.Delete {
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
	state := MustShimLegacyState(&State{
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
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Hooks:  []Hook{h},
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: state,
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	h.Active = true
	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	if invokeCount != 3 {
		t.Fatalf("bad: %d", invokeCount)
	}
}

// GH-7824
func TestContext2Apply_issue7824(t *testing.T) {
	p := testProvider("template")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"template_file": {
				Attributes: map[string]*configschema.Attribute{
					"template":                {Type: cty.String, Optional: true},
					"__template_requires_new": {Type: cty.Bool, Optional: true},
				},
			},
		},
	}

	m, snap := testModuleWithSnapshot(t, "issue-7824")

	// Apply cleanly step 0
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"template": testProviderFuncFixed(p),
			},
		),
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("err: %s", diags.Err())
	}

	// Write / Read plan to simulate running it through a Plan file
	ctxOpts, err := contextOptsForPlanViaFile(snap, ctx.State(), plan)
	if err != nil {
		t.Fatalf("failed to round-trip through planfile: %s", err)
	}

	ctxOpts.ProviderResolver = providers.ResolverFixed(
		map[string]providers.Factory{
			"template": testProviderFuncFixed(p),
		},
	)
	ctx, diags = NewContext(ctxOpts)
	if diags.HasErrors() {
		t.Fatalf("err: %s", diags.Err())
	}

	_, diags = ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("err: %s", diags.Err())
	}
}

// This deals with the situation where a splat expression is used referring
// to another resource whose count is non-constant.
func TestContext2Apply_issue5254(t *testing.T) {
	// Create a provider. We use "template" here just to match the repro
	// we got from the issue itself.
	p := testProvider("template")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"template_file": {
				Attributes: map[string]*configschema.Attribute{
					"template":                {Type: cty.String, Optional: true},
					"__template_requires_new": {Type: cty.Bool, Optional: true},
					"id":                      {Type: cty.String, Computed: true},
					"type":                    {Type: cty.String, Computed: true},
				},
			},
		},
	}

	// Apply cleanly step 0
	ctx := testContext2(t, &ContextOpts{
		Config: testModule(t, "issue-5254/step-0"),
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"template": testProviderFuncFixed(p),
			},
		),
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("err: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("err: %s", diags.Err())
	}

	m, snap := testModuleWithSnapshot(t, "issue-5254/step-1")

	// Application success. Now make the modification and store a plan
	ctx = testContext2(t, &ContextOpts{
		Config: m,
		State:  state,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"template": testProviderFuncFixed(p),
			},
		),
	})

	plan, diags = ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("err: %s", diags.Err())
	}

	// Write / Read plan to simulate running it through a Plan file
	ctxOpts, err := contextOptsForPlanViaFile(snap, state, plan)
	if err != nil {
		t.Fatalf("failed to round-trip through planfile: %s", err)
	}

	ctxOpts.ProviderResolver = providers.ResolverFixed(
		map[string]providers.Factory{
			"template": testProviderFuncFixed(p),
		},
	)
	ctx, diags = NewContext(ctxOpts)
	if diags.HasErrors() {
		t.Fatalf("err: %s", diags.Err())
	}

	state, diags = ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("err: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(`
template_file.child:
  ID = foo
  provider = provider.template
  __template_requires_new = true
  template = Hi
  type = template_file

  Dependencies:
    template_file.parent
template_file.parent.0:
  ID = foo
  provider = provider.template
  template = Hi
  type = template_file
`)
	if actual != expected {
		t.Fatalf("wrong final state\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestContext2Apply_targetedWithTaintedInState(t *testing.T) {
	p := testProvider("aws")
	p.DiffFn = testDiffFn
	p.ApplyFn = testApplyFn
	m, snap := testModuleWithSnapshot(t, "apply-tainted-targets")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "iambeingadded",
			),
		},
		State: MustShimLegacyState(&State{
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
		}),
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("err: %s", diags.Err())
	}

	// Write / Read plan to simulate running it through a Plan file
	ctxOpts, err := contextOptsForPlanViaFile(snap, ctx.State(), plan)
	if err != nil {
		t.Fatalf("failed to round-trip through planfile: %s", err)
	}

	ctxOpts.ProviderResolver = providers.ResolverFixed(
		map[string]providers.Factory{
			"aws": testProviderFuncFixed(p),
		},
	)
	ctx, diags = NewContext(ctxOpts)
	if diags.HasErrors() {
		t.Fatalf("err: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("err: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(`
aws_instance.iambeingadded:
  ID = foo
  provider = provider.aws
aws_instance.ifailedprovisioners: (tainted)
  ID = ifailedprovisioners
  provider = provider.aws
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

	instanceSchema := p.GetSchemaReturn.ResourceTypes["aws_instance"]
	instanceSchema.Attributes["required_field"] = &configschema.Attribute{
		Type:     cty.String,
		Required: true,
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if p, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	} else {
		t.Logf(legacyDiffComparisonString(p.Changes))
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
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
  provider = provider.aws
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
	s := MustShimLegacyState(&State{
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
						Type: "aws_eip",
						Primary: &InstanceState{
							ID: "eip-abc123",
							Attributes: map[string]string{
								"id":       "eip-abc123",
								"instance": "i-abc123",
							},
						},
					},
					"aws_eip.foo.1": &ResourceState{
						Type: "aws_eip",
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
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: s,
	})

	_, diags := ctx.Plan()
	assertNoErrors(t, diags)

	state, diags := ctx.Apply()
	assertNoErrors(t, diags)

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(s.String())
	if actual != expected {
		t.Fatalf("expected:\n%s\n\ngot:\n%s", expected, actual)
	}
}

func TestContext2Apply_ignoreChangesWildcard(t *testing.T) {
	m := testModule(t, "apply-ignore-changes-wildcard")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	instanceSchema := p.GetSchemaReturn.ResourceTypes["aws_instance"]
	instanceSchema.Attributes["required_field"] = &configschema.Attribute{
		Type:     cty.String,
		Required: true,
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if p, diags := ctx.Plan(); diags.HasErrors() {
		logDiagnostics(t, diags)
		t.Fatal("plan failed")
	} else {
		t.Logf(legacyDiffComparisonString(p.Changes))
	}

	state, diags := ctx.Apply()
	assertNoErrors(t, diags)

	mod := state.RootModule()
	if len(mod.Resources) != 1 {
		t.Fatalf("bad: %s", state)
	}

	actual := strings.TrimSpace(state.String())
	// Expect no changes from original state
	expected := strings.TrimSpace(`
aws_instance.foo:
  ID = foo
  provider = provider.aws
  required_field = set
  type = aws_instance
`)
	if actual != expected {
		t.Fatalf("expected:\n%s\ngot:\n%s", expected, actual)
	}
}

// https://github.com/hashicorp/terraform/issues/7378
func TestContext2Apply_destroyNestedModuleWithAttrsReferencingResource(t *testing.T) {
	m, snap := testModuleWithSnapshot(t, "apply-destroy-nested-module-with-attrs")
	p := testProvider("null")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	var state *states.State
	var diags tfdiags.Diagnostics
	{
		ctx := testContext2(t, &ContextOpts{
			Config: m,
			ProviderResolver: providers.ResolverFixed(
				map[string]providers.Factory{
					"null": testProviderFuncFixed(p),
				},
			),
		})

		// First plan and apply a create operation
		if _, diags := ctx.Plan(); diags.HasErrors() {
			t.Fatalf("plan err: %s", diags.Err())
		}

		state, diags = ctx.Apply()
		if diags.HasErrors() {
			t.Fatalf("apply err: %s", diags.Err())
		}
	}

	{
		ctx := testContext2(t, &ContextOpts{
			Destroy: true,
			Config:  m,
			State:   state,
			ProviderResolver: providers.ResolverFixed(
				map[string]providers.Factory{
					"null": testProviderFuncFixed(p),
				},
			),
		})

		plan, diags := ctx.Plan()
		if diags.HasErrors() {
			t.Fatalf("destroy plan err: %s", diags.Err())
		}

		ctxOpts, err := contextOptsForPlanViaFile(snap, state, plan)
		if err != nil {
			t.Fatalf("failed to round-trip through planfile: %s", err)
		}

		ctxOpts.ProviderResolver = providers.ResolverFixed(
			map[string]providers.Factory{
				"null": testProviderFuncFixed(p),
			},
		)
		ctx, diags = NewContext(ctxOpts)
		if diags.HasErrors() {
			t.Fatalf("err: %s", diags.Err())
		}

		state, diags = ctx.Apply()
		if diags.HasErrors() {
			t.Fatalf("destroy apply err: %s", diags.Err())
		}
	}

	if !state.Empty() {
		t.Fatalf("state after apply: %s\nwant empty state", spew.Sdump(state))
	}
}

// If a data source explicitly depends on another resource, it's because we need
// that resource to be applied first.
func TestContext2Apply_dataDependsOn(t *testing.T) {
	p := testProvider("null")
	m := testModule(t, "apply-data-depends-on")

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
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
	p.ReadDataSourceFn = func(req providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
		return providers.ReadDataSourceResponse{
			State: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("boop"),
				"foo": cty.StringVal(provisionerOutput),
			}),
		}
	}

	_, diags := ctx.Refresh()
	assertNoErrors(t, diags)

	_, diags = ctx.Plan()
	assertNoErrors(t, diags)

	state, diags := ctx.Apply()
	assertNoErrors(t, diags)

	root := state.Module(addrs.RootModuleInstance)
	is := root.ResourceInstance(addrs.Resource{
		Mode: addrs.DataResourceMode,
		Type: "null_data_source",
		Name: "read",
	}.Instance(addrs.NoKey))
	if is == nil {
		t.Fatal("data resource instance is not present in state; should be")
	}
	var attrs map[string]interface{}
	err := json.Unmarshal(is.Current.AttrsJSON, &attrs)
	if err != nil {
		t.Fatal(err)
	}
	actual := attrs["foo"]
	expected := "APPLIED"
	if actual != expected {
		t.Fatalf("bad:\n%s", strings.TrimSpace(state.String()))
	}
}

func TestContext2Apply_terraformWorkspace(t *testing.T) {
	m := testModule(t, "apply-terraform-workspace")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	ctx := testContext2(t, &ContextOpts{
		Meta:   &ContextMeta{Env: "foo"},
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	actual := state.RootModule().OutputValues["output"]
	expected := cty.StringVal("foo")
	if actual == nil || actual.Value != expected {
		t.Fatalf("wrong value\ngot:  %#v\nwant: %#v", actual.Value, expected)
	}
}

// verify that multiple config references only create a single depends_on entry
func TestContext2Apply_multiRef(t *testing.T) {
	m := testModule(t, "apply-multi-ref")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("err: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("err: %s", diags.Err())
	}

	deps := state.Modules[""].Resources["aws_instance.other"].Instances[addrs.NoKey].Current.Dependencies
	if len(deps) != 1 || deps[0].String() != "aws_instance.create" {
		t.Fatalf("expected 1 depends_on entry for aws_instance.create, got %q", deps)
	}
}

func TestContext2Apply_targetedModuleRecursive(t *testing.T) {
	m := testModule(t, "apply-targeted-module-recursive")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Child("child", addrs.NoKey),
		},
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("err: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("err: %s", diags.Err())
	}

	mod := state.Module(
		addrs.RootModuleInstance.Child("child", addrs.NoKey).Child("subchild", addrs.NoKey),
	)
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
    provider = provider.aws
    num = 2
    type = aws_instance
	`)
}

func TestContext2Apply_localVal(t *testing.T) {
	m := testModule(t, "apply-local-val")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("error during plan: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("error during apply: %s", diags.Err())
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

func TestContext2Apply_destroyWithLocals(t *testing.T) {
	m := testModule(t, "apply-destroy-with-locals")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = func(info *InstanceInfo, s *InstanceState, c *ResourceConfig) (*InstanceDiff, error) {
		d, err := testDiffFn(info, s, c)
		return d, err
	}
	s := MustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Outputs: map[string]*OutputState{
					"name": &OutputState{
						Type:  "string",
						Value: "test-bar",
					},
				},
				Resources: map[string]*ResourceState{
					"aws_instance.foo": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
							// FIXME: id should only exist in one place
							Attributes: map[string]string{
								"id": "foo",
							},
						},
						Provider: "provider.aws",
					},
				},
			},
		},
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State:   s,
		Destroy: true,
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("err: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("error during apply: %s", diags.Err())
	}

	got := strings.TrimSpace(state.String())
	want := strings.TrimSpace(`<no state>`)
	if got != want {
		t.Fatalf("wrong final state\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestContext2Apply_providerWithLocals(t *testing.T) {
	m := testModule(t, "provider-with-locals")
	p := testProvider("aws")

	providerRegion := ""
	// this should not be overridden during destroy
	p.ConfigureFn = func(c *ResourceConfig) error {
		if r, ok := c.Get("region"); ok {
			providerRegion = r.(string)
		}
		return nil
	}
	p.DiffFn = testDiffFn
	p.ApplyFn = testApplyFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("err: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("err: %s", diags.Err())
	}

	ctx = testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State:   state,
		Destroy: true,
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("err: %s", diags.Err())
	}

	state, diags = ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("err: %s", diags.Err())
	}

	if state.HasResources() {
		t.Fatal("expected no state, got:", state)
	}

	if providerRegion != "bar" {
		t.Fatalf("expected region %q, got: %q", "bar", providerRegion)
	}
}

func TestContext2Apply_destroyWithProviders(t *testing.T) {
	m := testModule(t, "destroy-module-with-provider")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	s := MustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
			},
			&ModuleState{
				Path: []string{"root", "child"},
			},
			&ModuleState{
				Path: []string{"root", "mod", "removed"},
				Resources: map[string]*ResourceState{
					"aws_instance.child": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
						},
						// this provider doesn't exist
						Provider: "provider.aws.baz",
					},
				},
			},
		},
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State:   s,
		Destroy: true,
	})

	// test that we can't destroy if the provider is missing
	if _, diags := ctx.Plan(); diags == nil {
		t.Fatal("expected plan error, provider.aws.baz doesn't exist")
	}

	// correct the state
	s.Modules["module.mod.module.removed"].Resources["aws_instance.child"].ProviderConfig = addrs.ProviderConfig{
		Type:  "aws",
		Alias: "bar",
	}.Absolute(addrs.RootModuleInstance)

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatal(diags.Err())
	}
	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("error during apply: %s", diags.Err())
	}

	got := strings.TrimSpace(state.String())

	want := strings.TrimSpace("<no state>")
	if got != want {
		t.Fatalf("wrong final state\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestContext2Apply_providersFromState(t *testing.T) {
	m := configs.NewEmptyConfig()
	p := testProvider("aws")
	p.DiffFn = testDiffFn

	for _, tc := range []struct {
		name   string
		state  *states.State
		output string
		err    bool
	}{
		{
			name: "add implicit provider",
			state: MustShimLegacyState(&State{
				Modules: []*ModuleState{
					&ModuleState{
						Path: []string{"root"},
						Resources: map[string]*ResourceState{
							"aws_instance.a": &ResourceState{
								Type: "aws_instance",
								Primary: &InstanceState{
									ID: "bar",
								},
								Provider: "provider.aws",
							},
						},
					},
				},
			}),
			err:    false,
			output: "<no state>",
		},

		// an aliased provider must be in the config to remove a resource
		{
			name: "add aliased provider",
			state: MustShimLegacyState(&State{
				Modules: []*ModuleState{
					&ModuleState{
						Path: []string{"root"},
						Resources: map[string]*ResourceState{
							"aws_instance.a": &ResourceState{
								Type: "aws_instance",
								Primary: &InstanceState{
									ID: "bar",
								},
								Provider: "provider.aws.bar",
							},
						},
					},
				},
			}),
			err: true,
		},

		// a provider in a module implies some sort of config, so this isn't
		// allowed even without an alias
		{
			name: "add unaliased module provider",
			state: MustShimLegacyState(&State{
				Modules: []*ModuleState{
					&ModuleState{
						Path: []string{"root", "child"},
						Resources: map[string]*ResourceState{
							"aws_instance.a": &ResourceState{
								Type: "aws_instance",
								Primary: &InstanceState{
									ID: "bar",
								},
								Provider: "module.child.provider.aws",
							},
						},
					},
				},
			}),
			err: true,
		},
	} {

		t.Run(tc.name, func(t *testing.T) {
			ctx := testContext2(t, &ContextOpts{
				Config: m,
				ProviderResolver: providers.ResolverFixed(
					map[string]providers.Factory{
						"aws": testProviderFuncFixed(p),
					},
				),
				State: tc.state,
			})

			_, diags := ctx.Plan()
			if tc.err {
				if diags == nil {
					t.Fatal("expected error")
				} else {
					return
				}
			}
			if !tc.err && diags.HasErrors() {
				t.Fatal(diags.Err())
			}

			state, diags := ctx.Apply()
			if diags.HasErrors() {
				t.Fatalf("diags: %s", diags.Err())
			}

			checkStateString(t, state, "<no state>")

		})
	}
}

func TestContext2Apply_plannedInterpolatedCount(t *testing.T) {
	m, snap := testModuleWithSnapshot(t, "apply-interpolated-count")

	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	providerResolver := providers.ResolverFixed(
		map[string]providers.Factory{
			"aws": testProviderFuncFixed(p),
		},
	)

	s := MustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.test": {
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
						Provider: "provider.aws",
					},
				},
			},
		},
	})

	ctx := testContext2(t, &ContextOpts{
		Config:           m,
		ProviderResolver: providerResolver,
		State:            s,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("plan failed: %s", diags.Err())
	}

	// We'll marshal and unmarshal the plan here, to ensure that we have
	// a clean new context as would be created if we separately ran
	// terraform plan -out=tfplan && terraform apply tfplan
	ctxOpts, err := contextOptsForPlanViaFile(snap, ctx.State(), plan)
	if err != nil {
		t.Fatalf("failed to round-trip through planfile: %s", err)
	}

	ctxOpts.ProviderResolver = providerResolver
	ctx, diags = NewContext(ctxOpts)
	if diags.HasErrors() {
		t.Fatalf("err: %s", diags.Err())
	}

	// Applying the plan should now succeed
	_, diags = ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("apply failed: %s", diags.Err())
	}
}

func TestContext2Apply_plannedDestroyInterpolatedCount(t *testing.T) {
	m, snap := testModuleWithSnapshot(t, "plan-destroy-interpolated-count")

	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	providerResolver := providers.ResolverFixed(
		map[string]providers.Factory{
			"aws": testProviderFuncFixed(p),
		},
	)

	s := MustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.a.0": {
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
						Provider: "provider.aws",
					},
					"aws_instance.a.1": {
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
						Provider: "provider.aws",
					},
				},
				Outputs: map[string]*OutputState{
					"out": {
						Type:  "list",
						Value: []string{"foo", "foo"},
					},
				},
			},
		},
	})

	ctx := testContext2(t, &ContextOpts{
		Config:           m,
		ProviderResolver: providerResolver,
		State:            s,
		Destroy:          true,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("plan failed: %s", diags.Err())
	}

	// We'll marshal and unmarshal the plan here, to ensure that we have
	// a clean new context as would be created if we separately ran
	// terraform plan -out=tfplan && terraform apply tfplan
	ctxOpts, err := contextOptsForPlanViaFile(snap, ctx.State(), plan)
	if err != nil {
		t.Fatalf("failed to round-trip through planfile: %s", err)
	}

	ctxOpts.ProviderResolver = providerResolver
	ctxOpts.Destroy = true
	ctx, diags = NewContext(ctxOpts)
	if diags.HasErrors() {
		t.Fatalf("err: %s", diags.Err())
	}

	// Applying the plan should now succeed
	_, diags = ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("apply failed: %s", diags.Err())
	}
}

func TestContext2Apply_scaleInMultivarRef(t *testing.T) {
	m := testModule(t, "apply-resource-scale-in")

	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	providerResolver := providers.ResolverFixed(
		map[string]providers.Factory{
			"aws": testProviderFuncFixed(p),
		},
	)

	s := MustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.one": {
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
						Provider: "provider.aws",
					},
					"aws_instance.two": {
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
							Attributes: map[string]string{
								"value": "foo",
							},
						},
						Provider: "provider.aws",
					},
				},
			},
		},
	})

	ctx := testContext2(t, &ContextOpts{
		Config:           m,
		ProviderResolver: providerResolver,
		State:            s,
		Variables: InputValues{
			"instance_count": {
				Value:      cty.NumberIntVal(0),
				SourceType: ValueFromCaller,
			},
		},
	})

	_, diags := ctx.Plan()
	assertNoErrors(t, diags)

	// Applying the plan should now succeed
	_, diags = ctx.Apply()
	assertNoErrors(t, diags)
}

func TestContext2Apply_inconsistentWithPlan(t *testing.T) {
	m := testModule(t, "apply-inconsistent-with-plan")
	p := testProvider("test")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test": {
				Attributes: map[string]*configschema.Attribute{
					"id": {Type: cty.String, Computed: true},
				},
			},
		},
	}
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		return providers.PlanResourceChangeResponse{
			PlannedState: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("before"),
			}),
		}
	}
	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
		return providers.ApplyResourceChangeResponse{
			NewState: cty.ObjectVal(map[string]cty.Value{
				// This is intentionally incorrect: because id was fixed at "before"
				// during plan, it must not change during apply.
				"id": cty.StringVal("after"),
			}),
		}
	}
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"test": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	_, diags := ctx.Apply()
	if !diags.HasErrors() {
		t.Fatalf("apply succeeded; want error")
	}
	if got, want := diags.Err().Error(), "Provider produced inconsistent result after apply"; !strings.Contains(got, want) {
		t.Fatalf("wrong error\ngot: %s\nshould contain: %s", got, want)
	}
}

// Issue 19908 was about retaining an existing object in the state when an
// update to it fails and the provider does not return a partially-updated
// value for it. Previously we were incorrectly removing it from the state
// in that case, but instead it should be retained so the update can be
// retried.
func TestContext2Apply_issue19908(t *testing.T) {
	m := testModule(t, "apply-issue19908")
	p := testProvider("test")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test": {
				Attributes: map[string]*configschema.Attribute{
					"baz": {Type: cty.String, Required: true},
				},
			},
		},
	}
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		return providers.PlanResourceChangeResponse{
			PlannedState: req.ProposedNewState,
		}
	}
	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
		var diags tfdiags.Diagnostics
		diags = diags.Append(fmt.Errorf("update failed"))
		return providers.ApplyResourceChangeResponse{
			Diagnostics: diags,
		}
	}
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		State: states.BuildState(func(s *states.SyncState) {
			s.SetResourceInstanceCurrent(
				addrs.Resource{
					Mode: addrs.ManagedResourceMode,
					Type: "test",
					Name: "foo",
				}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
				&states.ResourceInstanceObjectSrc{
					AttrsJSON: []byte(`{"baz":"old"}`),
					Status:    states.ObjectReady,
				},
				addrs.ProviderConfig{
					Type: "test",
				}.Absolute(addrs.RootModuleInstance),
			)
		}),
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"test": testProviderFuncFixed(p),
			},
		),
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, diags := ctx.Apply()
	if !diags.HasErrors() {
		t.Fatalf("apply succeeded; want error")
	}
	if got, want := diags.Err().Error(), "update failed"; !strings.Contains(got, want) {
		t.Fatalf("wrong error\ngot: %s\nshould contain: %s", got, want)
	}

	mod := state.RootModule()
	rs := mod.Resources["test.foo"]
	if rs == nil {
		t.Fatalf("test.foo not in state after apply, but should be")
	}
	is := rs.Instances[addrs.NoKey]
	if is == nil {
		t.Fatalf("test.foo not in state after apply, but should be")
	}
	obj := is.Current
	if obj == nil {
		t.Fatalf("test.foo has no current object in state after apply, but should do")
	}

	if got, want := obj.Status, states.ObjectReady; got != want {
		t.Errorf("test.foo has wrong status %s after apply; want %s", got, want)
	}
	if got, want := obj.AttrsJSON, []byte(`"old"`); !bytes.Contains(got, want) {
		t.Errorf("test.foo attributes JSON doesn't contain %s after apply\ngot: %s", want, got)
	}
}

func TestContext2Apply_invalidIndexRef(t *testing.T) {
	p := testProvider("test")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_instance": {
				Attributes: map[string]*configschema.Attribute{
					"value": {Type: cty.String, Optional: true, Computed: true},
				},
			},
		},
	}
	p.DiffFn = testDiffFn

	m := testModule(t, "apply-invalid-index")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"test": testProviderFuncFixed(p),
			},
		),
	})

	diags := c.Validate()
	if diags.HasErrors() {
		t.Fatalf("unexpected validation failure: %s", diags.Err())
	}

	wantErr := `The given key does not identify an element in this collection value`
	_, diags = c.Plan()

	if !diags.HasErrors() {
		t.Fatalf("plan succeeded; want error")
	}
	gotErr := diags.Err().Error()

	if !strings.Contains(gotErr, wantErr) {
		t.Fatalf("missing expected error\ngot: %s\n\nwant: error containing %q", gotErr, wantErr)
	}
}
