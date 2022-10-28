package terraform

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Test that the PreApply hook is called with the correct deposed key
func TestContext2Apply_createBeforeDestroy_deposedKeyPreApply(t *testing.T) {
	m := testModule(t, "apply-cbd-deposed-only")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn
	p.ApplyResourceChangeFn = testApplyFn

	deposedKey := states.NewDeposedKey()

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.bar").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"bar"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceDeposed(
		mustResourceInstanceAddr("aws_instance.bar").Resource,
		deposedKey,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectTainted,
			AttrsJSON: []byte(`{"id":"foo"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	hook := new(MockHook)
	ctx := testContext2(t, &ContextOpts{
		Hooks: []Hook{hook},
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, DefaultPlanOpts)
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	} else {
		t.Logf(legacyDiffComparisonString(plan.Changes))
	}

	_, diags = ctx.Apply(plan, m)
	if diags.HasErrors() {
		t.Fatalf("diags: %s", diags.Err())
	}

	// Verify PreApply was called correctly
	if !hook.PreApplyCalled {
		t.Fatalf("PreApply hook not called")
	}
	if addr, wantAddr := hook.PreApplyAddr, mustResourceInstanceAddr("aws_instance.bar"); !addr.Equal(wantAddr) {
		t.Errorf("expected addr to be %s, but was %s", wantAddr, addr)
	}
	if gen := hook.PreApplyGen; gen != deposedKey {
		t.Errorf("expected gen to be %q, but was %q", deposedKey, gen)
	}
}

func TestContext2Apply_destroyWithDataSourceExpansion(t *testing.T) {
	// While managed resources store their destroy-time dependencies, data
	// sources do not. This means that if a provider were only included in a
	// destroy graph because of data sources, it could have dependencies which
	// are not correctly ordered. Here we verify that the provider is not
	// included in the destroy operation, and all dependency evaluations
	// succeed.

	m := testModuleInline(t, map[string]string{
		"main.tf": `
module "mod" {
  source = "./mod"
}

provider "other" {
  foo = module.mod.data
}

# this should not require the provider be present during destroy
data "other_data_source" "a" {
}
`,

		"mod/main.tf": `
data "test_data_source" "a" {
  count = 1
}

data "test_data_source" "b" {
  count = data.test_data_source.a[0].foo == "ok" ? 1 : 0
}

output "data" {
  value = data.test_data_source.a[0].foo == "ok" ? data.test_data_source.b[0].foo : "nope"
}
`,
	})

	testP := testProvider("test")
	otherP := testProvider("other")

	readData := func(req providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
		return providers.ReadDataSourceResponse{
			State: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("data_source"),
				"foo": cty.StringVal("ok"),
			}),
		}
	}

	testP.ReadDataSourceFn = readData
	otherP.ReadDataSourceFn = readData

	ps := map[addrs.Provider]providers.Factory{
		addrs.NewDefaultProvider("test"):  testProviderFuncFixed(testP),
		addrs.NewDefaultProvider("other"): testProviderFuncFixed(otherP),
	}

	otherP.ConfigureProviderFn = func(req providers.ConfigureProviderRequest) (resp providers.ConfigureProviderResponse) {
		foo := req.Config.GetAttr("foo")
		if foo.IsNull() || foo.AsString() != "ok" {
			resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("incorrect config val: %#v\n", foo))
		}
		return resp
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: ps,
	})

	plan, diags := ctx.Plan(m, states.NewState(), DefaultPlanOpts)
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	_, diags = ctx.Apply(plan, m)
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// now destroy the whole thing
	ctx = testContext2(t, &ContextOpts{
		Providers: ps,
	})

	plan, diags = ctx.Plan(m, states.NewState(), &PlanOpts{
		Mode: plans.DestroyMode,
	})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	otherP.ConfigureProviderFn = func(req providers.ConfigureProviderRequest) (resp providers.ConfigureProviderResponse) {
		// should not be used to destroy data sources
		resp.Diagnostics = resp.Diagnostics.Append(errors.New("provider should not be used"))
		return resp
	}

	_, diags = ctx.Apply(plan, m)
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}
}

func TestContext2Apply_destroyThenUpdate(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_instance" "a" {
	value = "udpated"
}
`,
	})

	p := testProvider("test")
	p.PlanResourceChangeFn = testDiffFn

	var orderMu sync.Mutex
	var order []string
	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) (resp providers.ApplyResourceChangeResponse) {
		id := req.PriorState.GetAttr("id").AsString()
		if id == "b" {
			// slow down the b destroy, since a should wait for it
			time.Sleep(100 * time.Millisecond)
		}

		orderMu.Lock()
		order = append(order, id)
		orderMu.Unlock()

		resp.NewState = req.PlannedState
		return resp
	}

	addrA := mustResourceInstanceAddr(`test_instance.a`)
	addrB := mustResourceInstanceAddr(`test_instance.b`)

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(addrA, &states.ResourceInstanceObjectSrc{
			AttrsJSON: []byte(`{"id":"a","value":"old","type":"test"}`),
			Status:    states.ObjectReady,
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))

		// test_instance.b depended on test_instance.a, and therefor should be
		// destroyed before any changes to test_instance.a
		s.SetResourceInstanceCurrent(addrB, &states.ResourceInstanceObjectSrc{
			AttrsJSON:    []byte(`{"id":"b"}`),
			Status:       states.ObjectReady,
			Dependencies: []addrs.ConfigResource{addrA.ContainingResource().Config()},
		}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
	})

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, DefaultPlanOpts)
	assertNoErrors(t, diags)

	_, diags = ctx.Apply(plan, m)
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	if order[0] != "b" {
		t.Fatalf("expected apply order [b, a], got: %v\n", order)
	}
}

