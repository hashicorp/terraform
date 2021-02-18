package terraform

import (
	"testing"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
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
