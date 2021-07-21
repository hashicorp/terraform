package terraform

import (
	"bytes"
	"errors"
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

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/configs/hcl2shim"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/provisioners"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestContext2Plan_basic(t *testing.T) {
	m := testModule(t, "plan-good")
	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
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

	if !ctx.State().Empty() {
		t.Fatalf("expected empty state, got %#v\n", ctx.State())
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()
	for _, r := range plan.Changes.Resources {
		ric, err := r.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "aws_instance.bar":
			foo := ric.After.GetAttr("foo").AsString()
			if foo != "2" {
				t.Fatalf("incorrect plan for 'bar': %#v", ric.After)
			}
		case "aws_instance.foo":
			num, _ := ric.After.GetAttr("num").AsBigFloat().Int64()
			if num != 2 {
				t.Fatalf("incorrect plan for 'foo': %#v", ric.After)
			}
		default:
			t.Fatal("unknown instance:", i)
		}
	}

	if !p.ValidateProviderConfigCalled {
		t.Fatal("provider config was not checked before Configure")
	}

}

func TestContext2Plan_createBefore_deposed(t *testing.T) {
	m := testModule(t, "plan-cbd")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"baz","type":"aws_instance"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceDeposed(
		mustResourceInstanceAddr("aws_instance.foo").Resource,
		states.DeposedKey("00000001"),
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"foo"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		State: state,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	// the state should still show one deposed
	expectedState := strings.TrimSpace(`
 aws_instance.foo: (1 deposed)
  ID = baz
  provider = provider["registry.terraform.io/hashicorp/aws"]
  type = aws_instance
  Deposed ID 1 = foo`)

	if ctx.State().String() != expectedState {
		t.Fatalf("\nexpected: %q\ngot:      %q\n", expectedState, ctx.State().String())
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	type InstanceGen struct {
		Addr       string
		DeposedKey states.DeposedKey
	}
	want := map[InstanceGen]bool{
		{
			Addr: "aws_instance.foo",
		}: true,
		{
			Addr:       "aws_instance.foo",
			DeposedKey: states.DeposedKey("00000001"),
		}: true,
	}
	got := make(map[InstanceGen]bool)
	changes := make(map[InstanceGen]*plans.ResourceInstanceChangeSrc)

	for _, change := range plan.Changes.Resources {
		k := InstanceGen{
			Addr:       change.Addr.String(),
			DeposedKey: change.DeposedKey,
		}
		got[k] = true
		changes[k] = change
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("wrong resource instance object changes in plan\ngot: %s\nwant: %s", spew.Sdump(got), spew.Sdump(want))
	}

	{
		ric, err := changes[InstanceGen{Addr: "aws_instance.foo"}].Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		if got, want := ric.Action, plans.NoOp; got != want {
			t.Errorf("current object change action is %s; want %s", got, want)
		}

		// the existing instance should only have an unchanged id
		expected, err := schema.CoerceValue(cty.ObjectVal(map[string]cty.Value{
			"id":   cty.StringVal("baz"),
			"type": cty.StringVal("aws_instance"),
		}))
		if err != nil {
			t.Fatal(err)
		}

		checkVals(t, expected, ric.After)
	}

	{
		ric, err := changes[InstanceGen{Addr: "aws_instance.foo", DeposedKey: states.DeposedKey("00000001")}].Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		if got, want := ric.Action, plans.Delete; got != want {
			t.Errorf("deposed object change action is %s; want %s", got, want)
		}
	}
}

func TestContext2Plan_createBefore_maintainRoot(t *testing.T) {
	m := testModule(t, "plan-cbd-maintain-root")
	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if !ctx.State().Empty() {
		t.Fatal("expected empty state, got:", ctx.State())
	}

	if len(plan.Changes.Resources) != 4 {
		t.Error("expected 4 resource in plan, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		// these should all be creates
		if res.Action != plans.Create {
			t.Fatalf("unexpected action %s for %s", res.Action, res.Addr.String())
		}
	}
}

func TestContext2Plan_emptyDiff(t *testing.T) {
	m := testModule(t, "plan-empty")
	p := testProvider("aws")
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		resp.PlannedState = req.ProposedNewState
		return resp
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if !ctx.State().Empty() {
		t.Fatal("expected empty state, got:", ctx.State())
	}

	if len(plan.Changes.Resources) != 2 {
		t.Error("expected 2 resource in plan, got", len(plan.Changes.Resources))
	}

	actions := map[string]plans.Action{}

	for _, res := range plan.Changes.Resources {
		actions[res.Addr.String()] = res.Action
	}

	expected := map[string]plans.Action{
		"aws_instance.foo": plans.Create,
		"aws_instance.bar": plans.Create,
	}
	if !cmp.Equal(expected, actions) {
		t.Fatal(cmp.Diff(expected, actions))
	}
}

func TestContext2Plan_escapedVar(t *testing.T) {
	m := testModule(t, "plan-escaped-var")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if len(plan.Changes.Resources) != 1 {
		t.Error("expected 1 resource in plan, got", len(plan.Changes.Resources))
	}

	res := plan.Changes.Resources[0]
	if res.Action != plans.Create {
		t.Fatalf("expected resource creation, got %s", res.Action)
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	ric, err := res.Decode(ty)
	if err != nil {
		t.Fatal(err)
	}

	expected := objectVal(t, schema, map[string]cty.Value{
		"id":   cty.UnknownVal(cty.String),
		"foo":  cty.StringVal("bar-${baz}"),
		"type": cty.UnknownVal(cty.String),
	})

	checkVals(t, expected, ric.After)
}

func TestContext2Plan_minimal(t *testing.T) {
	m := testModule(t, "plan-empty")
	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if !ctx.State().Empty() {
		t.Fatal("expected empty state, got:", ctx.State())
	}

	if len(plan.Changes.Resources) != 2 {
		t.Error("expected 2 resource in plan, got", len(plan.Changes.Resources))
	}

	actions := map[string]plans.Action{}

	for _, res := range plan.Changes.Resources {
		actions[res.Addr.String()] = res.Action
	}

	expected := map[string]plans.Action{
		"aws_instance.foo": plans.Create,
		"aws_instance.bar": plans.Create,
	}
	if !cmp.Equal(expected, actions) {
		t.Fatal(cmp.Diff(expected, actions))
	}
}

func TestContext2Plan_modules(t *testing.T) {
	m := testModule(t, "plan-modules")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if len(plan.Changes.Resources) != 3 {
		t.Error("expected 3 resource in plan, got", len(plan.Changes.Resources))
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	expectFoo := objectVal(t, schema, map[string]cty.Value{
		"id":   cty.UnknownVal(cty.String),
		"foo":  cty.StringVal("2"),
		"type": cty.UnknownVal(cty.String),
	})

	expectNum := objectVal(t, schema, map[string]cty.Value{
		"id":   cty.UnknownVal(cty.String),
		"num":  cty.NumberIntVal(2),
		"type": cty.UnknownVal(cty.String),
	})

	for _, res := range plan.Changes.Resources {
		if res.Action != plans.Create {
			t.Fatalf("expected resource creation, got %s", res.Action)
		}
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		var expected cty.Value
		switch i := ric.Addr.String(); i {
		case "aws_instance.bar":
			expected = expectFoo
		case "aws_instance.foo":
			expected = expectNum
		case "module.child.aws_instance.foo":
			expected = expectNum
		default:
			t.Fatal("unknown instance:", i)
		}

		checkVals(t, expected, ric.After)
	}
}
func TestContext2Plan_moduleExpand(t *testing.T) {
	// Test a smattering of plan expansion behavior
	m := testModule(t, "plan-modules-expand")
	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	expected := map[string]struct{}{
		`aws_instance.foo["a"]`:                          struct{}{},
		`module.count_child[1].aws_instance.foo[0]`:      struct{}{},
		`module.count_child[1].aws_instance.foo[1]`:      struct{}{},
		`module.count_child[0].aws_instance.foo[0]`:      struct{}{},
		`module.count_child[0].aws_instance.foo[1]`:      struct{}{},
		`module.for_each_child["a"].aws_instance.foo[1]`: struct{}{},
		`module.for_each_child["a"].aws_instance.foo[0]`: struct{}{},
	}

	for _, res := range plan.Changes.Resources {
		if res.Action != plans.Create {
			t.Fatalf("expected resource creation, got %s", res.Action)
		}
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		_, ok := expected[ric.Addr.String()]
		if !ok {
			t.Fatal("unexpected resource:", ric.Addr.String())
		}
		delete(expected, ric.Addr.String())
	}
	for addr := range expected {
		t.Error("missing resource", addr)
	}
}

// GH-1475
func TestContext2Plan_moduleCycle(t *testing.T) {
	m := testModule(t, "plan-module-cycle")
	p := testProvider("aws")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Computed: true},
					"some_input": {Type: cty.String, Optional: true},
					"type":       {Type: cty.String, Computed: true},
				},
			},
		},
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 2 {
		t.Fatal("expected 2 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		if res.Action != plans.Create {
			t.Fatalf("expected resource creation, got %s", res.Action)
		}
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		var expected cty.Value
		switch i := ric.Addr.String(); i {
		case "aws_instance.b":
			expected = objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"type": cty.UnknownVal(cty.String),
			})
		case "aws_instance.c":
			expected = objectVal(t, schema, map[string]cty.Value{
				"id":         cty.UnknownVal(cty.String),
				"some_input": cty.UnknownVal(cty.String),
				"type":       cty.UnknownVal(cty.String),
			})
		default:
			t.Fatal("unknown instance:", i)
		}

		checkVals(t, expected, ric.After)
	}
}

func TestContext2Plan_moduleDeadlock(t *testing.T) {
	testCheckDeadlock(t, func() {
		m := testModule(t, "plan-module-deadlock")
		p := testProvider("aws")
		p.PlanResourceChangeFn = testDiffFn

		ctx := testContext2(t, &ContextOpts{
			Config: m,
			Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
			},
		})

		plan, err := ctx.Plan()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
		ty := schema.ImpliedType()

		for _, res := range plan.Changes.Resources {
			if res.Action != plans.Create {
				t.Fatalf("expected resource creation, got %s", res.Action)
			}
			ric, err := res.Decode(ty)
			if err != nil {
				t.Fatal(err)
			}

			expected := objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"type": cty.UnknownVal(cty.String),
			})
			switch i := ric.Addr.String(); i {
			case "module.child.aws_instance.foo[0]":
			case "module.child.aws_instance.foo[1]":
			case "module.child.aws_instance.foo[2]":
			default:
				t.Fatal("unknown instance:", i)
			}

			checkVals(t, expected, ric.After)
		}
	})
}

func TestContext2Plan_moduleInput(t *testing.T) {
	m := testModule(t, "plan-module-input")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 2 {
		t.Fatal("expected 2 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		if res.Action != plans.Create {
			t.Fatalf("expected resource creation, got %s", res.Action)
		}
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		var expected cty.Value

		switch i := ric.Addr.String(); i {
		case "aws_instance.bar":
			expected = objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"foo":  cty.StringVal("2"),
				"type": cty.UnknownVal(cty.String),
			})
		case "module.child.aws_instance.foo":
			expected = objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"foo":  cty.StringVal("42"),
				"type": cty.UnknownVal(cty.String),
			})
		default:
			t.Fatal("unknown instance:", i)
		}

		checkVals(t, expected, ric.After)
	}
}

func TestContext2Plan_moduleInputComputed(t *testing.T) {
	m := testModule(t, "plan-module-input-computed")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 2 {
		t.Fatal("expected 2 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		if res.Action != plans.Create {
			t.Fatalf("expected resource creation, got %s", res.Action)
		}
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "aws_instance.bar":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":      cty.UnknownVal(cty.String),
				"foo":     cty.UnknownVal(cty.String),
				"type":    cty.UnknownVal(cty.String),
				"compute": cty.StringVal("foo"),
			}), ric.After)
		case "module.child.aws_instance.foo":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"foo":  cty.UnknownVal(cty.String),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		default:
			t.Fatal("unknown instance:", i)
		}
	}
}

func TestContext2Plan_moduleInputFromVar(t *testing.T) {
	m := testModule(t, "plan-module-input-var")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
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

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 2 {
		t.Fatal("expected 2 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		if res.Action != plans.Create {
			t.Fatalf("expected resource creation, got %s", res.Action)
		}
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "aws_instance.bar":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"foo":  cty.StringVal("2"),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		case "module.child.aws_instance.foo":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"foo":  cty.StringVal("52"),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		default:
			t.Fatal("unknown instance:", i)
		}
	}
}

func TestContext2Plan_moduleMultiVar(t *testing.T) {
	m := testModule(t, "plan-module-multi-var")
	p := testProvider("aws")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"id":  {Type: cty.String, Computed: true},
					"foo": {Type: cty.String, Optional: true},
					"baz": {Type: cty.String, Optional: true},
				},
			},
		},
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 5 {
		t.Fatal("expected 5 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		if res.Action != plans.Create {
			t.Fatalf("expected resource creation, got %s", res.Action)
		}

		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "aws_instance.parent[0]":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
			}), ric.After)
		case "aws_instance.parent[1]":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
			}), ric.After)
		case "module.child.aws_instance.bar[0]":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":  cty.UnknownVal(cty.String),
				"baz": cty.StringVal("baz"),
			}), ric.After)
		case "module.child.aws_instance.bar[1]":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":  cty.UnknownVal(cty.String),
				"baz": cty.StringVal("baz"),
			}), ric.After)
		case "module.child.aws_instance.foo":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":  cty.UnknownVal(cty.String),
				"foo": cty.StringVal("baz,baz"),
			}), ric.After)
		default:
			t.Fatal("unknown instance:", i)
		}
	}
}