// verify that dependencies are updated in the state during refresh and apply
func TestApply_updateDependencies(t *testing.T) {
	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)

	fooAddr := mustResourceInstanceAddr("aws_instance.foo")
	barAddr := mustResourceInstanceAddr("aws_instance.bar")
	bazAddr := mustResourceInstanceAddr("aws_instance.baz")
	bamAddr := mustResourceInstanceAddr("aws_instance.bam")
	binAddr := mustResourceInstanceAddr("aws_instance.bin")
	root.SetResourceInstanceCurrent(
		fooAddr.Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"foo"}`),
			Dependencies: []addrs.ConfigResource{
				bazAddr.ContainingResource().Config(),
			},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceCurrent(
		binAddr.Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"bin","type":"aws_instance","unknown":"ok"}`),
			Dependencies: []addrs.ConfigResource{
				bazAddr.ContainingResource().Config(),
			},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceCurrent(
		bazAddr.Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"baz"}`),
			Dependencies: []addrs.ConfigResource{
				// Existing dependencies should not be removed from orphaned instances
				bamAddr.ContainingResource().Config(),
			},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceCurrent(
		barAddr.Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"bar","foo":"foo"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "aws_instance" "bar" {
  foo = aws_instance.foo.id
}

resource "aws_instance" "foo" {
}

resource "aws_instance" "bin" {
}
`,
	})

	p := testProvider("aws")

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, DefaultPlanOpts)
	assertNoErrors(t, diags)

	bar := plan.PriorState.ResourceInstance(barAddr)
	if len(bar.Current.Dependencies) == 0 || !bar.Current.Dependencies[0].Equal(fooAddr.ContainingResource().Config()) {
		t.Fatalf("bar should depend on foo after refresh, but got %s", bar.Current.Dependencies)
	}

	foo := plan.PriorState.ResourceInstance(fooAddr)
	if len(foo.Current.Dependencies) == 0 || !foo.Current.Dependencies[0].Equal(bazAddr.ContainingResource().Config()) {
		t.Fatalf("foo should depend on baz after refresh because of the update, but got %s", foo.Current.Dependencies)
	}

	bin := plan.PriorState.ResourceInstance(binAddr)
	if len(bin.Current.Dependencies) != 0 {
		t.Fatalf("bin should depend on nothing after refresh because there is no change, but got %s", bin.Current.Dependencies)
	}

	baz := plan.PriorState.ResourceInstance(bazAddr)
	if len(baz.Current.Dependencies) == 0 || !baz.Current.Dependencies[0].Equal(bamAddr.ContainingResource().Config()) {
		t.Fatalf("baz should depend on bam after refresh, but got %s", baz.Current.Dependencies)
	}

	state, diags = ctx.Apply(plan, m)
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	bar = state.ResourceInstance(barAddr)
	if len(bar.Current.Dependencies) == 0 || !bar.Current.Dependencies[0].Equal(fooAddr.ContainingResource().Config()) {
		t.Fatalf("bar should still depend on foo after apply, but got %s", bar.Current.Dependencies)
	}

	foo = state.ResourceInstance(fooAddr)
	if len(foo.Current.Dependencies) != 0 {
		t.Fatalf("foo should have no deps after apply, but got %s", foo.Current.Dependencies)
	}

}

func TestContext2Apply_additionalSensitiveFromState(t *testing.T) {
	// Ensure we're not trying to double-mark values decoded from state
	m := testModuleInline(t, map[string]string{
		"main.tf": `
variable "secret" {
  sensitive = true
  default = ["secret"]
}

resource "test_resource" "a" {
  sensitive_attr = var.secret
}

resource "test_resource" "b" {
  value = test_resource.a.id
}
`,
	})

	p := new(MockProvider)
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_resource": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"value": {
						Type:     cty.String,
						Optional: true,
					},
					"sensitive_attr": {
						Type:      cty.List(cty.String),
						Optional:  true,
						Sensitive: true,
					},
				},
			},
		},
	})

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			mustResourceInstanceAddr(`test_resource.a`),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"a","sensitive_attr":["secret"]}`),
				AttrSensitivePaths: []cty.PathValueMarks{
					{
						Path:  cty.GetAttrPath("sensitive_attr"),
						Marks: cty.NewValueMarks(marks.Sensitive),
					},
				},
				Status: states.ObjectReady,
			}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
	})

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, SimplePlanOpts(plans.NormalMode, testInputValuesUnset(m.Module.Variables)))
	assertNoErrors(t, diags)

	_, diags = ctx.Apply(plan, m)
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}
}

func TestContext2Apply_sensitiveOutputPassthrough(t *testing.T) {
	// Ensure we're not trying to double-mark values decoded from state
	m := testModuleInline(t, map[string]string{
		"main.tf": `
module "mod" {
  source = "./mod"
}

resource "test_object" "a" {
  test_string = module.mod.out
}
`,

		"mod/main.tf": `
