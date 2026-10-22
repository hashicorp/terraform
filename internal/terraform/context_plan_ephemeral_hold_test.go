// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Since all of the ephemeral hold tests and potential regressions could result
// in the walk hanging, we'll timeout after a reasonable amount of time.
const ephemeralHoldTestTimeout = 10 * time.Second

func TestContext2Plan_EphemeralHold(t *testing.T) {
	h := newEphemeralHoldTest(t)

	type heldState struct {
		value  cty.Value
		opens  int32
		closes int32
	}
	held := make(chan heldState, 1)
	release := make(chan struct{})
	done := make(chan tfdiags.Diagnostics, 1)
	go func() {
		opts := &PlanOpts{
			Mode:                      plans.NormalMode,
			AllowRootEphemeralOutputs: true,
			EphemeralHold: func(scope *lang.Scope) <-chan struct{} {
				v, _ := scope.Data.GetOutput(addrs.OutputValue{Name: "value"}, tfdiags.SourceRange{})
				held <- heldState{value: v, opens: h.opens.Load(), closes: h.closes.Load()}
				return release
			},
		}
		_, diags := h.ctx.Plan(h.module, nil, opts)
		done <- diags
	}()

	// assert the data we received from the callback
	var state heldState
	select {
	case state = <-held:
	case diags := <-done:
		t.Fatalf("plan returned before the hold was reached: %s", diags.Err())
	case <-time.After(ephemeralHoldTestTimeout):
		t.Fatal("timed out waiting for the callback")
	}
	if got, _ := state.value.Unmark(); !got.RawEquals(cty.StringVal("secret")) {
		t.Errorf("wrong output value %#v", got)
	}
	if state.opens != 1 || state.closes != 0 {
		t.Errorf("at callback time, ephemeral resource was already closed: opens=%d closes=%d", state.opens, state.closes)
	}

	// while held, assert the walk doesn't return, the resource isn't closed, and that it keeps being renewed
	deadline := time.After(ephemeralHoldTestTimeout)
	for h.renews.Load() < 2 {
		select {
		case diags := <-done:
			t.Fatalf("plan returned while held: %s", diags.Err())
		case <-deadline:
			t.Fatal("ephemeral resource was not renewed while held")
		case <-time.After(time.Millisecond):
		}
	}
	if got := h.closes.Load(); got != 0 {
		t.Fatalf("ephemeral resource closed while held (%d)", got)
	}

	// release the hold, assert no diagnostics are returned and the ephemeral resource was closed
	close(release)
	select {
	case diags := <-done:
		tfdiags.AssertNoDiagnostics(t, diags)
	case <-time.After(ephemeralHoldTestTimeout):
		t.Fatal("plan did not return after release")
	}
	if got := h.closes.Load(); got != 1 {
		t.Errorf("expected 1 close after release, got %d", got)
	}
}

func TestContext2Plan_EphemeralHold_nilChannel(t *testing.T) {
	h := newEphemeralHoldTest(t)

	called := false
	opts := &PlanOpts{
		Mode:                      plans.NormalMode,
		AllowRootEphemeralOutputs: true,
		EphemeralHold: func(*lang.Scope) <-chan struct{} {
			called = true
			return nil
		},
	}
	diags := h.plan(t, h.module, opts)
	tfdiags.AssertNoDiagnostics(t, diags)
	if !called {
		t.Error("callback not called")
	}
	if got := h.closes.Load(); got != 1 {
		t.Errorf("expected 1 close, got %d", got)
	}
}

func TestContext2Plan_EphemeralHold_stop(t *testing.T) {
	h := newEphemeralHoldTest(t)

	stopped := make(chan struct{})
	opts := &PlanOpts{
		Mode:                      plans.NormalMode,
		AllowRootEphemeralOutputs: true,
		EphemeralHold: func(*lang.Scope) <-chan struct{} {
			go func() {
				h.ctx.Stop()
				close(stopped)
			}()
			return make(chan struct{}) // never released
		},
	}
	h.plan(t, h.module, opts)

	select {
	case <-stopped:
	case <-time.After(ephemeralHoldTestTimeout):
		t.Fatal("the callback was never called")
	}
	if got := h.closes.Load(); got != 1 {
		t.Errorf("expected the resource to be closed after stop, got %d closes", got)
	}
}