func TestContext2Plan_moduleOrphans(t *testing.T) {
	m := testModule(t, "plan-modules-remove")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn

	state := states.NewState()
	child := state.EnsureModule(addrs.RootModuleInstance.Child("child", addrs.NoKey))
	child.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"baz"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		State: state,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}
	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 2 {
		t.Fatal("expected 2 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {

		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "aws_instance.foo":
			if res.Action != plans.Create {
				t.Fatalf("expected resource creation, got %s", res.Action)
			}
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"num":  cty.NumberIntVal(2),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		case "module.child.aws_instance.foo":
			if res.Action != plans.Delete {
				t.Fatalf("expected resource delete, got %s", res.Action)
			}
		default:
			t.Fatal("unknown instance:", i)
		}
	}

	expectedState := `<no state>
module.child:
  aws_instance.foo:
    ID = baz
    provider = provider["registry.terraform.io/hashicorp/aws"]`

	if ctx.State().String() != expectedState {
		t.Fatalf("\nexpected state: %q\n\ngot: %q", expectedState, ctx.State().String())
	}
}

// https://github.com/hashicorp/terraform/issues/3114
func TestContext2Plan_moduleOrphansWithProvisioner(t *testing.T) {
	m := testModule(t, "plan-modules-remove-provisioners")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn
	pr := testProvisioner()

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.top").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"top","type":"aws_instance"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	child1 := state.EnsureModule(addrs.RootModuleInstance.Child("parent", addrs.NoKey).Child("child1", addrs.NoKey))
	child1.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"baz","type":"aws_instance"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	child2 := state.EnsureModule(addrs.RootModuleInstance.Child("parent", addrs.NoKey).Child("child2", addrs.NoKey))
	child2.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"baz","type":"aws_instance"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		Provisioners: map[string]provisioners.Factory{
			"shell": testProvisionerFuncFixed(pr),
		},
		State: state,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 3 {
		t.Error("expected 3 planned resources, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {

		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "module.parent.module.child1.aws_instance.foo":
			if res.Action != plans.Delete {
				t.Fatalf("expected resource Delete, got %s", res.Action)
			}
		case "module.parent.module.child2.aws_instance.foo":
			if res.Action != plans.Delete {
				t.Fatalf("expected resource Delete, got %s", res.Action)
			}
		case "aws_instance.top":
			if res.Action != plans.NoOp {
				t.Fatalf("expected no changes, got %s", res.Action)
			}
		default:
			t.Fatalf("unknown instance: %s\nafter: %#v", i, hcl2shim.ConfigValueFromHCL2(ric.After))
		}
	}

	expectedState := `aws_instance.top:
  ID = top
  provider = provider["registry.terraform.io/hashicorp/aws"]
  type = aws_instance

module.parent.child1:
  aws_instance.foo:
    ID = baz
    provider = provider["registry.terraform.io/hashicorp/aws"]
    type = aws_instance
module.parent.child2:
  aws_instance.foo:
    ID = baz
    provider = provider["registry.terraform.io/hashicorp/aws"]
    type = aws_instance`

	if expectedState != ctx.State().String() {
		t.Fatalf("\nexpect state:\n%s\n\ngot state:\n%s\n", expectedState, ctx.State().String())
	}
}

func TestContext2Plan_moduleProviderInherit(t *testing.T) {
	var l sync.Mutex
	var calls []string

	m := testModule(t, "plan-module-provider-inherit")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): func() (providers.Interface, error) {
				l.Lock()
				defer l.Unlock()

				p := testProvider("aws")
				p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
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
				})
				p.ConfigureProviderFn = func(req providers.ConfigureProviderRequest) (resp providers.ConfigureProviderResponse) {
					from := req.Config.GetAttr("from")
					if from.IsNull() || from.AsString() != "root" {
						resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("not root"))
					}

					return
				}
				p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
					from := req.Config.GetAttr("from").AsString()

					l.Lock()
					defer l.Unlock()
					calls = append(calls, from)
					return testDiffFn(req)
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

// This tests (for GH-11282) that deeply nested modules properly inherit
// configuration.
func TestContext2Plan_moduleProviderInheritDeep(t *testing.T) {
	var l sync.Mutex

	m := testModule(t, "plan-module-provider-inherit-deep")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): func() (providers.Interface, error) {
				l.Lock()
				defer l.Unlock()

				var from string
				p := testProvider("aws")

				p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
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
				})

				p.ConfigureProviderFn = func(req providers.ConfigureProviderRequest) (resp providers.ConfigureProviderResponse) {
					v := req.Config.GetAttr("from")
					if v.IsNull() || v.AsString() != "root" {
						resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("not root"))
					}
					from = v.AsString()

					return
				}

				p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
					if from != "root" {
						resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("bad resource"))
						return
					}

					return testDiffFn(req)
				}
				return p, nil
			},
		},
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
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): func() (providers.Interface, error) {
				l.Lock()
				defer l.Unlock()

				p := testProvider("aws")
				p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
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
				})
				p.ConfigureProviderFn = func(req providers.ConfigureProviderRequest) (resp providers.ConfigureProviderResponse) {
					var buf bytes.Buffer
					from := req.Config.GetAttr("from")
					if !from.IsNull() {
						buf.WriteString(from.AsString() + "\n")
					}
					to := req.Config.GetAttr("to")
					if !to.IsNull() {
						buf.WriteString(to.AsString() + "\n")
					}

					l.Lock()
					defer l.Unlock()
					calls = append(calls, buf.String())
					return
				}

				return p, nil
			},
		},
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
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
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
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 1 {
		t.Fatal("expected 1 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		if res.Action != plans.Create {
			t.Fatalf("expected resource creation, got %s", res.Action)
		}
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "module.child.aws_instance.test":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"value": cty.StringVal("hello"),
			}), ric.After)
		default:
			t.Fatal("unknown instance:", i)
		}
	}
}

func TestContext2Plan_moduleVar(t *testing.T) {
	m := testModule(t, "plan-module-var")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 2 {
		t.Fatal("expected 2 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		if res.Action != plans.Create {
			t.Fatalf("expected resource creation, got %s", res.Action)
		}
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		var expected cty.Value

		switch i := ric.Addr.String(); i {
		case "aws_instance.bar":
			expected = objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"foo":  cty.StringVal("2"),
				"type": cty.UnknownVal(cty.String),
			})
		case "module.child.aws_instance.foo":
			expected = objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"num":  cty.NumberIntVal(2),
				"type": cty.UnknownVal(cty.String),
			})
		default:
			t.Fatal("unknown instance:", i)
		}

		checkVals(t, expected, ric.After)
	}
}

func TestContext2Plan_moduleVarWrongTypeBasic(t *testing.T) {
	m := testModule(t, "plan-module-wrong-var-type")
	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan()
	if !diags.HasErrors() {
		t.Fatalf("succeeded; want errors")
	}
}

func TestContext2Plan_moduleVarWrongTypeNested(t *testing.T) {
	m := testModule(t, "plan-module-wrong-var-type-nested")
	p := testProvider("null")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("null"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan()
	if !diags.HasErrors() {
		t.Fatalf("succeeded; want errors")
	}
}

func TestContext2Plan_moduleVarWithDefaultValue(t *testing.T) {
	m := testModule(t, "plan-module-var-with-default-value")
	p := testProvider("null")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("null"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}
}

func TestContext2Plan_moduleVarComputed(t *testing.T) {
	m := testModule(t, "plan-module-var-computed")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}
	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 2 {
		t.Fatal("expected 2 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		if res.Action != plans.Create {
			t.Fatalf("expected resource creation, got %s", res.Action)
		}
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "aws_instance.bar":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"foo":  cty.UnknownVal(cty.String),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		case "module.child.aws_instance.foo":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":      cty.UnknownVal(cty.String),
				"foo":     cty.UnknownVal(cty.String),
				"type":    cty.UnknownVal(cty.String),
				"compute": cty.StringVal("foo"),
			}), ric.After)
		default:
			t.Fatal("unknown instance:", i)
		}
	}
}

func TestContext2Plan_preventDestroy_bad(t *testing.T) {
	m := testModule(t, "plan-prevent-destroy-bad")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn
	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"i-abc123"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		State: state,
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
	p.PlanResourceChangeFn = testDiffFn

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"i-abc123","type":"aws_instance"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		State: state,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if !plan.Changes.Empty() {
		t.Fatalf("expected no changes, got %#v\n", plan.Changes)
	}
}

func TestContext2Plan_preventDestroy_countBad(t *testing.T) {
	m := testModule(t, "plan-prevent-destroy-count-bad")
	p := testProvider("aws")

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo[0]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"i-abc123"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo[1]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"i-abc345"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		State: state,
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
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"current": {Type: cty.String, Optional: true},
					"id":      {Type: cty.String, Computed: true},
				},
			},
		},
	})

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo[0]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"i-abc123"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo[1]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"i-abc345"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		State: state,
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
	p.PlanResourceChangeFn = testDiffFn
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"current": {Type: cty.String, Optional: true},
					"type":    {Type: cty.String, Optional: true, Computed: true},
					"id":      {Type: cty.String, Computed: true},
				},
			},
		},
	})

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo[0]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"i-abc123","current":"0","type":"aws_instance"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		State: state,
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

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"i-abc123"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		State:    state,
		PlanMode: plans.DestroyMode,
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
	pr := testProvisioner()
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		Provisioners: map[string]provisioners.Factory{
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
	p.PlanResourceChangeFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 2 {
		t.Fatal("expected 2 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		if res.Action != plans.Create {
			t.Fatalf("expected resource creation, got %s", res.Action)
		}
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "aws_instance.bar":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"foo":  cty.UnknownVal(cty.String),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		case "aws_instance.foo":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":      cty.UnknownVal(cty.String),
				"foo":     cty.UnknownVal(cty.String),
				"num":     cty.NumberIntVal(2),
				"type":    cty.UnknownVal(cty.String),
				"compute": cty.StringVal("foo"),
			}), ric.After)
		default:
			t.Fatal("unknown instance:", i)
		}
	}
}

func TestContext2Plan_blockNestingGroup(t *testing.T) {
	m := testModule(t, "plan-block-nesting-group")
	p := testProvider("test")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test": {
				BlockTypes: map[string]*configschema.NestedBlock{
					"blah": {
						Nesting: configschema.NestingGroup,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"baz": {Type: cty.String, Required: true},
							},
						},
					},
				},
			},
		},
	})
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		return providers.PlanResourceChangeResponse{
			PlannedState: req.ProposedNewState,
		}
	}
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if got, want := 1, len(plan.Changes.Resources); got != want {
		t.Fatalf("wrong number of planned resource changes %d; want %d\n%s", got, want, spew.Sdump(plan.Changes.Resources))
	}

	if !p.PlanResourceChangeCalled {
		t.Fatalf("PlanResourceChange was not called at all")
	}

	got := p.PlanResourceChangeRequest
	want := providers.PlanResourceChangeRequest{
		TypeName: "test",

		// Because block type "blah" is defined as NestingGroup, we get a non-null
		// value for it with null nested attributes, rather than the "blah" object
		// itself being null, when there's no "blah" block in the config at all.
		//
		// This represents the situation where the remote service _always_ creates
		// a single "blah", regardless of whether the block is present, but when
		// the block _is_ present the user can override some aspects of it. The
		// absense of the block means "use the defaults", in that case.
		Config: cty.ObjectVal(map[string]cty.Value{
			"blah": cty.ObjectVal(map[string]cty.Value{
				"baz": cty.NullVal(cty.String),
			}),
		}),
		ProposedNewState: cty.ObjectVal(map[string]cty.Value{
			"blah": cty.ObjectVal(map[string]cty.Value{
				"baz": cty.NullVal(cty.String),
			}),
		}),
	}
	if !cmp.Equal(got, want, valueTrans) {
		t.Errorf("wrong PlanResourceChange request\n%s", cmp.Diff(got, want, valueTrans))
	}
}

func TestContext2Plan_computedDataResource(t *testing.T) {
	m := testModule(t, "plan-computed-data-resource")
	p := testProvider("aws")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
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
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}
	schema := p.GetProviderSchemaResponse.DataSources["aws_vpc"].Block
	ty := schema.ImpliedType()

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

	rc, err := rcs.Decode(ty)
	if err != nil {
		t.Fatal(err)
	}

	checkVals(t,
		cty.ObjectVal(map[string]cty.Value{
			"foo": cty.UnknownVal(cty.String),
		}),
		rc.After,
	)
}