variable "in" {
  sensitive = true
  default = "foo"
}
output "out" {
  value = var.in
}
`,
	})

	p := simpleMockProvider()

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, states.NewState(), DefaultPlanOpts)
	assertNoErrors(t, diags)

	state, diags := ctx.Apply(plan, m)
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}

	obj := state.ResourceInstance(mustResourceInstanceAddr("test_object.a"))
	if len(obj.Current.AttrSensitivePaths) != 1 {
		t.Fatalf("Expected 1 sensitive mark for test_object.a, got %#v\n", obj.Current.AttrSensitivePaths)
	}

	plan, diags = ctx.Plan(m, state, DefaultPlanOpts)
	assertNoErrors(t, diags)

	// make sure the same marks are compared in the next plan as well
	for _, c := range plan.Changes.Resources {
		if c.Action != plans.NoOp {
			t.Errorf("Unexpcetd %s change for %s", c.Action, c.Addr)
		}
	}
}

func TestContext2Apply_ignoreImpureFunctionChanges(t *testing.T) {
	// The impure function call should not cause a planned change with
	// ignore_changes
	m := testModuleInline(t, map[string]string{
		"main.tf": `
variable "pw" {
  sensitive = true
  default = "foo"
}

resource "test_object" "x" {
  test_map = {
	string = "X${bcrypt(var.pw)}"
  }
  lifecycle {
    ignore_changes = [ test_map["string"] ]
  }
}

resource "test_object" "y" {
  test_map = {
	string = "X${bcrypt(var.pw)}"
  }
  lifecycle {
    ignore_changes = [ test_map ]
  }
}

`,
	})

	p := simpleMockProvider()

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, states.NewState(), SimplePlanOpts(plans.NormalMode, testInputValuesUnset(m.Module.Variables)))
	assertNoErrors(t, diags)

	state, diags := ctx.Apply(plan, m)
	assertNoErrors(t, diags)

	// FINAL PLAN:
	plan, diags = ctx.Plan(m, state, SimplePlanOpts(plans.NormalMode, testInputValuesUnset(m.Module.Variables)))
	assertNoErrors(t, diags)

	// make sure the same marks are compared in the next plan as well
	for _, c := range plan.Changes.Resources {
		if c.Action != plans.NoOp {
			t.Logf("marks before: %#v", c.BeforeValMarks)
			t.Logf("marks after:  %#v", c.AfterValMarks)
			t.Errorf("Unexpcetd %s change for %s", c.Action, c.Addr)
		}
	}
}

func TestContext2Apply_destroyWithDeposed(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "x" {
  test_string = "ok"
  lifecycle {
    create_before_destroy = true
  }
}`,
	})

	p := simpleMockProvider()

	deposedKey := states.NewDeposedKey()

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceDeposed(
		mustResourceInstanceAddr("test_object.x").Resource,
		deposedKey,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectTainted,
			AttrsJSON: []byte(`{"test_string":"deposed"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.DestroyMode,
	})
	if diags.HasErrors() {
		t.Fatalf("plan: %s", diags.Err())
	}

	_, diags = ctx.Apply(plan, m)
	if diags.HasErrors() {
		t.Fatalf("apply: %s", diags.Err())
	}

}

func TestContext2Apply_nullableVariables(t *testing.T) {
	m := testModule(t, "apply-nullable-variables")
	state := states.NewState()
	ctx := testContext2(t, &ContextOpts{})
	plan, diags := ctx.Plan(m, state, &PlanOpts{})
	if diags.HasErrors() {
		t.Fatalf("plan: %s", diags.Err())
	}
	state, diags = ctx.Apply(plan, m)
	if diags.HasErrors() {
		t.Fatalf("apply: %s", diags.Err())
	}

	outputs := state.Module(addrs.RootModuleInstance).OutputValues
	// we check for null outputs be seeing that they don't exists
	if _, ok := outputs["nullable_null_default"]; ok {
		t.Error("nullable_null_default: expected no output value")
	}
	if _, ok := outputs["nullable_non_null_default"]; ok {
		t.Error("nullable_non_null_default: expected no output value")
	}
	if _, ok := outputs["nullable_no_default"]; ok {
		t.Error("nullable_no_default: expected no output value")
	}

	if v := outputs["non_nullable_default"].Value; v.AsString() != "ok" {
		t.Fatalf("incorrect 'non_nullable_default' output value: %#v\n", v)
	}
	if v := outputs["non_nullable_no_default"].Value; v.AsString() != "ok" {
		t.Fatalf("incorrect 'non_nullable_no_default' output value: %#v\n", v)
	}
}

func TestContext2Apply_targetedDestroyWithMoved(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
module "modb" {
  source = "./mod"
  for_each = toset(["a", "b"])
}
`,
		"./mod/main.tf": `
resource "test_object" "a" {
}

module "sub" {
  for_each = toset(["a", "b"])
  source = "./sub"
}

moved {
  from = module.old
  to = module.sub
}
`,
		"./mod/sub/main.tf": `
resource "test_object" "s" {
}
`})

	p := simpleMockProvider()

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, states.NewState(), DefaultPlanOpts)
	assertNoErrors(t, diags)

	state, diags := ctx.Apply(plan, m)
	assertNoErrors(t, diags)

	// destroy only a single instance not included in the moved statements
	_, diags = ctx.Plan(m, state, &PlanOpts{
		Mode:    plans.DestroyMode,
		Targets: []addrs.Targetable{mustResourceInstanceAddr(`module.modb["a"].test_object.a`)},
	})
	assertNoErrors(t, diags)
}