func TestContext2Plan_EphemeralHold_walkErrors(t *testing.T) {
	h := newEphemeralHoldTest(t)
	h.provider.OpenEphemeralResourceFn = func(providers.OpenEphemeralResourceRequest) (resp providers.OpenEphemeralResourceResponse) {
		resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("open failed"))
		return resp
	}

	called := false
	opts := &PlanOpts{
		Mode:                      plans.NormalMode,
		AllowRootEphemeralOutputs: true,
		EphemeralHold: func(*lang.Scope) <-chan struct{} {
			// This callback won't be executed as the ephemeral resource will produce a diagnostic
			called = true
			return nil
		},
	}

	diags := h.plan(t, h.module, opts)
	wantDiags := tfdiags.Diagnostics{
		tfdiags.Sourceless(tfdiags.Error, "open failed", ""),
	}
	tfdiags.AssertDiagnosticsMatch(t, diags, wantDiags)
	if called {
		t.Error("callback must not be called when the walk had errors")
	}
}

func TestContext2Plan_EphemeralHold_noEphemeralResources(t *testing.T) {
	h := newEphemeralHoldTest(t)
	empty := testModuleInline(t, map[string]string{"main.tf": `
output "value" {
  value = "plain"
}
`})

	called := false
	opts := &PlanOpts{
		Mode:                      plans.NormalMode,
		AllowRootEphemeralOutputs: true,
		EphemeralHold: func(*lang.Scope) <-chan struct{} {
			called = true
			return nil
		},
	}
	diags := h.plan(t, empty, opts)
	tfdiags.AssertNoDiagnostics(t, diags)
	if !called {
		t.Error("callback must be called for every successful walk")
	}
}

type ephemeralHoldTest struct {
	module   *configs.Config
	provider *testing_provider.MockProvider
	ctx      *Context

	opens, renews, closes atomic.Int32
}

func newEphemeralHoldTest(t *testing.T) *ephemeralHoldTest {
	t.Helper()

	h := &ephemeralHoldTest{
		module: testModuleInline(t, map[string]string{
			"main.tf": `
ephemeral "ephem_resource" "data" {}

output "value" {
  ephemeral = true
  value     = ephemeral.ephem_resource.data.value
}
`,
		}),
		provider: &testing_provider.MockProvider{
			GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
				EphemeralResourceTypes: map[string]providers.Schema{
					"ephem_resource": {
						Body: &configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"value": {
									Type:     cty.String,
									Computed: true,
								},
							},
						},
					},
				},
			},
		},
	}

	h.provider.OpenEphemeralResourceFn = func(providers.OpenEphemeralResourceRequest) (resp providers.OpenEphemeralResourceResponse) {
		h.opens.Add(1)
		resp.Result = cty.ObjectVal(map[string]cty.Value{"value": cty.StringVal("secret")})
		resp.RenewAt = time.Now().Add(5 * time.Millisecond)
		return resp
	}
	h.provider.RenewEphemeralResourceFn = func(providers.RenewEphemeralResourceRequest) (resp providers.RenewEphemeralResourceResponse) {
		h.renews.Add(1)
		resp.RenewAt = time.Now().Add(5 * time.Millisecond)
		return resp
	}
	h.provider.CloseEphemeralResourceFn = func(providers.CloseEphemeralResourceRequest) (resp providers.CloseEphemeralResourceResponse) {
		h.closes.Add(1)
		return resp
	}

	h.ctx = testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("ephem"): testProviderFuncFixed(h.provider),
		},
	})
	return h
}

func (h *ephemeralHoldTest) plan(t *testing.T, config *configs.Config, opts *PlanOpts) tfdiags.Diagnostics {
	t.Helper()

	done := make(chan tfdiags.Diagnostics, 1)
	go func() {
		_, diags := h.ctx.Plan(config, nil, opts)
		done <- diags
	}()
	select {
	case diags := <-done:
		return diags
	case <-time.After(ephemeralHoldTestTimeout):
		t.Fatal("plan did not return")
		return nil
	}
}