func TestContext2Plan_computedInFunction(t *testing.T) {
	m := testModule(t, "plan-computed-in-function")
	p := testProvider("aws")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"attr": {Type: cty.Number, Optional: true},
				},
			},
		},
		DataSources: map[string]*configschema.Block{
			"aws_data_source": {
				Attributes: map[string]*configschema.Attribute{
					"computed": {Type: cty.List(cty.String), Computed: true},
				},
			},
		},
	})
	p.ReadDataSourceResponse = &providers.ReadDataSourceResponse{
		State: cty.ObjectVal(map[string]cty.Value{
			"computed": cty.ListVal([]cty.Value{
				cty.StringVal("foo"),
			}),
		}),
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate()
	assertNoErrors(t, diags)

	_, diags = ctx.Plan()
	assertNoErrors(t, diags)

	if !p.ReadDataSourceCalled {
		t.Fatalf("ReadDataSource was not called on provider during plan; should've been called")
	}
}

func TestContext2Plan_computedDataCountResource(t *testing.T) {
	m := testModule(t, "plan-computed-data-count")
	p := testProvider("aws")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
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
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
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
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
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

func TestContext2Plan_dataResourceBecomesComputed(t *testing.T) {
	m := testModule(t, "plan-data-resource-becomes-computed")
	p := testProvider("aws")

	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo":      {Type: cty.String, Optional: true},
					"computed": {Type: cty.String, Computed: true},
				},
			},
		},
		DataSources: map[string]*configschema.Block{
			"aws_data_source": {
				Attributes: map[string]*configschema.Attribute{
					"id":  {Type: cty.String, Computed: true},
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
	})

	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		fooVal := req.ProposedNewState.GetAttr("foo")
		return providers.PlanResourceChangeResponse{
			PlannedState: cty.ObjectVal(map[string]cty.Value{
				"foo":      fooVal,
				"computed": cty.UnknownVal(cty.String),
			}),
			PlannedPrivate: req.PriorPrivate,
		}
	}

	schema := p.GetProviderSchemaResponse.DataSources["aws_data_source"].Block
	ty := schema.ImpliedType()

	p.ReadDataSourceResponse = &providers.ReadDataSourceResponse{
		// This should not be called, because the configuration for the
		// data resource contains an unknown value for "foo".
		Diagnostics: tfdiags.Diagnostics(nil).Append(fmt.Errorf("ReadDataSource called, but should not have been")),
	}

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("data.aws_data_source.foo").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"i-abc123","foo":"baz"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		State: state,
	})

	_, diags := ctx.Refresh()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors during refresh: %s", diags.Err())
	}

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors during plan: %s", diags.Err())
	}

	rcs := plan.Changes.ResourceInstance(addrs.Resource{
		Mode: addrs.DataResourceMode,
		Type: "aws_data_source",
		Name: "foo",
	}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance))
	if rcs == nil {
		t.Logf("full changeset: %s", spew.Sdump(plan.Changes))
		t.Fatalf("missing diff for data.aws_data_resource.foo")
	}

	rc, err := rcs.Decode(ty)
	if err != nil {
		t.Fatal(err)
	}

	// foo should now be unknown
	foo := rc.After.GetAttr("foo")
	if foo.IsKnown() {
		t.Fatalf("foo should be unknown, got %#v", foo)
	}
}

func TestContext2Plan_computedList(t *testing.T) {
	m := testModule(t, "plan-computed-list")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
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
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 2 {
		t.Fatal("expected 2 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		if res.Action != plans.Create {
			t.Fatalf("expected resource creation, got %s", res.Action)
		}
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "aws_instance.bar":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"foo": cty.UnknownVal(cty.String),
			}), ric.After)
		case "aws_instance.foo":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"list":    cty.UnknownVal(cty.List(cty.String)),
				"num":     cty.NumberIntVal(2),
				"compute": cty.StringVal("list.#"),
			}), ric.After)
		default:
			t.Fatal("unknown instance:", i)
		}
	}
}

// GH-8695. This tests that you can index into a computed list on a
// splatted resource.
func TestContext2Plan_computedMultiIndex(t *testing.T) {
	m := testModule(t, "plan-computed-multi-index")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn

	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"compute": {Type: cty.String, Optional: true},
					"foo":     {Type: cty.List(cty.String), Optional: true},
					"ip":      {Type: cty.List(cty.String), Computed: true},
				},
			},
		},
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 3 {
		t.Fatal("expected 3 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		if res.Action != plans.Create {
			t.Fatalf("expected resource creation, got %s", res.Action)
		}
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "aws_instance.foo[0]":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"ip":      cty.UnknownVal(cty.List(cty.String)),
				"foo":     cty.NullVal(cty.List(cty.String)),
				"compute": cty.StringVal("ip.#"),
			}), ric.After)
		case "aws_instance.foo[1]":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"ip":      cty.UnknownVal(cty.List(cty.String)),
				"foo":     cty.NullVal(cty.List(cty.String)),
				"compute": cty.StringVal("ip.#"),
			}), ric.After)
		case "aws_instance.bar[0]":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"foo": cty.UnknownVal(cty.List(cty.String)),
			}), ric.After)
		default:
			t.Fatal("unknown instance:", i)
		}
	}
}

func TestContext2Plan_count(t *testing.T) {
	m := testModule(t, "plan-count")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 6 {
		t.Fatal("expected 6 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		if res.Action != plans.Create {
			t.Fatalf("expected resource creation, got %s", res.Action)
		}
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "aws_instance.bar":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"foo":  cty.StringVal("foo,foo,foo,foo,foo"),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		case "aws_instance.foo[0]":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"foo":  cty.StringVal("foo"),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		case "aws_instance.foo[1]":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"foo":  cty.StringVal("foo"),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		case "aws_instance.foo[2]":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"foo":  cty.StringVal("foo"),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		case "aws_instance.foo[3]":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"foo":  cty.StringVal("foo"),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		case "aws_instance.foo[4]":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"foo":  cty.StringVal("foo"),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		default:
			t.Fatal("unknown instance:", i)
		}
	}
}

func TestContext2Plan_countComputed(t *testing.T) {
	m := testModule(t, "plan-count-computed")
	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	_, err := ctx.Plan()
	if err == nil {
		t.Fatal("should error")
	}
}

func TestContext2Plan_countComputedModule(t *testing.T) {
	m := testModule(t, "plan-count-computed-module")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
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
	p.PlanResourceChangeFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 3 {
		t.Fatal("expected 3 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		if res.Action != plans.Create {
			t.Fatalf("expected resource creation, got %s", res.Action)
		}
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "module.child.aws_instance.foo[0]":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		case "module.child.aws_instance.foo[1]":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		case "module.child.aws_instance.foo[2]":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		default:
			t.Fatal("unknown instance:", i)
		}
	}
}

func TestContext2Plan_countModuleStaticGrandchild(t *testing.T) {
	m := testModule(t, "plan-count-module-static-grandchild")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 3 {
		t.Fatal("expected 3 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		if res.Action != plans.Create {
			t.Fatalf("expected resource creation, got %s", res.Action)
		}
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "module.child.module.child.aws_instance.foo[0]":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		case "module.child.module.child.aws_instance.foo[1]":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		case "module.child.module.child.aws_instance.foo[2]":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		default:
			t.Fatal("unknown instance:", i)
		}
	}
}

func TestContext2Plan_countIndex(t *testing.T) {
	m := testModule(t, "plan-count-index")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 2 {
		t.Fatal("expected 2 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		if res.Action != plans.Create {
			t.Fatalf("expected resource creation, got %s", res.Action)
		}
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "aws_instance.foo[0]":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"foo":  cty.StringVal("0"),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		case "aws_instance.foo[1]":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"foo":  cty.StringVal("1"),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		default:
			t.Fatal("unknown instance:", i)
		}
	}
}

func TestContext2Plan_countVar(t *testing.T) {
	m := testModule(t, "plan-count-var")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
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
	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 4 {
		t.Fatal("expected 4 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		if res.Action != plans.Create {
			t.Fatalf("expected resource creation, got %s", res.Action)
		}
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "aws_instance.bar":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"foo":  cty.StringVal("foo,foo,foo"),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		case "aws_instance.foo[0]":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"foo":  cty.StringVal("foo"),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		case "aws_instance.foo[1]":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"foo":  cty.StringVal("foo"),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		case "aws_instance.foo[2]":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"foo":  cty.StringVal("foo"),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		default:
			t.Fatal("unknown instance:", i)
		}
	}
}

func TestContext2Plan_countZero(t *testing.T) {
	m := testModule(t, "plan-count-zero")
	p := testProvider("aws")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.DynamicPseudoType, Optional: true},
				},
			},
		},
	})

	// This schema contains a DynamicPseudoType, and therefore can't go through any shim functions
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		resp.PlannedState = req.ProposedNewState
		resp.PlannedPrivate = req.PriorPrivate
		return resp
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}
	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 1 {
		t.Fatal("expected 1 changes, got", len(plan.Changes.Resources))
	}

	res := plan.Changes.Resources[0]

	if res.Action != plans.Create {
		t.Fatalf("expected resource creation, got %s", res.Action)
	}
	ric, err := res.Decode(ty)
	if err != nil {
		t.Fatal(err)
	}

	expected := cty.TupleVal(nil)

	foo := ric.After.GetAttr("foo")

	if !cmp.Equal(expected, foo, valueComparer) {
		t.Fatal(cmp.Diff(expected, foo, valueComparer))
	}
}

func TestContext2Plan_countOneIndex(t *testing.T) {
	m := testModule(t, "plan-count-one-index")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 2 {
		t.Fatal("expected 2 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		if res.Action != plans.Create {
			t.Fatalf("expected resource creation, got %s", res.Action)
		}
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "aws_instance.bar":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"foo":  cty.StringVal("foo"),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		case "aws_instance.foo[0]":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"foo":  cty.StringVal("foo"),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		default:
			t.Fatal("unknown instance:", i)
		}
	}
}

func TestContext2Plan_countDecreaseToOne(t *testing.T) {
	m := testModule(t, "plan-count-dec")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo[0]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"bar","foo":"foo","type":"aws_instance"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo[1]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"bar"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo[2]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"bar"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		State: state,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}
	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 4 {
		t.Fatal("expected 4 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {

		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "aws_instance.bar":
			if res.Action != plans.Create {
				t.Fatalf("expected resource create, got %s", res.Action)
			}
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"foo":  cty.StringVal("bar"),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		case "aws_instance.foo":
			if res.Action != plans.NoOp {
				t.Fatalf("resource %s should be unchanged", i)
			}
		case "aws_instance.foo[1]":
			if res.Action != plans.Delete {
				t.Fatalf("expected resource delete, got %s", res.Action)
			}
		case "aws_instance.foo[2]":
			if res.Action != plans.Delete {
				t.Fatalf("expected resource delete, got %s", res.Action)
			}
		default:
			t.Fatal("unknown instance:", i)
		}
	}

	expectedState := `aws_instance.foo:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = foo
  type = aws_instance
aws_instance.foo.1:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/aws"]
aws_instance.foo.2:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/aws"]`

	if ctx.State().String() != expectedState {
		t.Fatalf("epected state:\n%q\n\ngot state:\n%q\n", expectedState, ctx.State().String())
	}
}

func TestContext2Plan_countIncreaseFromNotSet(t *testing.T) {
	m := testModule(t, "plan-count-inc")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"bar","type":"aws_instance","foo":"foo"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		State: state,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}
	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 4 {
		t.Fatal("expected 4 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {

		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "aws_instance.bar":
			if res.Action != plans.Create {
				t.Fatalf("expected resource create, got %s", res.Action)
			}
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"foo":  cty.StringVal("bar"),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		case "aws_instance.foo[0]":
			if res.Action != plans.NoOp {
				t.Fatalf("resource %s should be unchanged", i)
			}
		case "aws_instance.foo[1]":
			if res.Action != plans.Create {
				t.Fatalf("expected resource create, got %s", res.Action)
			}
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"foo":  cty.StringVal("foo"),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		case "aws_instance.foo[2]":
			if res.Action != plans.Create {
				t.Fatalf("expected resource create, got %s", res.Action)
			}
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"foo":  cty.StringVal("foo"),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		default:
			t.Fatal("unknown instance:", i)
		}
	}
}