func TestContext2Apply_graphError(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "a" {
  test_string = "ok"
}

resource "test_object" "b" {
  test_string = test_object.a.test_string
}
`,
	})

	p := simpleMockProvider()

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.a").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectTainted,
			AttrsJSON: []byte(`{"test_string":"ok"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.b").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectTainted,
			AttrsJSON: []byte(`{"test_string":"ok"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.DestroyMode,
	})
	if diags.HasErrors() {
		t.Fatalf("plan: %s", diags.Err())
	}

	// We're going to corrupt the stored state so that the dependencies will
	// cause a cycle when building the apply graph.
	testObjA := plan.PriorState.Modules[""].Resources["test_object.a"].Instances[addrs.NoKey].Current
	testObjA.Dependencies = append(testObjA.Dependencies, mustResourceInstanceAddr("test_object.b").ContainingResource().Config())

	_, diags = ctx.Apply(plan, m)
	if !diags.HasErrors() {
		t.Fatal("expected cycle error from apply")
	}
}

func TestContext2Apply_resourcePostcondition(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
variable "boop" {
  type = string
}

resource "test_resource" "a" {
	value = var.boop
}

resource "test_resource" "b" {
  value = test_resource.a.output
  lifecycle {
    postcondition {
      condition     = self.output != ""
      error_message = "Output must not be blank."
    }
  }
}

resource "test_resource" "c" {
  value = test_resource.b.output
}
`,
	})

	p := testProvider("test")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_resource": {
				Attributes: map[string]*configschema.Attribute{
					"value": {
						Type:     cty.String,
						Required: true,
					},
					"output": {
						Type:     cty.String,
						Computed: true,
					},
				},
			},
		},
	})
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		m := req.ProposedNewState.AsValueMap()
		m["output"] = cty.UnknownVal(cty.String)

		resp.PlannedState = cty.ObjectVal(m)
		resp.LegacyTypeSystem = true
		return resp
	}
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	t.Run("condition pass", func(t *testing.T) {
		plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
			Mode: plans.NormalMode,
			SetVariables: InputValues{
				"boop": &InputValue{
					Value:      cty.StringVal("boop"),
					SourceType: ValueFromCLIArg,
				},
			},
		})
		assertNoErrors(t, diags)
		if len(plan.Changes.Resources) != 3 {
			t.Fatalf("unexpected plan changes: %#v", plan.Changes)
		}

		p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) (resp providers.ApplyResourceChangeResponse) {
			m := req.PlannedState.AsValueMap()
			m["output"] = cty.StringVal(fmt.Sprintf("new-%s", m["value"].AsString()))

			resp.NewState = cty.ObjectVal(m)
			return resp
		}
		state, diags := ctx.Apply(plan, m)
		assertNoErrors(t, diags)

		wantResourceAttrs := map[string]struct{ value, output string }{
			"a": {"boop", "new-boop"},
			"b": {"new-boop", "new-new-boop"},
			"c": {"new-new-boop", "new-new-new-boop"},
		}
		for name, attrs := range wantResourceAttrs {
			addr := mustResourceInstanceAddr(fmt.Sprintf("test_resource.%s", name))
			r := state.ResourceInstance(addr)
			rd, err := r.Current.Decode(cty.Object(map[string]cty.Type{
				"value":  cty.String,
				"output": cty.String,
			}))
			if err != nil {
				t.Fatalf("error decoding test_resource.a: %s", err)
			}
			want := cty.ObjectVal(map[string]cty.Value{
				"value":  cty.StringVal(attrs.value),
				"output": cty.StringVal(attrs.output),
			})
			if !cmp.Equal(want, rd.Value, valueComparer) {
				t.Errorf("wrong attrs for %s\n%s", addr, cmp.Diff(want, rd.Value, valueComparer))
			}
		}
	})
	t.Run("condition fail", func(t *testing.T) {
		plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
			Mode: plans.NormalMode,
			SetVariables: InputValues{
				"boop": &InputValue{
					Value:      cty.StringVal("boop"),
					SourceType: ValueFromCLIArg,
				},
			},
		})
		assertNoErrors(t, diags)
		if len(plan.Changes.Resources) != 3 {
			t.Fatalf("unexpected plan changes: %#v", plan.Changes)
		}

		p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) (resp providers.ApplyResourceChangeResponse) {
			m := req.PlannedState.AsValueMap()

			// For the resource with a constraint, fudge the output to make the
			// condition fail.
			if value := m["value"].AsString(); value == "new-boop" {
				m["output"] = cty.StringVal("")
			} else {
				m["output"] = cty.StringVal(fmt.Sprintf("new-%s", value))
			}

			resp.NewState = cty.ObjectVal(m)
			return resp
		}
		state, diags := ctx.Apply(plan, m)
		if !diags.HasErrors() {
			t.Fatal("succeeded; want errors")
		}
		if got, want := diags.Err().Error(), "Resource postcondition failed: Output must not be blank."; got != want {
			t.Fatalf("wrong error:\ngot:  %s\nwant: %q", got, want)
		}

		// Resources a and b should still be recorded in state
		wantResourceAttrs := map[string]struct{ value, output string }{
			"a": {"boop", "new-boop"},
			"b": {"new-boop", ""},
		}
		for name, attrs := range wantResourceAttrs {
			addr := mustResourceInstanceAddr(fmt.Sprintf("test_resource.%s", name))
			r := state.ResourceInstance(addr)
			rd, err := r.Current.Decode(cty.Object(map[string]cty.Type{
				"value":  cty.String,
				"output": cty.String,
			}))
			if err != nil {
				t.Fatalf("error decoding test_resource.a: %s", err)
			}
			want := cty.ObjectVal(map[string]cty.Value{
				"value":  cty.StringVal(attrs.value),
				"output": cty.StringVal(attrs.output),
			})
			if !cmp.Equal(want, rd.Value, valueComparer) {
				t.Errorf("wrong attrs for %s\n%s", addr, cmp.Diff(want, rd.Value, valueComparer))
			}
		}

		// Resource c should not be in state
		if state.ResourceInstance(mustResourceInstanceAddr("test_resource.c")) != nil {
			t.Error("test_resource.c should not exist in state, but is")
		}
	})
}

