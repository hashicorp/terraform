package terraform

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/zclconf/go-cty/cty"
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
		Config: m,
		Hooks:  []Hook{hook},
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
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
		Config:    m,
		Providers: ps,
	})

	_, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	_, diags = ctx.Apply()
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	// now destroy the whole thing
	ctx = testContext2(t, &ContextOpts{
		Config:    m,
		Providers: ps,
		PlanMode:  plans.DestroyMode,
	})

	_, diags = ctx.Plan()
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	otherP.ConfigureProviderFn = func(req providers.ConfigureProviderRequest) (resp providers.ConfigureProviderResponse) {
		// should not be used to destroy data sources
		resp.Diagnostics = resp.Diagnostics.Append(errors.New("provider should not be used"))
		return resp
	}

	_, diags = ctx.Apply()
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
		Config: m,
		State:  state,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	_, diags := ctx.Apply()
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
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		State: state,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

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

	state, diags = ctx.Apply()
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
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
		State: state,
	})

	_, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}

	_, diags = ctx.Apply()
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
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}

	state, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}

	obj := state.ResourceInstance(mustResourceInstanceAddr("test_object.a"))
	if len(obj.Current.AttrSensitivePaths) != 1 {
		t.Fatalf("Expected 1 sensitive mark for test_object.a, got %#v\n", obj.Current.AttrSensitivePaths)
	}

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}

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
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}

	_, diags = ctx.Apply()
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}

	// FINAL PLAN:
	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}

	// make sure the same marks are compared in the next plan as well
	for _, c := range plan.Changes.Resources {
		if c.Action != plans.NoOp {
			t.Logf("marks before: %#v", c.BeforeValMarks)
			t.Logf("marks after:  %#v", c.AfterValMarks)
			t.Errorf("Unexpcetd %s change for %s", c.Action, c.Addr)
		}
	}
}