func TestContext2Plan_countIncreaseFromOne(t *testing.T) {
	m := testModule(t, "plan-count-inc")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn
	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo[0]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"bar","foo":"foo","type":"aws_instance"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		State: state,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}
	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 4 {
		t.Fatal("expected 4 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {

		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "aws_instance.bar":
			if res.Action != plans.Create {
				t.Fatalf("expected resource create, got %s", res.Action)
			}
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"foo":  cty.StringVal("bar"),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		case "aws_instance.foo[0]":
			if res.Action != plans.NoOp {
				t.Fatalf("resource %s should be unchanged", i)
			}
		case "aws_instance.foo[1]":
			if res.Action != plans.Create {
				t.Fatalf("expected resource create, got %s", res.Action)
			}
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"foo":  cty.StringVal("foo"),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		case "aws_instance.foo[2]":
			if res.Action != plans.Create {
				t.Fatalf("expected resource create, got %s", res.Action)
			}
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"foo":  cty.StringVal("foo"),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		default:
			t.Fatal("unknown instance:", i)
		}
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
	p.PlanResourceChangeFn = testDiffFn

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"bar","foo":"foo","type":"aws_instance"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo[0]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"bar","foo":"foo","type":"aws_instance"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		State: state,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}
	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 5 {
		t.Fatal("expected 5 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {

		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "aws_instance.bar":
			if res.Action != plans.Create {
				t.Fatalf("expected resource create, got %s", res.Action)
			}
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"foo":  cty.StringVal("bar"),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		case "aws_instance.foo":
			if res.Action != plans.Delete {
				t.Fatalf("resource %s should be removed", i)
			}
		case "aws_instance.foo[0]":
			if res.Action != plans.NoOp {
				t.Fatalf("resource %s should be unchanged", i)
			}
		case "aws_instance.foo[1]":
			if res.Action != plans.Create {
				t.Fatalf("expected resource create, got %s", res.Action)
			}
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"foo":  cty.StringVal("foo"),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		case "aws_instance.foo[2]":
			if res.Action != plans.Create {
				t.Fatalf("expected resource create, got %s", res.Action)
			}
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"foo":  cty.StringVal("foo"),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		default:
			t.Fatal("unknown instance:", i)
		}
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
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"name":     {Type: cty.String, Optional: true},
					"foo_name": {Type: cty.String, Optional: true},
					"id":       {Type: cty.String, Computed: true},
				},
			},
		},
	})

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo[0]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"bar","name":"foo 0"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo[1]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"bar","name":"foo 1"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.bar[0]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"bar","foo_name":"foo 0"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.bar[1]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"bar","foo_name":"foo 1"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		State: state,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 6 {
		t.Fatal("expected 6 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "aws_instance.bar[0]", "aws_instance.bar[1]", "aws_instance.foo[0]", "aws_instance.foo[1]":
			if res.Action != plans.NoOp {
				t.Fatalf("resource %s should be unchanged", i)
			}
		case "aws_instance.bar[2]":
			if res.Action != plans.Create {
				t.Fatalf("expected resource create, got %s", res.Action)
			}
			// The instance ID changed, so just check that the name updated
			if ric.After.GetAttr("foo_name") != cty.StringVal("foo 2") {
				t.Fatalf("resource %s attr \"foo_name\" should be changed", i)
			}
		case "aws_instance.foo[2]":
			if res.Action != plans.Create {
				t.Fatalf("expected resource create, got %s", res.Action)
			}
			// The instance ID changed, so just check that the name updated
			if ric.After.GetAttr("name") != cty.StringVal("foo 2") {
				t.Fatalf("resource %s attr \"name\" should be changed", i)
			}
		default:
			t.Fatal("unknown instance:", i)
		}
	}
}

func TestContext2Plan_forEach(t *testing.T) {
	m := testModule(t, "plan-for-each")
	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 8 {
		t.Fatal("expected 8 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		if res.Action != plans.Create {
			t.Fatalf("expected resource creation, got %s", res.Action)
		}
		_, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestContext2Plan_forEachUnknownValue(t *testing.T) {
	// This module has a variable defined, but it's value is unknown. We
	// expect this to produce an error, but not to panic.
	m := testModule(t, "plan-for-each-unknown-value")
	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		Variables: InputValues{
			"foo": {
				Value:      cty.UnknownVal(cty.String),
				SourceType: ValueFromCLIArg,
			},
		},
	})

	_, diags := ctx.Plan()
	if !diags.HasErrors() {
		// Should get this error:
		// Invalid for_each argument: The "for_each" value depends on resource attributes that cannot be determined until apply...
		t.Fatal("succeeded; want errors")
	}

	gotErrStr := diags.Err().Error()
	wantErrStr := "Invalid for_each argument"
	if !strings.Contains(gotErrStr, wantErrStr) {
		t.Fatalf("missing expected error\ngot: %s\n\nwant: error containing %q", gotErrStr, wantErrStr)
	}
}

func TestContext2Plan_destroy(t *testing.T) {
	m := testModule(t, "plan-destroy")
	p := testProvider("aws")

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.one").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"bar"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.two").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"baz"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		State:    state,
		PlanMode: plans.DestroyMode,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 2 {
		t.Fatal("expected 2 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "aws_instance.one", "aws_instance.two":
			if res.Action != plans.Delete {
				t.Fatalf("resource %s should be removed", i)
			}

		default:
			t.Fatal("unknown instance:", i)
		}
	}
}

func TestContext2Plan_moduleDestroy(t *testing.T) {
	m := testModule(t, "plan-module-destroy")
	p := testProvider("aws")

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"bar"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	child := state.EnsureModule(addrs.RootModuleInstance.Child("child", addrs.NoKey))
	child.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"bar"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		State:    state,
		PlanMode: plans.DestroyMode,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}
	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 2 {
		t.Fatal("expected 2 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "aws_instance.foo", "module.child.aws_instance.foo":
			if res.Action != plans.Delete {
				t.Fatalf("resource %s should be removed", i)
			}

		default:
			t.Fatal("unknown instance:", i)
		}
	}
}

// GH-1835
func TestContext2Plan_moduleDestroyCycle(t *testing.T) {
	m := testModule(t, "plan-module-destroy-gh-1835")
	p := testProvider("aws")

	state := states.NewState()
	aModule := state.EnsureModule(addrs.RootModuleInstance.Child("a_module", addrs.NoKey))
	aModule.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.a").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"a"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	bModule := state.EnsureModule(addrs.RootModuleInstance.Child("b_module", addrs.NoKey))
	bModule.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.b").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"b"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		State:    state,
		PlanMode: plans.DestroyMode,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 2 {
		t.Fatal("expected 2 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "module.a_module.aws_instance.a", "module.b_module.aws_instance.b":
			if res.Action != plans.Delete {
				t.Fatalf("resource %s should be removed", i)
			}

		default:
			t.Fatal("unknown instance:", i)
		}
	}
}

func TestContext2Plan_moduleDestroyMultivar(t *testing.T) {
	m := testModule(t, "plan-module-destroy-multivar")
	p := testProvider("aws")

	state := states.NewState()
	child := state.EnsureModule(addrs.RootModuleInstance.Child("child", addrs.NoKey))
	child.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo[0]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"bar0"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	child.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo[1]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"bar1"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		State:    state,
		PlanMode: plans.DestroyMode,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 2 {
		t.Fatal("expected 2 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "module.child.aws_instance.foo[0]", "module.child.aws_instance.foo[1]":
			if res.Action != plans.Delete {
				t.Fatalf("resource %s should be removed", i)
			}

		default:
			t.Fatal("unknown instance:", i)
		}
	}
}

func TestContext2Plan_pathVar(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	m := testModule(t, "plan-path-var")
	p := testProvider("aws")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"cwd":    {Type: cty.String, Optional: true},
					"module": {Type: cty.String, Optional: true},
					"root":   {Type: cty.String, Optional: true},
				},
			},
		},
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("err: %s", diags.Err())
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 1 {
		t.Fatal("expected 1 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "aws_instance.foo":
			if res.Action != plans.Create {
				t.Fatalf("resource %s should be created", i)
			}
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"cwd":    cty.StringVal(cwd + "/barpath"),
				"module": cty.StringVal(m.Module.SourceDir + "/foopath"),
				"root":   cty.StringVal(m.Module.SourceDir + "/barpath"),
			}), ric.After)
		default:
			t.Fatal("unknown instance:", i)
		}
	}
}

func TestContext2Plan_diffVar(t *testing.T) {
	m := testModule(t, "plan-diffvar")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn
	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"bar","num":"2","type":"aws_instance"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		State: state,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 2 {
		t.Fatal("expected 2 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "aws_instance.bar":
			if res.Action != plans.Create {
				t.Fatalf("resource %s should be created", i)
			}
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"num":  cty.NumberIntVal(3),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		case "aws_instance.foo":
			if res.Action != plans.Update {
				t.Fatalf("resource %s should be updated", i)
			}
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.StringVal("bar"),
				"num":  cty.NumberIntVal(2),
				"type": cty.StringVal("aws_instance"),
			}), ric.Before)
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.StringVal("bar"),
				"num":  cty.NumberIntVal(3),
				"type": cty.StringVal("aws_instance"),
			}), ric.After)
		default:
			t.Fatal("unknown instance:", i)
		}
	}
}

func TestContext2Plan_hook(t *testing.T) {
	m := testModule(t, "plan-good")
	h := new(MockHook)
	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Hooks:  []Hook{h},
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
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
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
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
	p.PlanResourceChangeFn = testDiffFn
	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.baz").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"bar"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		State: state,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 2 {
		t.Fatal("expected 2 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "aws_instance.baz":
			if res.Action != plans.Delete {
				t.Fatalf("resource %s should be removed", i)
			}
			if got, want := ric.ActionReason, plans.ResourceInstanceChangeNoReason; got != want {
				t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
			}
		case "aws_instance.foo":
			if res.Action != plans.Create {
				t.Fatalf("resource %s should be created", i)
			}
			if got, want := ric.ActionReason, plans.ResourceInstanceChangeNoReason; got != want {
				t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
			}
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"num":  cty.NumberIntVal(2),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		default:
			t.Fatal("unknown instance:", i)
		}
	}
}

// This tests that configurations with UUIDs don't produce errors.
// For shadows, this would produce errors since a UUID changes every time.
func TestContext2Plan_shadowUuid(t *testing.T) {
	m := testModule(t, "plan-shadow-uuid")
	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}
}

func TestContext2Plan_state(t *testing.T) {
	m := testModule(t, "plan-good")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn
	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"bar"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		State: state,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if len(plan.Changes.Resources) < 2 {
		t.Fatalf("bad: %#v", plan.Changes.Resources)
	}
	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 2 {
		t.Fatal("expected 2 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "aws_instance.bar":
			if res.Action != plans.Create {
				t.Fatalf("resource %s should be created", i)
			}
			if got, want := ric.ActionReason, plans.ResourceInstanceChangeNoReason; got != want {
				t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
			}
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"foo":  cty.StringVal("2"),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		case "aws_instance.foo":
			if res.Action != plans.Update {
				t.Fatalf("resource %s should be updated", i)
			}
			if got, want := ric.ActionReason, plans.ResourceInstanceChangeNoReason; got != want {
				t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
			}
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.StringVal("bar"),
				"num":  cty.NullVal(cty.Number),
				"type": cty.NullVal(cty.String),
			}), ric.Before)
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.StringVal("bar"),
				"num":  cty.NumberIntVal(2),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		default:
			t.Fatal("unknown instance:", i)
		}
	}
}

func TestContext2Plan_requiresReplace(t *testing.T) {
	m := testModule(t, "plan-requires-replace")
	p := testProvider("test")
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{
			Block: &configschema.Block{},
		},
		ResourceTypes: map[string]providers.Schema{
			"test_thing": providers.Schema{
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"v": {
							Type:     cty.String,
							Required: true,
						},
					},
				},
			},
		},
	}
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		return providers.PlanResourceChangeResponse{
			PlannedState: req.ProposedNewState,
			RequiresReplace: []cty.Path{
				cty.GetAttrPath("v"),
			},
		}
	}

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_thing.foo").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"v":"hello"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
		State: state,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["test_thing"].Block
	ty := schema.ImpliedType()

	if got, want := len(plan.Changes.Resources), 1; got != want {
		t.Fatalf("got %d changes; want %d", got, want)
	}

	for _, res := range plan.Changes.Resources {
		t.Run(res.Addr.String(), func(t *testing.T) {
			ric, err := res.Decode(ty)
			if err != nil {
				t.Fatal(err)
			}

			switch i := ric.Addr.String(); i {
			case "test_thing.foo":
				if got, want := ric.Action, plans.DeleteThenCreate; got != want {
					t.Errorf("wrong action\ngot:  %s\nwant: %s", got, want)
				}
				if got, want := ric.ActionReason, plans.ResourceInstanceReplaceBecauseCannotUpdate; got != want {
					t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
				}
				checkVals(t, objectVal(t, schema, map[string]cty.Value{
					"v": cty.StringVal("goodbye"),
				}), ric.After)
			default:
				t.Fatalf("unexpected resource instance %s", i)
			}
		})
	}
}