func TestContext2Apply_outputValuePrecondition(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
			variable "input" {
				type = string
			}

			module "child" {
				source = "./child"

				input = var.input
			}

			output "result" {
				value = module.child.result

				precondition {
					condition     = var.input != ""
					error_message = "Input must not be empty."
				}
			}
		`,
		"child/main.tf": `
			variable "input" {
				type = string
			}

			output "result" {
				value = var.input

				precondition {
					condition     = var.input != ""
					error_message = "Input must not be empty."
				}
			}
		`,
	})

	checkableObjects := []addrs.Checkable{
		addrs.OutputValue{Name: "result"}.Absolute(addrs.RootModuleInstance),
		addrs.OutputValue{Name: "result"}.Absolute(addrs.RootModuleInstance.Child("child", addrs.NoKey)),
	}

	t.Run("pass", func(t *testing.T) {
		ctx := testContext2(t, &ContextOpts{})
		plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
			Mode: plans.NormalMode,
			SetVariables: InputValues{
				"input": &InputValue{
					Value:      cty.StringVal("beep"),
					SourceType: ValueFromCLIArg,
				},
			},
		})
		assertNoDiagnostics(t, diags)

		for _, addr := range checkableObjects {
			result := plan.Checks.GetObjectResult(addr)
			if result == nil {
				t.Fatalf("no check result for %s in the plan", addr)
			}
			if got, want := result.Status, checks.StatusPass; got != want {
				t.Fatalf("wrong check status for %s during planning\ngot:  %s\nwant: %s", addr, got, want)
			}
		}

		state, diags := ctx.Apply(plan, m)
		assertNoDiagnostics(t, diags)
		for _, addr := range checkableObjects {
			result := state.CheckResults.GetObjectResult(addr)
			if result == nil {
				t.Fatalf("no check result for %s in the final state", addr)
			}
			if got, want := result.Status, checks.StatusPass; got != want {
				t.Errorf("wrong check status for %s after apply\ngot:  %s\nwant: %s", addr, got, want)
			}
		}
	})

	t.Run("fail", func(t *testing.T) {
		// NOTE: This test actually catches a failure during planning and so
		// cannot proceed to apply, so it's really more of a plan test
		// than an apply test but better to keep all of these
		// thematically-related test cases together.
		ctx := testContext2(t, &ContextOpts{})
		_, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
			Mode: plans.NormalMode,
			SetVariables: InputValues{
				"input": &InputValue{
					Value:      cty.StringVal(""),
					SourceType: ValueFromCLIArg,
				},
			},
		})
		if !diags.HasErrors() {
			t.Fatalf("succeeded; want error")
		}

		const wantSummary = "Module output value precondition failed"
		found := false
		for _, diag := range diags {
			if diag.Severity() == tfdiags.Error && diag.Description().Summary == wantSummary {
				found = true
				break
			}
		}

		if !found {
			t.Fatalf("missing expected error\nwant summary: %s\ngot: %s", wantSummary, diags.Err().Error())
		}
	})
}

func TestContext2Apply_resourceConditionApplyTimeFail(t *testing.T) {
	// This tests the less common situation where a condition fails due to
	// a change in a resource other than the one the condition is attached to,
	// and the condition result is unknown during planning.
	//
	// This edge case is a tricky one because it relies on Terraform still
	// visiting test_resource.b (in the configuration below) to evaluate
	// its conditions even though there aren't any changes directly planned
	// for it, so that we can consider whether changes to test_resource.a
	// have changed the outcome.

	m := testModuleInline(t, map[string]string{
		"main.tf": `
			variable "input" {
				type = string
			}

			resource "test_resource" "a" {
				value = var.input
			}

			resource "test_resource" "b" {
				value = "beep"

				lifecycle {
					postcondition {
						condition     = test_resource.a.output == self.output
						error_message = "Outputs must match."
					}
				}
			}
		`,
	})

	p := testProvider("test")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_resource": {
				Attributes: map[string]*configschema.Attribute{
					"value": {
						Type:     cty.String,
						Required: true,
					},
					"output": {
						Type:     cty.String,
						Computed: true,
					},
				},
			},
		},
	})
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		// Whenever "value" changes, "output" follows it during the apply step,
		// but is initially unknown during the plan step.

		m := req.ProposedNewState.AsValueMap()
		priorVal := cty.NullVal(cty.String)
		if !req.PriorState.IsNull() {
			priorVal = req.PriorState.GetAttr("value")
		}
		if m["output"].IsNull() || !priorVal.RawEquals(m["value"]) {
			m["output"] = cty.UnknownVal(cty.String)
		}

		resp.PlannedState = cty.ObjectVal(m)
		resp.LegacyTypeSystem = true
		return resp
	}
	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) (resp providers.ApplyResourceChangeResponse) {
		m := req.PlannedState.AsValueMap()
		m["output"] = m["value"]
		resp.NewState = cty.ObjectVal(m)
		return resp
	}
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})
	instA := mustResourceInstanceAddr("test_resource.a")
	instB := mustResourceInstanceAddr("test_resource.b")

	// Preparation: an initial plan and apply with a correct input variable
	// should succeed and give us a valid and complete state to use for the
	// subsequent plan and apply that we'll expect to fail.
	var prevRunState *states.State
	{
		plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
			Mode: plans.NormalMode,
			SetVariables: InputValues{
				"input": &InputValue{
					Value:      cty.StringVal("beep"),
					SourceType: ValueFromCLIArg,
				},
			},
		})
		assertNoErrors(t, diags)
		planA := plan.Changes.ResourceInstance(instA)
		if planA == nil || planA.Action != plans.Create {
			t.Fatalf("incorrect initial plan for instance A\nwant a 'create' change\ngot: %s", spew.Sdump(planA))
		}
		planB := plan.Changes.ResourceInstance(instB)
		if planB == nil || planB.Action != plans.Create {
			t.Fatalf("incorrect initial plan for instance B\nwant a 'create' change\ngot: %s", spew.Sdump(planB))
		}

		state, diags := ctx.Apply(plan, m)
		assertNoErrors(t, diags)

		stateA := state.ResourceInstance(instA)
		if stateA == nil || stateA.Current == nil || !bytes.Contains(stateA.Current.AttrsJSON, []byte(`"beep"`)) {
			t.Fatalf("incorrect initial state for instance A\ngot: %s", spew.Sdump(stateA))
		}
		stateB := state.ResourceInstance(instB)
		if stateB == nil || stateB.Current == nil || !bytes.Contains(stateB.Current.AttrsJSON, []byte(`"beep"`)) {
			t.Fatalf("incorrect initial state for instance B\ngot: %s", spew.Sdump(stateB))
		}
		prevRunState = state
	}

	// Now we'll run another plan and apply with a different value for
	// var.input that should cause the test_resource.b condition to be unknown
	// during planning and then fail during apply.
	{
		plan, diags := ctx.Plan(m, prevRunState, &PlanOpts{
			Mode: plans.NormalMode,
			SetVariables: InputValues{
				"input": &InputValue{
					Value:      cty.StringVal("boop"), // NOTE: This has changed
					SourceType: ValueFromCLIArg,
				},
			},
		})
		assertNoErrors(t, diags)
		planA := plan.Changes.ResourceInstance(instA)
		if planA == nil || planA.Action != plans.Update {
			t.Fatalf("incorrect initial plan for instance A\nwant an 'update' change\ngot: %s", spew.Sdump(planA))
		}
		planB := plan.Changes.ResourceInstance(instB)
		if planB == nil || planB.Action != plans.NoOp {
			t.Fatalf("incorrect initial plan for instance B\nwant a 'no-op' change\ngot: %s", spew.Sdump(planB))
		}

		_, diags = ctx.Apply(plan, m)
		if !diags.HasErrors() {
			t.Fatal("final apply succeeded, but should've failed with a postcondition error")
		}
		if len(diags) != 1 {
			t.Fatalf("expected exactly one diagnostic, but got: %s", diags.Err().Error())
		}
		if got, want := diags[0].Description().Summary, "Resource postcondition failed"; got != want {
			t.Fatalf("wrong diagnostic summary\ngot:  %s\nwant: %s", got, want)
		}
	}
}

// pass an input through some expanded values, and back to a provider to make
// sure we can fully evaluate a provider configuration during a destroy plan.
func TestContext2Apply_destroyWithConfiguredProvider(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
variable "in" {
  type = map(string)
  default = {
    "a" = "first"
    "b" = "second"
  }
}

module "mod" {
  source = "./mod"
  for_each = var.in
  in = each.value
}

locals {
  config = [for each in module.mod : each.out]
}

provider "other" {
  output = [for each in module.mod : each.out]
  local = local.config
  var = var.in
}

resource "other_object" "other" {
}
`,
		"./mod/main.tf": `
variable "in" {
  type = string
}

data "test_object" "d" {
  test_string = var.in
}

resource "test_object" "a" {
  test_string = var.in
}

output "out" {
  value = data.test_object.d.output
}
`})

	testProvider := &MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			Provider: providers.Schema{Block: simpleTestSchema()},
			ResourceTypes: map[string]providers.Schema{
				"test_object": providers.Schema{Block: simpleTestSchema()},
			},
			DataSources: map[string]providers.Schema{
				"test_object": providers.Schema{
					Block: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"test_string": {
								Type:     cty.String,
								Optional: true,
							},
							"output": {
								Type:     cty.String,
								Computed: true,
							},
						},
					},
				},
			},
		},
	}

	testProvider.ReadDataSourceFn = func(req providers.ReadDataSourceRequest) (resp providers.ReadDataSourceResponse) {
		cfg := req.Config.AsValueMap()
		s := cfg["test_string"].AsString()
		if !strings.Contains("firstsecond", s) {
			resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("expected 'first' or 'second', got %s", s))
			return resp
		}

		cfg["output"] = cty.StringVal(s + "-ok")
		resp.State = cty.ObjectVal(cfg)
		return resp
	}

	otherProvider := &MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			Provider: providers.Schema{
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"output": {
							Type:     cty.List(cty.String),
							Optional: true,
						},
						"local": {
							Type:     cty.List(cty.String),
							Optional: true,
						},
						"var": {
							Type:     cty.Map(cty.String),
							Optional: true,
						},
					},
				},
			},
			ResourceTypes: map[string]providers.Schema{
				"other_object": providers.Schema{Block: simpleTestSchema()},
			},
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"):  testProviderFuncFixed(testProvider),
			addrs.NewDefaultProvider("other"): testProviderFuncFixed(otherProvider),
		},
	})

	opts := SimplePlanOpts(plans.NormalMode, testInputValuesUnset(m.Module.Variables))
	plan, diags := ctx.Plan(m, states.NewState(), opts)
	assertNoErrors(t, diags)

	state, diags := ctx.Apply(plan, m)
	assertNoErrors(t, diags)

	// Resource changes which have dependencies across providers which
	// themselves depend on resources can result in cycles.
	// Because other_object transitively depends on the module resources
	// through its provider, we trigger changes on both sides of this boundary
	// to ensure we can create a valid plan.
	//
	// Taint the object to make sure a replacement works in the plan.
	otherObjAddr := mustResourceInstanceAddr("other_object.other")
	otherObj := state.ResourceInstance(otherObjAddr)
	otherObj.Current.Status = states.ObjectTainted
	// Force a change which needs to be reverted.
	testObjAddr := mustResourceInstanceAddr(`module.mod["a"].test_object.a`)
	testObjA := state.ResourceInstance(testObjAddr)
	testObjA.Current.AttrsJSON = []byte(`{"test_bool":null,"test_list":null,"test_map":null,"test_number":null,"test_string":"changed"}`)

	_, diags = ctx.Plan(m, state, opts)
	assertNoErrors(t, diags)
	return

	otherProvider.ConfigureProviderCalled = false
	otherProvider.ConfigureProviderFn = func(req providers.ConfigureProviderRequest) (resp providers.ConfigureProviderResponse) {
		// check that our config is complete, even during a destroy plan
		expected := cty.ObjectVal(map[string]cty.Value{
			"local":  cty.ListVal([]cty.Value{cty.StringVal("first-ok"), cty.StringVal("second-ok")}),
			"output": cty.ListVal([]cty.Value{cty.StringVal("first-ok"), cty.StringVal("second-ok")}),
			"var": cty.MapVal(map[string]cty.Value{
				"a": cty.StringVal("first"),
				"b": cty.StringVal("second"),
			}),
		})

		if !req.Config.RawEquals(expected) {
			resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf(
				`incorrect provider config:
expected: %#v
got:      %#v`,
				expected, req.Config))
		}

		return resp
	}

	opts.Mode = plans.DestroyMode
	// skip refresh so that we don't configure the provider before the destroy plan
	opts.SkipRefresh = true

	// destroy only a single instance not included in the moved statements
	_, diags = ctx.Plan(m, state, opts)
	assertNoErrors(t, diags)

	if !otherProvider.ConfigureProviderCalled {
		t.Fatal("failed to configure provider during destroy plan")
	}
}

// check that a provider can verify a planned destroy
func TestContext2Apply_plannedDestroy(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "x" {
  test_string = "ok"
}`,
	})

	p := simpleMockProvider()
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		if !req.ProposedNewState.IsNull() {
			// we should only be destroying in this test
			resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("unexpected plan with %#v", req.ProposedNewState))
			return resp
		}

		resp.PlannedState = req.ProposedNewState
		// we're going to verify the destroy plan by inserting private data required for destroy
		resp.PlannedPrivate = append(resp.PlannedPrivate, []byte("planned")...)
		return resp
	}

	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) (resp providers.ApplyResourceChangeResponse) {
		// if the value is nil, we return that directly to correspond to a delete
		if !req.PlannedState.IsNull() {
			resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("unexpected apply with %#v", req.PlannedState))
			return resp
		}

		resp.NewState = req.PlannedState

		// make sure we get our private data from the plan
		private := string(req.PlannedPrivate)
		if private != "planned" {
			resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("missing private data from plan, got %q", private))
		}
		return resp
	}

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.x").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"test_string":"ok"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, state, &PlanOpts{
		Mode: plans.DestroyMode,
		// we don't want to refresh, because that actually runs a normal plan
		SkipRefresh: true,
	})
	if diags.HasErrors() {
		t.Fatalf("plan: %s", diags.Err())
	}

	_, diags = ctx.Apply(plan, m)
	if diags.HasErrors() {
		t.Fatalf("apply: %s", diags.Err())
	}
}

func TestContext2Apply_missingOrphanedResource(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
# changed resource address to create a new object
resource "test_object" "y" {
  test_string = "y"
}
`,
	})

	p := simpleMockProvider()

	// report the prior value is missing
	p.ReadResourceFn = func(req providers.ReadResourceRequest) (resp providers.ReadResourceResponse) {
		resp.NewState = cty.NullVal(req.PriorState.Type())
		return resp
	}

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.x").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"test_string":"x"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	opts := SimplePlanOpts(plans.NormalMode, nil)
	plan, diags := ctx.Plan(m, state, opts)
	assertNoErrors(t, diags)

	_, diags = ctx.Apply(plan, m)
	assertNoErrors(t, diags)
}