func TestContext2Plan_taint(t *testing.T) {
	m := testModule(t, "plan-taint")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn
	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"bar","num":"2","type":"aws_instance"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.bar").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectTainted,
			AttrsJSON: []byte(`{"id":"baz"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		State: state,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 2 {
		t.Fatal("expected 2 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		t.Run(res.Addr.String(), func(t *testing.T) {
			ric, err := res.Decode(ty)
			if err != nil {
				t.Fatal(err)
			}

			switch i := ric.Addr.String(); i {
			case "aws_instance.bar":
				if got, want := res.Action, plans.DeleteThenCreate; got != want {
					t.Errorf("wrong action\ngot:  %s\nwant: %s", got, want)
				}
				if got, want := res.ActionReason, plans.ResourceInstanceReplaceBecauseTainted; got != want {
					t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
				}
				checkVals(t, objectVal(t, schema, map[string]cty.Value{
					"id":   cty.UnknownVal(cty.String),
					"foo":  cty.StringVal("2"),
					"type": cty.UnknownVal(cty.String),
				}), ric.After)
			case "aws_instance.foo":
				if got, want := res.Action, plans.NoOp; got != want {
					t.Errorf("wrong action\ngot:  %s\nwant: %s", got, want)
				}
				if got, want := res.ActionReason, plans.ResourceInstanceChangeNoReason; got != want {
					t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
				}
			default:
				t.Fatal("unknown instance:", i)
			}
		})
	}
}

func TestContext2Plan_taintIgnoreChanges(t *testing.T) {
	m := testModule(t, "plan-taint-ignore-changes")
	p := testProvider("aws")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"id":   {Type: cty.String, Computed: true},
					"vars": {Type: cty.String, Optional: true},
					"type": {Type: cty.String, Computed: true},
				},
			},
		},
	})

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectTainted,
			AttrsJSON: []byte(`{"id":"foo","vars":"foo","type":"aws_instance"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		State: state,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 1 {
		t.Fatal("expected 1 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "aws_instance.foo":
			if got, want := res.Action, plans.DeleteThenCreate; got != want {
				t.Errorf("wrong action\ngot:  %s\nwant: %s", got, want)
			}
			if got, want := res.ActionReason, plans.ResourceInstanceReplaceBecauseTainted; got != want {
				t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
			}
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.StringVal("foo"),
				"vars": cty.StringVal("foo"),
				"type": cty.StringVal("aws_instance"),
			}), ric.Before)
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"vars": cty.StringVal("foo"),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		default:
			t.Fatal("unknown instance:", i)
		}
	}
}