// Outputs should not cause evaluation errors during destroy
// Check eval from both root level outputs and module outputs, which are
// handled differently during apply.
func TestContext2Apply_outputsNotToEvaluate(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
module "mod" {
  source = "./mod"
  cond = false
}

output "from_resource" {
  value = module.mod.from_resource
}

output "from_data" {
  value = module.mod.from_data
}
`,

		"./mod/main.tf": `
variable "cond" {
  type = bool
}

module "mod" {
  source = "../mod2/"
  cond = var.cond
}

output "from_resource" {
  value = module.mod.resource
}

output "from_data" {
  value = module.mod.data
}
`,

		"./mod2/main.tf": `
variable "cond" {
  type = bool
}

resource "test_object" "x" {
  count = var.cond ? 0:1
}

data "test_object" "d" {
  count = var.cond ? 0:1
}

output "resource" {
  value = var.cond ? null : test_object.x.*.test_string[0]
}

output "data" {
  value = one(data.test_object.d[*].test_string)
}
`})

	p := simpleMockProvider()
	p.ReadDataSourceFn = func(req providers.ReadDataSourceRequest) (resp providers.ReadDataSourceResponse) {
		resp.State = req.Config
		return resp
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	// apply the state
	opts := SimplePlanOpts(plans.NormalMode, nil)
	plan, diags := ctx.Plan(m, states.NewState(), opts)
	assertNoErrors(t, diags)

	state, diags := ctx.Apply(plan, m)
	assertNoErrors(t, diags)

	// and destroy
	opts = SimplePlanOpts(plans.DestroyMode, nil)
	plan, diags = ctx.Plan(m, state, opts)
	assertNoErrors(t, diags)

	state, diags = ctx.Apply(plan, m)
	assertNoErrors(t, diags)

	// and destroy again with no state
	if !state.Empty() {
		t.Fatal("expected empty state, got", state)
	}

	opts = SimplePlanOpts(plans.DestroyMode, nil)
	plan, diags = ctx.Plan(m, state, opts)
	assertNoErrors(t, diags)

	_, diags = ctx.Apply(plan, m)
	assertNoErrors(t, diags)
}

// don't evaluate conditions on outputs when destroying
func TestContext2Apply_noOutputChecksOnDestroy(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
module "mod" {
  source = "./mod"
}

output "from_resource" {
  value = module.mod.from_resource
}
`,

		"./mod/main.tf": `
resource "test_object" "x" {
  test_string = "wrong val"
}

output "from_resource" {
  value = test_object.x.test_string
  precondition {
    condition     = test_object.x.test_string == "ok"
    error_message = "resource error"
  }
}
`})

	p := simpleMockProvider()

	state := states.NewState()
	mod := state.EnsureModule(addrs.RootModuleInstance.Child("mod", addrs.NoKey))
	mod.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.x").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"test_string":"wrong_val"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	opts := SimplePlanOpts(plans.DestroyMode, nil)
	plan, diags := ctx.Plan(m, state, opts)
	assertNoErrors(t, diags)

	_, diags = ctx.Apply(plan, m)
	assertNoErrors(t, diags)
}

// -refresh-only should update checks
func TestContext2Apply_refreshApplyUpdatesChecks(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_object" "x" {
  test_string = "ok"
  lifecycle {
    postcondition {
      condition = self.test_string == "ok"
      error_message = "wrong val"
    }
  }
}

output "from_resource" {
  value = test_object.x.test_string
  precondition {
	condition     = test_object.x.test_string == "ok"
	error_message = "wrong val"
  }
}
`})

	p := simpleMockProvider()
	p.ReadResourceResponse = &providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"test_string": cty.StringVal("ok"),
		}),
	}

	state := states.NewState()
	mod := state.EnsureModule(addrs.RootModuleInstance)
	mod.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.x").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"test_string":"wrong val"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)
	mod.SetOutputValue("from_resource", cty.StringVal("wrong val"), false)

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	opts := SimplePlanOpts(plans.RefreshOnlyMode, nil)
	plan, diags := ctx.Plan(m, state, opts)
	assertNoErrors(t, diags)

	state, diags = ctx.Apply(plan, m)
	assertNoErrors(t, diags)

	resCheck := state.CheckResults.GetObjectResult(mustResourceInstanceAddr("test_object.x"))
	if resCheck.Status != checks.StatusPass {
		t.Fatalf("unexpected check %s: %s\n", resCheck.Status, resCheck.FailureMessages)
	}

	outAddr := addrs.AbsOutputValue{
		Module: addrs.RootModuleInstance,
		OutputValue: addrs.OutputValue{
			Name: "from_resource",
		},
	}
	outCheck := state.CheckResults.GetObjectResult(outAddr)
	if outCheck.Status != checks.StatusPass {
		t.Fatalf("unexpected check %s: %s\n", outCheck.Status, outCheck.FailureMessages)
	}
}