// Fails about 50% of the time before the fix for GH-4982, covers the fix.
func TestContext2Plan_taintDestroyInterpolatedCountRace(t *testing.T) {
	m := testModule(t, "plan-taint-interpolated-count")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn
	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo[0]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectTainted,
			AttrsJSON: []byte(`{"id":"bar","type":"aws_instance"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo[1]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"bar","type":"aws_instance"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo[2]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"bar","type":"aws_instance"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	for i := 0; i < 100; i++ {
		ctx := testContext2(t, &ContextOpts{
			Config: m,
			Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
			},
			State: state.DeepCopy(),
		})

		plan, diags := ctx.Plan()
		if diags.HasErrors() {
			t.Fatalf("unexpected errors: %s", diags.Err())
		}

		schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
		ty := schema.ImpliedType()

		if len(plan.Changes.Resources) != 3 {
			t.Fatal("expected 3 changes, got", len(plan.Changes.Resources))
		}

		for _, res := range plan.Changes.Resources {
			ric, err := res.Decode(ty)
			if err != nil {
				t.Fatal(err)
			}

			switch i := ric.Addr.String(); i {
			case "aws_instance.foo[0]":
				if got, want := ric.Action, plans.DeleteThenCreate; got != want {
					t.Errorf("wrong action\ngot:  %s\nwant: %s", got, want)
				}
				if got, want := ric.ActionReason, plans.ResourceInstanceReplaceBecauseTainted; got != want {
					t.Errorf("wrong action reason\ngot:  %s\nwant: %s", got, want)
				}
				checkVals(t, objectVal(t, schema, map[string]cty.Value{
					"id":   cty.StringVal("bar"),
					"type": cty.StringVal("aws_instance"),
				}), ric.Before)
				checkVals(t, objectVal(t, schema, map[string]cty.Value{
					"id":   cty.UnknownVal(cty.String),
					"type": cty.UnknownVal(cty.String),
				}), ric.After)
			case "aws_instance.foo[1]", "aws_instance.foo[2]":
				if res.Action != plans.NoOp {
					t.Fatalf("resource %s should not be changed", i)
				}
			default:
				t.Fatal("unknown instance:", i)
			}
		}
	}
}

func TestContext2Plan_targeted(t *testing.T) {
	m := testModule(t, "plan-targeted")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
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
	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 1 {
		t.Fatal("expected 1 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "aws_instance.foo":
			if res.Action != plans.Create {
				t.Fatalf("resource %s should be created", i)
			}
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"num":  cty.NumberIntVal(2),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		default:
			t.Fatal("unknown instance:", i)
		}
	}
}

// Test that targeting a module properly plans any inputs that depend
// on another module.
func TestContext2Plan_targetedCrossModule(t *testing.T) {
	m := testModule(t, "plan-targeted-cross-module")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Child("B", addrs.NoKey),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}
	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 2 {
		t.Fatal("expected 2 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}
		if res.Action != plans.Create {
			t.Fatalf("resource %s should be created", ric.Addr)
		}
		switch i := ric.Addr.String(); i {
		case "module.A.aws_instance.foo":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"foo":  cty.StringVal("bar"),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		case "module.B.aws_instance.bar":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"foo":  cty.UnknownVal(cty.String),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		default:
			t.Fatal("unknown instance:", i)
		}
	}
}

func TestContext2Plan_targetedModuleWithProvider(t *testing.T) {
	m := testModule(t, "plan-targeted-module-with-provider")
	p := testProvider("null")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
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
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("null"): testProviderFuncFixed(p),
		},
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Child("child2", addrs.NoKey),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["null_resource"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 1 {
		t.Fatal("expected 1 changes, got", len(plan.Changes.Resources))
	}

	res := plan.Changes.Resources[0]
	ric, err := res.Decode(ty)
	if err != nil {
		t.Fatal(err)
	}

	if ric.Addr.String() != "module.child2.null_resource.foo" {
		t.Fatalf("unexpcetd resource: %s", ric.Addr)
	}
}

func TestContext2Plan_targetedOrphan(t *testing.T) {
	m := testModule(t, "plan-targeted-orphan")
	p := testProvider("aws")

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.orphan").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"i-789xyz"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.nottargeted").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"i-abc123"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		State:    state,
		PlanMode: plans.DestroyMode,
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

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 1 {
		t.Fatal("expected 1 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "aws_instance.orphan":
			if res.Action != plans.Delete {
				t.Fatalf("resource %s should be destroyed", ric.Addr)
			}
		default:
			t.Fatal("unknown instance:", i)
		}
	}
}

// https://github.com/hashicorp/terraform/issues/2538
func TestContext2Plan_targetedModuleOrphan(t *testing.T) {
	m := testModule(t, "plan-targeted-module-orphan")
	p := testProvider("aws")

	state := states.NewState()
	child := state.EnsureModule(addrs.RootModuleInstance.Child("child", addrs.NoKey))
	child.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.orphan").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"i-789xyz"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	child.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.nottargeted").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"i-abc123"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		State:    state,
		PlanMode: plans.DestroyMode,
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

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 1 {
		t.Fatal("expected 1 changes, got", len(plan.Changes.Resources))
	}

	res := plan.Changes.Resources[0]
	ric, err := res.Decode(ty)
	if err != nil {
		t.Fatal(err)
	}

	if ric.Addr.String() != "module.child.aws_instance.orphan" {
		t.Fatalf("unexpected resource :%s", ric.Addr)
	}
	if res.Action != plans.Delete {
		t.Fatalf("resource %s should be deleted", ric.Addr)
	}
}

func TestContext2Plan_targetedModuleUntargetedVariable(t *testing.T) {
	m := testModule(t, "plan-targeted-module-untargeted-variable")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
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

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 2 {
		t.Fatal("expected 2 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}
		if res.Action != plans.Create {
			t.Fatalf("resource %s should be created", ric.Addr)
		}
		switch i := ric.Addr.String(); i {
		case "aws_instance.blue":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		case "module.blue_mod.aws_instance.mod":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":    cty.UnknownVal(cty.String),
				"value": cty.UnknownVal(cty.String),
				"type":  cty.UnknownVal(cty.String),
			}), ric.After)
		default:
			t.Fatal("unknown instance:", i)
		}
	}
}

// ensure that outputs missing references due to targetting are removed from
// the graph.
func TestContext2Plan_outputContainsTargetedResource(t *testing.T) {
	m := testModule(t, "plan-untargeted-resource-output")
	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Child("mod", addrs.NoKey).Resource(
				addrs.ManagedResourceMode, "aws_instance", "a",
			),
		},
	})

	_, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("err: %s", diags)
	}
	if len(diags) != 1 {
		t.Fatalf("got %d diagnostics; want 1", diags)
	}
	if got, want := diags[0].Severity(), tfdiags.Warning; got != want {
		t.Errorf("wrong diagnostic severity %#v; want %#v", got, want)
	}
	if got, want := diags[0].Description().Summary, "Resource targeting is in effect"; got != want {
		t.Errorf("wrong diagnostic summary %#v; want %#v", got, want)
	}
}

// https://github.com/hashicorp/terraform/issues/4515
func TestContext2Plan_targetedOverTen(t *testing.T) {
	m := testModule(t, "plan-targeted-over-ten")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	for i := 0; i < 13; i++ {
		key := fmt.Sprintf("aws_instance.foo[%d]", i)
		id := fmt.Sprintf("i-abc%d", i)
		attrs := fmt.Sprintf(`{"id":"%s","type":"aws_instance"}`, id)

		root.SetResourceInstanceCurrent(
			mustResourceInstanceAddr(key).Resource,
			&states.ResourceInstanceObjectSrc{
				Status:    states.ObjectReady,
				AttrsJSON: []byte(attrs),
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
		)
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		State: state,
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

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	for _, res := range plan.Changes.Resources {
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}
		if res.Action != plans.NoOp {
			t.Fatalf("unexpected action %s for %s", res.Action, ric.Addr)
		}
	}
}

func TestContext2Plan_provider(t *testing.T) {
	m := testModule(t, "plan-provider")
	p := testProvider("aws")

	var value interface{}
	p.ConfigureProviderFn = func(req providers.ConfigureProviderRequest) (resp providers.ConfigureProviderResponse) {
		value = req.Config.GetAttr("foo").AsString()
		return
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
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
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
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
	p.PlanResourceChangeFn = testDiffFn

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"bar","ami":"ami-abcd1234","type":"aws_instance"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		Variables: InputValues{
			"foo": &InputValue{
				Value:      cty.StringVal("ami-1234abcd"),
				SourceType: ValueFromCaller,
			},
		},
		State: state,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 1 {
		t.Fatal("expected 1 changes, got", len(plan.Changes.Resources))
	}

	res := plan.Changes.Resources[0]
	ric, err := res.Decode(ty)
	if err != nil {
		t.Fatal(err)
	}

	if ric.Addr.String() != "aws_instance.foo" {
		t.Fatalf("unexpected resource: %s", ric.Addr)
	}

	checkVals(t, objectVal(t, schema, map[string]cty.Value{
		"id":   cty.StringVal("bar"),
		"ami":  cty.StringVal("ami-abcd1234"),
		"type": cty.StringVal("aws_instance"),
	}), ric.After)
}

func TestContext2Plan_ignoreChangesWildcard(t *testing.T) {
	m := testModule(t, "plan-ignore-changes-wildcard")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"bar","ami":"ami-abcd1234","instance":"t2.micro","type":"aws_instance"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
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
		State: state,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	for _, res := range plan.Changes.Resources {
		if res.Action != plans.NoOp {
			t.Fatalf("unexpected resource diffs in root module: %s", spew.Sdump(plan.Changes.Resources))
		}
	}
}

func TestContext2Plan_ignoreChangesInMap(t *testing.T) {
	p := testProvider("test")

	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_ignore_changes_map": {
				Attributes: map[string]*configschema.Attribute{
					"tags": {Type: cty.Map(cty.String), Optional: true},
				},
			},
		},
	})
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		return providers.PlanResourceChangeResponse{
			PlannedState: req.ProposedNewState,
		}
	}

	s := states.BuildState(func(ss *states.SyncState) {
		ss.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_ignore_changes_map",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				Status:    states.ObjectReady,
				AttrsJSON: []byte(`{"id":"foo","tags":{"ignored":"from state","other":"from state"},"type":"aws_instance"}`),
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})
	m := testModule(t, "plan-ignore-changes-in-map")

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
		State: s,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["test_ignore_changes_map"].Block
	ty := schema.ImpliedType()

	if got, want := len(plan.Changes.Resources), 1; got != want {
		t.Fatalf("wrong number of changes %d; want %d", got, want)
	}

	res := plan.Changes.Resources[0]
	ric, err := res.Decode(ty)
	if err != nil {
		t.Fatal(err)
	}
	if res.Action != plans.Update {
		t.Fatalf("resource %s should be updated, got %s", ric.Addr, res.Action)
	}

	if got, want := ric.Addr.String(), "test_ignore_changes_map.foo"; got != want {
		t.Fatalf("unexpected resource address %s; want %s", got, want)
	}

	checkVals(t, objectVal(t, schema, map[string]cty.Value{
		"tags": cty.MapVal(map[string]cty.Value{
			"ignored": cty.StringVal("from state"),
			"other":   cty.StringVal("from config"),
		}),
	}), ric.After)
}

func TestContext2Plan_ignoreChangesSensitive(t *testing.T) {
	m := testModule(t, "plan-ignore-changes-sensitive")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"bar","ami":"ami-abcd1234","type":"aws_instance"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		Variables: InputValues{
			"foo": &InputValue{
				Value:      cty.StringVal("ami-1234abcd"),
				SourceType: ValueFromCaller,
			},
		},
		State: state,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 1 {
		t.Fatal("expected 1 changes, got", len(plan.Changes.Resources))
	}

	res := plan.Changes.Resources[0]
	ric, err := res.Decode(ty)
	if err != nil {
		t.Fatal(err)
	}

	if ric.Addr.String() != "aws_instance.foo" {
		t.Fatalf("unexpected resource: %s", ric.Addr)
	}

	checkVals(t, objectVal(t, schema, map[string]cty.Value{
		"id":   cty.StringVal("bar"),
		"ami":  cty.StringVal("ami-abcd1234"),
		"type": cty.StringVal("aws_instance"),
	}), ric.After)
}

func TestContext2Plan_moduleMapLiteral(t *testing.T) {
	m := testModule(t, "plan-module-map-literal")
	p := testProvider("aws")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"meta": {Type: cty.Map(cty.String), Optional: true},
					"tags": {Type: cty.Map(cty.String), Optional: true},
				},
			},
		},
	})
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		s := req.ProposedNewState.AsValueMap()
		m := s["tags"].AsValueMap()

		if m["foo"].AsString() != "bar" {
			t.Fatalf("Bad value in tags attr: %#v", m)
		}

		meta := s["meta"].AsValueMap()
		if len(meta) != 0 {
			t.Fatalf("Meta attr not empty: %#v", meta)
		}
		return testDiffFn(req)
	}
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}
}

func TestContext2Plan_computedValueInMap(t *testing.T) {
	m := testModule(t, "plan-computed-value-in-map")
	p := testProvider("aws")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
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
	})
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		resp = testDiffFn(req)

		if req.TypeName != "aws_computed_source" {
			return
		}

		planned := resp.PlannedState.AsValueMap()
		planned["computed_read_only"] = cty.UnknownVal(cty.String)
		resp.PlannedState = cty.ObjectVal(planned)
		return resp
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if len(plan.Changes.Resources) != 2 {
		t.Fatal("expected 2 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		schema := p.GetProviderSchemaResponse.ResourceTypes[res.Addr.Resource.Resource.Type].Block

		ric, err := res.Decode(schema.ImpliedType())
		if err != nil {
			t.Fatal(err)
		}

		if res.Action != plans.Create {
			t.Fatalf("resource %s should be created", ric.Addr)
		}

		switch i := ric.Addr.String(); i {
		case "aws_computed_source.intermediates":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"computed_read_only": cty.UnknownVal(cty.String),
			}), ric.After)
		case "module.test_mod.aws_instance.inner2":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"looked_up": cty.UnknownVal(cty.String),
			}), ric.After)
		default:
			t.Fatal("unknown instance:", i)
		}
	}
}

func TestContext2Plan_moduleVariableFromSplat(t *testing.T) {
	m := testModule(t, "plan-module-variable-from-splat")
	p := testProvider("aws")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"thing": {Type: cty.String, Optional: true},
				},
			},
		},
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if len(plan.Changes.Resources) != 4 {
		t.Fatal("expected 4 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		schema := p.GetProviderSchemaResponse.ResourceTypes[res.Addr.Resource.Resource.Type].Block

		ric, err := res.Decode(schema.ImpliedType())
		if err != nil {
			t.Fatal(err)
		}

		if res.Action != plans.Create {
			t.Fatalf("resource %s should be created", ric.Addr)
		}

		switch i := ric.Addr.String(); i {
		case "module.mod1.aws_instance.test[0]",
			"module.mod1.aws_instance.test[1]",
			"module.mod2.aws_instance.test[0]",
			"module.mod2.aws_instance.test[1]":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"thing": cty.StringVal("doesnt"),
			}), ric.After)
		default:
			t.Fatal("unknown instance:", i)
		}
	}
}

func TestContext2Plan_createBeforeDestroy_depends_datasource(t *testing.T) {
	m := testModule(t, "plan-cbd-depends-datasource")
	p := testProvider("aws")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"num":      {Type: cty.String, Optional: true},
					"computed": {Type: cty.String, Optional: true, Computed: true},
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
	})
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		computedVal := req.ProposedNewState.GetAttr("computed")
		if computedVal.IsNull() {
			computedVal = cty.UnknownVal(cty.String)
		}
		return providers.PlanResourceChangeResponse{
			PlannedState: cty.ObjectVal(map[string]cty.Value{
				"num":      req.ProposedNewState.GetAttr("num"),
				"computed": computedVal,
			}),
		}
	}
	p.ReadDataSourceFn = func(req providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
		cfg := req.Config.AsValueMap()
		cfg["id"] = cty.StringVal("data_id")
		return providers.ReadDataSourceResponse{
			State: cty.ObjectVal(cfg),
		}
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	seenAddrs := make(map[string]struct{})
	for _, res := range plan.Changes.Resources {
		var schema *configschema.Block
		switch res.Addr.Resource.Resource.Mode {
		case addrs.DataResourceMode:
			schema = p.GetProviderSchemaResponse.DataSources[res.Addr.Resource.Resource.Type].Block
		case addrs.ManagedResourceMode:
			schema = p.GetProviderSchemaResponse.ResourceTypes[res.Addr.Resource.Resource.Type].Block
		}

		ric, err := res.Decode(schema.ImpliedType())
		if err != nil {
			t.Fatal(err)
		}

		seenAddrs[ric.Addr.String()] = struct{}{}

		t.Run(ric.Addr.String(), func(t *testing.T) {
			switch i := ric.Addr.String(); i {
			case "aws_instance.foo[0]":
				if res.Action != plans.Create {
					t.Fatalf("resource %s should be created, got %s", ric.Addr, ric.Action)
				}
				checkVals(t, objectVal(t, schema, map[string]cty.Value{
					"num":      cty.StringVal("2"),
					"computed": cty.StringVal("data_id"),
				}), ric.After)
			case "aws_instance.foo[1]":
				if res.Action != plans.Create {
					t.Fatalf("resource %s should be created, got %s", ric.Addr, ric.Action)
				}
				checkVals(t, objectVal(t, schema, map[string]cty.Value{
					"num":      cty.StringVal("2"),
					"computed": cty.StringVal("data_id"),
				}), ric.After)
			default:
				t.Fatal("unknown instance:", i)
			}
		})
	}

	wantAddrs := map[string]struct{}{
		"aws_instance.foo[0]": struct{}{},
		"aws_instance.foo[1]": struct{}{},
	}
	if !cmp.Equal(seenAddrs, wantAddrs) {
		t.Errorf("incorrect addresses in changeset:\n%s", cmp.Diff(wantAddrs, seenAddrs))
	}
}

// interpolated lists need to be stored in the original order.
func TestContext2Plan_listOrder(t *testing.T) {
	m := testModule(t, "plan-list-order")
	p := testProvider("aws")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.List(cty.String), Optional: true},
				},
			},
		},
	})
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
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

	if !cmp.Equal(rDiffA.After, rDiffB.After, valueComparer) {
		t.Fatal(cmp.Diff(rDiffA.After, rDiffB.After, valueComparer))
	}
}

// Make sure ignore-changes doesn't interfere with set/list/map diffs.
// If a resource was being replaced by a RequiresNew attribute that gets
// ignored, we need to filter the diff properly to properly update rather than
// replace.
func TestContext2Plan_ignoreChangesWithFlatmaps(t *testing.T) {
	m := testModule(t, "plan-ignore-changes-with-flatmaps")
	p := testProvider("aws")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
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
	})

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo").Resource,
		&states.ResourceInstanceObjectSrc{
			Status: states.ObjectReady,
			AttrsJSON: []byte(`{
				"user_data":"x","require_new":"",
				"set":[{"a":"1"}],
				"lst":["j"]
			}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		State: state,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if len(plan.Changes.Resources) != 1 {
		t.Fatal("expected 1 changes, got", len(plan.Changes.Resources))
	}

	res := plan.Changes.Resources[0]
	schema := p.GetProviderSchemaResponse.ResourceTypes[res.Addr.Resource.Resource.Type].Block

	ric, err := res.Decode(schema.ImpliedType())
	if err != nil {
		t.Fatal(err)
	}

	if res.Action != plans.Update {
		t.Fatalf("resource %s should be updated, got %s", ric.Addr, ric.Action)
	}

	if ric.Addr.String() != "aws_instance.foo" {
		t.Fatalf("unknown resource: %s", ric.Addr)
	}

	checkVals(t, objectVal(t, schema, map[string]cty.Value{
		"lst": cty.ListVal([]cty.Value{
			cty.StringVal("j"),
			cty.StringVal("k"),
		}),
		"require_new": cty.StringVal(""),
		"user_data":   cty.StringVal("x"),
		"set": cty.ListVal([]cty.Value{cty.MapVal(map[string]cty.Value{
			"a": cty.StringVal("1"),
			"b": cty.StringVal("2"),
		})}),
	}), ric.After)
}

// TestContext2Plan_resourceNestedCount ensures resource sets that depend on
// the count of another resource set (ie: count of a data source that depends
// on another data source's instance count - data.x.foo.*.id) get properly
// normalized to the indexes they should be. This case comes up when there is
// an existing state (after an initial apply).
func TestContext2Plan_resourceNestedCount(t *testing.T) {
	m := testModule(t, "nested-resource-count-plan")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn
	p.ReadResourceFn = func(req providers.ReadResourceRequest) providers.ReadResourceResponse {
		return providers.ReadResourceResponse{
			NewState: req.PriorState,
		}
	}

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo[0]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"foo0","type":"aws_instance"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo[1]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"foo1","type":"aws_instance"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.bar[0]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:       states.ObjectReady,
			AttrsJSON:    []byte(`{"id":"bar0","type":"aws_instance"}`),
			Dependencies: []addrs.ConfigResource{mustConfigResourceAddr("aws_instance.foo")},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.bar[1]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:       states.ObjectReady,
			AttrsJSON:    []byte(`{"id":"bar1","type":"aws_instance"}`),
			Dependencies: []addrs.ConfigResource{mustConfigResourceAddr("aws_instance.foo")},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.baz[0]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:       states.ObjectReady,
			AttrsJSON:    []byte(`{"id":"baz0","type":"aws_instance"}`),
			Dependencies: []addrs.ConfigResource{mustConfigResourceAddr("aws_instance.bar")},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.baz[1]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:       states.ObjectReady,
			AttrsJSON:    []byte(`{"id":"baz1","type":"aws_instance"}`),
			Dependencies: []addrs.ConfigResource{mustConfigResourceAddr("aws_instance.bar")},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		State: state,
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

	for _, res := range plan.Changes.Resources {
		if res.Action != plans.NoOp {
			t.Fatalf("resource %s should not change, plan returned %s", res.Addr, res.Action)
		}
	}
}

// Higher level test at TestResource_dataSourceListApplyPanic
func TestContext2Plan_computedAttrRefTypeMismatch(t *testing.T) {
	m := testModule(t, "plan-computed-attr-ref-type-mismatch")
	p := testProvider("aws")
	p.ValidateResourceConfigFn = func(req providers.ValidateResourceConfigRequest) providers.ValidateResourceConfigResponse {
		var diags tfdiags.Diagnostics
		if req.TypeName == "aws_instance" {
			amiVal := req.Config.GetAttr("ami")
			if amiVal.Type() != cty.String {
				diags = diags.Append(fmt.Errorf("Expected ami to be cty.String, got %#v", amiVal))
			}
		}
		return providers.ValidateResourceConfigResponse{
			Diagnostics: diags,
		}
	}
	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) (resp providers.ApplyResourceChangeResponse) {
		if req.TypeName != "aws_ami_list" {
			t.Fatalf("Reached apply for unexpected resource type! %s", req.TypeName)
		}
		// Pretend like we make a thing and the computed list "ids" is populated
		s := req.PlannedState.AsValueMap()
		s["id"] = cty.StringVal("someid")
		s["ids"] = cty.ListVal([]cty.Value{
			cty.StringVal("ami-abc123"),
			cty.StringVal("ami-bcd345"),
		})

		resp.NewState = cty.ObjectVal(s)
		return
	}
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
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
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
	})

	m := testModule(t, "plan-self-ref")
	c := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
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
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
	})

	m := testModule(t, "plan-self-ref-multi")
	c := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
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
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.List(cty.String), Optional: true},
				},
			},
		},
	})

	m := testModule(t, "plan-self-ref-multi-all")
	c := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
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

	// The graph is checked for cycles before we can walk it, so we don't
	// encounter the self-reference check.
	//wantErrStr := "Self-referential block"
	wantErrStr := "Cycle"
	if !strings.Contains(gotErrStr, wantErrStr) {
		t.Fatalf("missing expected error\ngot: %s\n\nwant: error containing %q", gotErrStr, wantErrStr)
	}
}

func TestContext2Plan_invalidOutput(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
data "aws_data_source" "name" {}

output "out" {
  value = data.aws_data_source.name.missing
}`,
	})

	p := testProvider("aws")
	p.ReadDataSourceResponse = &providers.ReadDataSourceResponse{
		State: cty.ObjectVal(map[string]cty.Value{
			"id":  cty.StringVal("data_id"),
			"foo": cty.StringVal("foo"),
		}),
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan()
	if !diags.HasErrors() {
		// Should get this error:
		// Unsupported attribute: This object does not have an attribute named "missing"
		t.Fatal("succeeded; want errors")
	}

	gotErrStr := diags.Err().Error()
	wantErrStr := "Unsupported attribute"
	if !strings.Contains(gotErrStr, wantErrStr) {
		t.Fatalf("missing expected error\ngot: %s\n\nwant: error containing %q", gotErrStr, wantErrStr)
	}
}

func TestContext2Plan_invalidModuleOutput(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"child/main.tf": `
data "aws_data_source" "name" {}

output "out" {
  value = "${data.aws_data_source.name.missing}"
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
	p.ReadDataSourceResponse = &providers.ReadDataSourceResponse{
		State: cty.ObjectVal(map[string]cty.Value{
			"id":  cty.StringVal("data_id"),
			"foo": cty.StringVal("foo"),
		}),
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan()
	if !diags.HasErrors() {
		// Should get this error:
		// Unsupported attribute: This object does not have an attribute named "missing"
		t.Fatal("succeeded; want errors")
	}

	gotErrStr := diags.Err().Error()
	wantErrStr := "Unsupported attribute"
	if !strings.Contains(gotErrStr, wantErrStr) {
		t.Fatalf("missing expected error\ngot: %s\n\nwant: error containing %q", gotErrStr, wantErrStr)
	}
}

func TestContext2Plan_variableValidation(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
variable "x" {
  default = "bar"
}

resource "aws_instance" "foo" {
  foo = var.x
}`,
	})

	p := testProvider("aws")
	p.ValidateResourceConfigFn = func(req providers.ValidateResourceConfigRequest) (resp providers.ValidateResourceConfigResponse) {
		foo := req.Config.GetAttr("foo").AsString()
		if foo == "bar" {
			resp.Diagnostics = resp.Diagnostics.Append(errors.New("foo cannot be bar"))
		}
		return
	}

	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		resp.PlannedState = req.ProposedNewState
		return
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan()
	if !diags.HasErrors() {
		// Should get this error:
		// Unsupported attribute: This object does not have an attribute named "missing"
		t.Fatal("succeeded; want errors")
	}
}

func TestContext2Plan_variableSensitivity(t *testing.T) {
	m := testModule(t, "plan-variable-sensitivity")

	p := testProvider("aws")
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		resp.PlannedState = req.ProposedNewState
		return
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}
	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 1 {
		t.Fatal("expected 1 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		if res.Action != plans.Create {
			t.Fatalf("expected resource creation, got %s", res.Action)
		}
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "aws_instance.foo":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"foo": cty.StringVal("foo").Mark(marks.Sensitive),
			}), ric.After)
			if len(res.ChangeSrc.BeforeValMarks) != 0 {
				t.Errorf("unexpected BeforeValMarks: %#v", res.ChangeSrc.BeforeValMarks)
			}
			if len(res.ChangeSrc.AfterValMarks) != 1 {
				t.Errorf("unexpected AfterValMarks: %#v", res.ChangeSrc.AfterValMarks)
				continue
			}
			pvm := res.ChangeSrc.AfterValMarks[0]
			if got, want := pvm.Path, cty.GetAttrPath("foo"); !got.Equals(want) {
				t.Errorf("unexpected path for mark\n got: %#v\nwant: %#v", got, want)
			}
			if got, want := pvm.Marks, cty.NewValueMarks(marks.Sensitive); !got.Equal(want) {
				t.Errorf("unexpected value for mark\n got: %#v\nwant: %#v", got, want)
			}
		default:
			t.Fatal("unknown instance:", i)
		}
	}
}

func TestContext2Plan_variableSensitivityModule(t *testing.T) {
	m := testModule(t, "plan-variable-sensitivity-module")

	p := testProvider("aws")
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		resp.PlannedState = req.ProposedNewState
		return
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		Variables: InputValues{
			"another_var": &InputValue{
				Value:      cty.StringVal("boop"),
				SourceType: ValueFromCaller,
			},
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}
	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 1 {
		t.Fatal("expected 1 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		if res.Action != plans.Create {
			t.Fatalf("expected resource creation, got %s", res.Action)
		}
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "module.child.aws_instance.foo":
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"foo":   cty.StringVal("foo").Mark(marks.Sensitive),
				"value": cty.StringVal("boop").Mark(marks.Sensitive),
			}), ric.After)
			if len(res.ChangeSrc.BeforeValMarks) != 0 {
				t.Errorf("unexpected BeforeValMarks: %#v", res.ChangeSrc.BeforeValMarks)
			}
			if len(res.ChangeSrc.AfterValMarks) != 2 {
				t.Errorf("expected AfterValMarks to contain two elements: %#v", res.ChangeSrc.AfterValMarks)
				continue
			}
			// validate that the after marks have "foo" and "value"
			contains := func(pvmSlice []cty.PathValueMarks, stepName string) bool {
				for _, pvm := range pvmSlice {
					if pvm.Path.Equals(cty.GetAttrPath(stepName)) {
						if pvm.Marks.Equal(cty.NewValueMarks(marks.Sensitive)) {
							return true
						}
					}
				}
				return false
			}
			if !contains(res.ChangeSrc.AfterValMarks, "foo") {
				t.Error("unexpected AfterValMarks to contain \"foo\" with sensitive mark")
			}
			if !contains(res.ChangeSrc.AfterValMarks, "value") {
				t.Error("unexpected AfterValMarks to contain \"value\" with sensitive mark")
			}
		default:
			t.Fatal("unknown instance:", i)
		}
	}
}

func checkVals(t *testing.T, expected, got cty.Value) {
	t.Helper()
	// The GoStringer format seems to result in the closest thing to a useful
	// diff for values with marks.
	// TODO: if we want to continue using cmp.Diff on cty.Values, we should
	// make a transformer that creates a more comparable structure.
	valueTrans := cmp.Transformer("gostring", func(v cty.Value) string {
		return fmt.Sprintf("%#v\n", v)
	})
	if !cmp.Equal(expected, got, valueComparer, typeComparer, equateEmpty) {
		t.Fatal(cmp.Diff(expected, got, valueTrans, equateEmpty))
	}
}

func objectVal(t *testing.T, schema *configschema.Block, m map[string]cty.Value) cty.Value {
	t.Helper()
	v, err := schema.CoerceValue(
		cty.ObjectVal(m),
	)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestContext2Plan_requiredModuleOutput(t *testing.T) {
	m := testModule(t, "plan-required-output")
	p := testProvider("test")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_resource": {
				Attributes: map[string]*configschema.Attribute{
					"id":       {Type: cty.String, Computed: true},
					"required": {Type: cty.String, Required: true},
				},
			},
		},
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["test_resource"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 2 {
		t.Fatal("expected 2 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		t.Run(fmt.Sprintf("%s %s", res.Action, res.Addr), func(t *testing.T) {
			if res.Action != plans.Create {
				t.Fatalf("expected resource creation, got %s", res.Action)
			}
			ric, err := res.Decode(ty)
			if err != nil {
				t.Fatal(err)
			}

			var expected cty.Value
			switch i := ric.Addr.String(); i {
			case "test_resource.root":
				expected = objectVal(t, schema, map[string]cty.Value{
					"id":       cty.UnknownVal(cty.String),
					"required": cty.UnknownVal(cty.String),
				})
			case "module.mod.test_resource.for_output":
				expected = objectVal(t, schema, map[string]cty.Value{
					"id":       cty.UnknownVal(cty.String),
					"required": cty.StringVal("val"),
				})
			default:
				t.Fatal("unknown instance:", i)
			}

			checkVals(t, expected, ric.After)
		})
	}
}

func TestContext2Plan_requiredModuleObject(t *testing.T) {
	m := testModule(t, "plan-required-whole-mod")
	p := testProvider("test")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_resource": {
				Attributes: map[string]*configschema.Attribute{
					"id":       {Type: cty.String, Computed: true},
					"required": {Type: cty.String, Required: true},
				},
			},
		},
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	schema := p.GetProviderSchemaResponse.ResourceTypes["test_resource"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 2 {
		t.Fatal("expected 2 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		t.Run(fmt.Sprintf("%s %s", res.Action, res.Addr), func(t *testing.T) {
			if res.Action != plans.Create {
				t.Fatalf("expected resource creation, got %s", res.Action)
			}
			ric, err := res.Decode(ty)
			if err != nil {
				t.Fatal(err)
			}

			var expected cty.Value
			switch i := ric.Addr.String(); i {
			case "test_resource.root":
				expected = objectVal(t, schema, map[string]cty.Value{
					"id":       cty.UnknownVal(cty.String),
					"required": cty.UnknownVal(cty.String),
				})
			case "module.mod.test_resource.for_output":
				expected = objectVal(t, schema, map[string]cty.Value{
					"id":       cty.UnknownVal(cty.String),
					"required": cty.StringVal("val"),
				})
			default:
				t.Fatal("unknown instance:", i)
			}

			checkVals(t, expected, ric.After)
		})
	}
}

func TestContext2Plan_expandOrphan(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
module "mod" {
  count = 1
  source = "./mod"
}
`,
		"mod/main.tf": `
resource "aws_instance" "foo" {
}
`,
	})

	state := states.NewState()
	state.EnsureModule(addrs.RootModuleInstance.Child("mod", addrs.IntKey(0))).SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"child","type":"aws_instance"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	state.EnsureModule(addrs.RootModuleInstance.Child("mod", addrs.IntKey(1))).SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"child","type":"aws_instance"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		State: state,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}

	expected := map[string]plans.Action{
		`module.mod[1].aws_instance.foo`: plans.Delete,
		`module.mod[0].aws_instance.foo`: plans.NoOp,
	}

	for _, res := range plan.Changes.Resources {
		want := expected[res.Addr.String()]
		if res.Action != want {
			t.Fatalf("expected %s action, got: %q %s", want, res.Addr, res.Action)
		}
		delete(expected, res.Addr.String())
	}

	for res, action := range expected {
		t.Errorf("missing %s change for %s", action, res)
	}
}

func TestContext2Plan_indexInVar(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
module "a" {
  count = 1
  source = "./mod"
  in = "test"
}

module "b" {
  count = 1
  source = "./mod"
  in = length(module.a)
}
`,
		"mod/main.tf": `
resource "aws_instance" "foo" {
  foo = var.in
}

variable "in" {
}

output"out" {
  value = aws_instance.foo.id
}
`,
	})

	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}
}

func TestContext2Plan_targetExpandedAddress(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
module "mod" {
  count = 3
  source = "./mod"
}
`,
		"mod/main.tf": `
resource "aws_instance" "foo" {
  count = 2
}
`,
	})

	p := testProvider("aws")

	targets := []addrs.Targetable{}
	target, diags := addrs.ParseTargetStr("module.mod[1].aws_instance.foo[0]")
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}
	targets = append(targets, target.Subject)

	target, diags = addrs.ParseTargetStr("module.mod[2]")
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}
	targets = append(targets, target.Subject)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		Targets: targets,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}

	expected := map[string]plans.Action{
		// the single targeted mod[1] instances
		`module.mod[1].aws_instance.foo[0]`: plans.Create,
		// the whole mode[2]
		`module.mod[2].aws_instance.foo[0]`: plans.Create,
		`module.mod[2].aws_instance.foo[1]`: plans.Create,
	}

	for _, res := range plan.Changes.Resources {
		want := expected[res.Addr.String()]
		if res.Action != want {
			t.Fatalf("expected %s action, got: %q %s", want, res.Addr, res.Action)
		}
		delete(expected, res.Addr.String())
	}

	for res, action := range expected {
		t.Errorf("missing %s change for %s", action, res)
	}
}

func TestContext2Plan_targetResourceInModuleInstance(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
module "mod" {
  count = 3
  source = "./mod"
}
`,
		"mod/main.tf": `
resource "aws_instance" "foo" {
}
`,
	})

	p := testProvider("aws")

	target, diags := addrs.ParseTargetStr("module.mod[1].aws_instance.foo")
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}

	targets := []addrs.Targetable{target.Subject}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		Targets: targets,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}

	expected := map[string]plans.Action{
		// the single targeted mod[1] instance
		`module.mod[1].aws_instance.foo`: plans.Create,
	}

	for _, res := range plan.Changes.Resources {
		want := expected[res.Addr.String()]
		if res.Action != want {
			t.Fatalf("expected %s action, got: %q %s", want, res.Addr, res.Action)
		}
		delete(expected, res.Addr.String())
	}

	for res, action := range expected {
		t.Errorf("missing %s change for %s", action, res)
	}
}

func TestContext2Plan_moduleRefIndex(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
module "mod" {
  for_each = {
    a = "thing"
  }
  in = null
  source = "./mod"
}

module "single" {
  source = "./mod"
  in = module.mod["a"]
}
`,
		"mod/main.tf": `
variable "in" {
}

output "out" {
  value = "foo"
}

resource "aws_instance" "foo" {
}
`,
	})

	p := testProvider("aws")

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}
}

func TestContext2Plan_noChangeDataPlan(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
data "test_data_source" "foo" {}
`,
	})

	p := new(MockProvider)
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
		DataSources: map[string]*configschema.Block{
			"test_data_source": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"foo": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
		},
	})

	p.ReadDataSourceResponse = &providers.ReadDataSourceResponse{
		State: cty.ObjectVal(map[string]cty.Value{
			"id":  cty.StringVal("data_id"),
			"foo": cty.StringVal("foo"),
		}),
	}

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("data.test_data_source.foo").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"data_id", "foo":"foo"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
		State: state,
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}

	for _, res := range plan.Changes.Resources {
		if res.Action != plans.NoOp {
			t.Fatalf("expected NoOp, got: %q %s", res.Addr, res.Action)
		}
	}
}

// for_each can reference a resource with 0 instances
func TestContext2Plan_scaleInForEach(t *testing.T) {
	p := testProvider("test")

	m := testModuleInline(t, map[string]string{
		"main.tf": `
locals {
  m = {}
}

resource "test_instance" "a" {
  for_each = local.m
}

resource "test_instance" "b" {
  for_each = test_instance.a
}
`})

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_instance.a[0]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:       states.ObjectReady,
			AttrsJSON:    []byte(`{"id":"a0"}`),
			Dependencies: []addrs.ConfigResource{},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_instance.b").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:       states.ObjectReady,
			AttrsJSON:    []byte(`{"id":"b"}`),
			Dependencies: []addrs.ConfigResource{mustConfigResourceAddr("test_instance.a")},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
		State: state,
	})

	_, diags := ctx.Plan()
	assertNoErrors(t, diags)
}

func TestContext2Plan_targetedModuleInstance(t *testing.T) {
	m := testModule(t, "plan-targeted")
	p := testProvider("aws")
	p.PlanResourceChangeFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Child("mod", addrs.IntKey(0)),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}
	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
	ty := schema.ImpliedType()

	if len(plan.Changes.Resources) != 1 {
		t.Fatal("expected 1 changes, got", len(plan.Changes.Resources))
	}

	for _, res := range plan.Changes.Resources {
		ric, err := res.Decode(ty)
		if err != nil {
			t.Fatal(err)
		}

		switch i := ric.Addr.String(); i {
		case "module.mod[0].aws_instance.foo":
			if res.Action != plans.Create {
				t.Fatalf("resource %s should be created", i)
			}
			checkVals(t, objectVal(t, schema, map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"num":  cty.NumberIntVal(2),
				"type": cty.UnknownVal(cty.String),
			}), ric.After)
		default:
			t.Fatal("unknown instance:", i)
		}
	}
}

func TestContext2Plan_dataRefreshedInPlan(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
data "test_data_source" "d" {
}
`})

	p := testProvider("test")
	p.ReadDataSourceResponse = &providers.ReadDataSourceResponse{
		State: cty.ObjectVal(map[string]cty.Value{
			"id":  cty.StringVal("this"),
			"foo": cty.NullVal(cty.String),
		}),
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}

	d := plan.PriorState.ResourceInstance(mustResourceInstanceAddr("data.test_data_source.d"))
	if d == nil || d.Current == nil {
		t.Fatal("data.test_data_source.d not found in state:", plan.PriorState)
	}

	if d.Current.Status != states.ObjectReady {
		t.Fatal("expected data.test_data_source.d to be fully read in refreshed state, got status", d.Current.Status)
	}
}

func TestContext2Plan_dataReferencesResource(t *testing.T) {
	p := testProvider("test")

	p.ReadDataSourceFn = func(req providers.ReadDataSourceRequest) (resp providers.ReadDataSourceResponse) {
		resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("data source should not be read"))
		return resp
	}

	m := testModuleInline(t, map[string]string{
		"main.tf": `
locals {
  x = "value"
}

resource "test_resource" "a" {
  value = local.x
}

// test_resource.a.value can be resolved during plan, but the reference implies
// that the data source should wait until the resource is created.
data "test_data_source" "d" {
  foo = test_resource.a.value
}

// ensure referencing an indexed instance that has not yet created will also
// delay reading the data source
resource "test_resource" "b" {
  count = 2
  value = local.x
}

data "test_data_source" "e" {
  foo = test_resource.b[0].value
}
`})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan()
	assertNoErrors(t, diags)
}

func TestContext2Plan_skipRefresh(t *testing.T) {
	p := testProvider("test")
	p.PlanResourceChangeFn = testDiffFn

	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_instance" "a" {
}
`})

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_instance.a").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:       states.ObjectReady,
			AttrsJSON:    []byte(`{"id":"a","type":"test_instance"}`),
			Dependencies: []addrs.ConfigResource{},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
		State:       state,
		SkipRefresh: true,
	})

	plan, diags := ctx.Plan()
	assertNoErrors(t, diags)

	if p.ReadResourceCalled {
		t.Fatal("Resource should not have been refreshed")
	}

	for _, c := range plan.Changes.Resources {
		if c.Action != plans.NoOp {
			t.Fatalf("expected no changes, got %s for %q", c.Action, c.Addr)
		}
	}
}

func TestContext2Plan_dataInModuleDependsOn(t *testing.T) {
	p := testProvider("test")

	readDataSourceB := false
	p.ReadDataSourceFn = func(req providers.ReadDataSourceRequest) (resp providers.ReadDataSourceResponse) {
		cfg := req.Config.AsValueMap()
		foo := cfg["foo"].AsString()

		cfg["id"] = cty.StringVal("ID")
		cfg["foo"] = cty.StringVal("new")

		if foo == "b" {
			readDataSourceB = true
		}

		resp.State = cty.ObjectVal(cfg)
		return resp
	}

	m := testModuleInline(t, map[string]string{
		"main.tf": `
module "a" {
  source = "./mod_a"
}

module "b" {
  source = "./mod_b"
  depends_on = [module.a]
}`,
		"mod_a/main.tf": `
data "test_data_source" "a" {
  foo = "a"
}`,
		"mod_b/main.tf": `
data "test_data_source" "b" {
  foo = "b"
}`,
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan()
	assertNoErrors(t, diags)

	// The change to data source a should not prevent data source b from being
	// read.
	if !readDataSourceB {
		t.Fatal("data source b was not read during plan")
	}
}

func TestContext2Plan_rpcDiagnostics(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_instance" "a" {
}
`,
	})

	p := testProvider("test")
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		resp := testDiffFn(req)
		resp.Diagnostics = resp.Diagnostics.Append(tfdiags.SimpleWarning("don't frobble"))
		return resp
	}

	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_instance": {
				Attributes: map[string]*configschema.Attribute{
					"id": {Type: cty.String, Computed: true},
				},
			},
		},
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})
	_, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	if len(diags) == 0 {
		t.Fatal("expected warnings")
	}

	for _, d := range diags {
		des := d.Description().Summary
		if !strings.Contains(des, "frobble") {
			t.Fatalf(`expected frobble, got %q`, des)
		}
	}
}

// ignore_changes needs to be re-applied to the planned value for provider
// using the LegacyTypeSystem
func TestContext2Plan_legacyProviderIgnoreChanges(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_instance" "a" {
  lifecycle {
    ignore_changes = [data]
  }
}
`,
	})

	p := testProvider("test")
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		m := req.ProposedNewState.AsValueMap()
		// this provider "hashes" the data attribute as bar
		m["data"] = cty.StringVal("bar")

		resp.PlannedState = cty.ObjectVal(m)
		resp.LegacyTypeSystem = true
		return resp
	}

	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_instance": {
				Attributes: map[string]*configschema.Attribute{
					"id":   {Type: cty.String, Computed: true},
					"data": {Type: cty.String, Optional: true},
				},
			},
		},
	})

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_instance.a").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:       states.ObjectReady,
			AttrsJSON:    []byte(`{"id":"a","data":"foo"}`),
			Dependencies: []addrs.ConfigResource{},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
		State: state,
	})
	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	for _, c := range plan.Changes.Resources {
		if c.Action != plans.NoOp {
			t.Fatalf("expected no changes, got %s for %q", c.Action, c.Addr)
		}
	}
}

func TestContext2Plan_validateIgnoreAll(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_instance" "a" {
  lifecycle {
    ignore_changes = all
  }
}
`,
	})

	p := testProvider("test")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_instance": {
				Attributes: map[string]*configschema.Attribute{
					"id":   {Type: cty.String, Computed: true},
					"data": {Type: cty.String, Optional: true},
				},
			},
		},
	})
	p.ValidateResourceConfigFn = func(req providers.ValidateResourceConfigRequest) providers.ValidateResourceConfigResponse {
		var diags tfdiags.Diagnostics
		if req.TypeName == "test_instance" {
			if !req.Config.GetAttr("id").IsNull() {
				diags = diags.Append(errors.New("id cannot be set in config"))
			}
		}
		return providers.ValidateResourceConfigResponse{
			Diagnostics: diags,
		}
	}

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_instance.a").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:       states.ObjectReady,
			AttrsJSON:    []byte(`{"id":"a","data":"foo"}`),
			Dependencies: []addrs.ConfigResource{},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
		State: state,
	})
	_, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}
}

func TestContext2Plan_dataRemovalNoProvider(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_instance" "a" {
}
`,
	})

	p := testProvider("test")

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_instance.a").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:       states.ObjectReady,
			AttrsJSON:    []byte(`{"id":"a","data":"foo"}`),
			Dependencies: []addrs.ConfigResource{},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	// the provider for this data source is no longer in the config, but that
	// should not matter for state removal.
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("data.test_data_source.d").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:       states.ObjectReady,
			AttrsJSON:    []byte(`{"id":"d"}`),
			Dependencies: []addrs.ConfigResource{},
		},
		mustProviderConfig(`provider["registry.terraform.io/local/test"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
			// We still need to be able to locate the provider to decode the
			// state, since we do not know during init that this provider is
			// only used for an orphaned data source.
			addrs.NewProvider("registry.terraform.io", "local", "test"): testProviderFuncFixed(p),
		},
		State: state,
	})
	_, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}
}

func TestContext2Plan_noSensitivityChange(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
variable "sensitive_var" {
       default = "hello"
       sensitive = true
}

resource "test_resource" "foo" {
       value = var.sensitive_var
       sensitive_value = var.sensitive_var
}`,
	})

	p := testProvider("test")

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
		State: states.BuildState(func(s *states.SyncState) {
			s.SetResourceInstanceCurrent(
				addrs.Resource{
					Mode: addrs.ManagedResourceMode,
					Type: "test_resource",
					Name: "foo",
				}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
				&states.ResourceInstanceObjectSrc{
					Status:    states.ObjectReady,
					AttrsJSON: []byte(`{"id":"foo", "value":"hello", "sensitive_value":"hello"}`),
					AttrSensitivePaths: []cty.PathValueMarks{
						{Path: cty.Path{cty.GetAttrStep{Name: "value"}}, Marks: cty.NewValueMarks(marks.Sensitive)},
						{Path: cty.Path{cty.GetAttrStep{Name: "sensitive_value"}}, Marks: cty.NewValueMarks(marks.Sensitive)},
					},
				},
				addrs.AbsProviderConfig{
					Provider: addrs.NewDefaultProvider("test"),
					Module:   addrs.RootModule,
				},
			)
		}),
	})
	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	for _, c := range plan.Changes.Resources {
		if c.Action != plans.NoOp {
			t.Fatalf("expected no changes, got %s for %q", c.Action, c.Addr)
		}
	}
}

func TestContext2Plan_variableCustomValidationsSensitive(t *testing.T) {
	m := testModule(t, "validate-variable-custom-validations-child-sensitive")

	p := testProvider("test")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan()
	if !diags.HasErrors() {
		t.Fatal("succeeded; want errors")
	}
	if got, want := diags.Err().Error(), `Invalid value for variable: Value must not be "nope".`; !strings.Contains(got, want) {
		t.Fatalf("wrong error:\ngot:  %s\nwant: message containing %q", got, want)
	}
}

func TestContext2Plan_nullOutputNoOp(t *testing.T) {
	// this should always plan a NoOp change for the output
	m := testModuleInline(t, map[string]string{
		"main.tf": `
output "planned" {
  value = false ? 1 : null
}
`,
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		State: states.BuildState(func(s *states.SyncState) {
			r := s.Module(addrs.RootModuleInstance)
			r.SetOutputValue("planned", cty.NullVal(cty.DynamicPseudoType), false)
		}),
	})
	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	for _, c := range plan.Changes.Outputs {
		if c.Action != plans.NoOp {
			t.Fatalf("expected no changes, got %s for %q", c.Action, c.Addr)
		}
	}
}

func TestContext2Plan_createOutput(t *testing.T) {
	// this should always plan a NoOp change for the output
	m := testModuleInline(t, map[string]string{
		"main.tf": `
output "planned" {
  value = 1
}
`,
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		State:  states.NewState(),
	})
	plan, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	for _, c := range plan.Changes.Outputs {
		if c.Action != plans.Create {
			t.Fatalf("expected Create change, got %s for %q", c.Action, c.Addr)
		}
	}
}

////////////////////////////////////////////////////////////////////////////////
// NOTE: Due to the size of this file, new tests should be added to
// context_plan2_test.go.
////////////////////////////////////////////////////////////////////////////////
