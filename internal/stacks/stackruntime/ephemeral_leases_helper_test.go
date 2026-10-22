// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package stackruntime

import (
	"fmt"
	"sync"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/providers"
	stacks_testing_provider "github.com/hashicorp/terraform/internal/stacks/stackruntime/testing"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

func assertEphemeralLeases(t *testing.T, leases *ephemeralLeases, operation string, expectedTotal int) {
	t.Helper()

	open, total := leases.leaseCounts()
	if total != expectedTotal {
		t.Errorf("after the %s, expected %d ephemeral resources to be opened, got %d", operation, expectedTotal, total)
	}
	if open != 0 {
		t.Errorf("after the %s, %d of %d ephemeral resources were not closed", operation, open, total)
	}
	if leases.used == 0 {
		t.Errorf("after the %s, no ephemeral value was used", operation)
	}
	leases.reset()
}

// ephemeralLeases is a helper struct for keeping track of operations performed
// on ephemeral resources and the values that they return, used within the context
// of a mock provider.
type ephemeralLeases struct {
	mu   sync.Mutex
	open map[string]bool
	used int
}

func (l *ephemeralLeases) openNew() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	value := fmt.Sprintf("secret-%d", len(l.open)+1)
	l.open[value] = true
	return value
}

func (l *ephemeralLeases) close(value string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.open[value] = false
}

func (l *ephemeralLeases) use(value string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.open[value] {
		return fmt.Errorf("%q was used after its ephemeral resource was closed", value)
	}
	l.used++
	return nil
}

func (l *ephemeralLeases) reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.open = make(map[string]bool, 0)
	l.used = 0
}

func (l *ephemeralLeases) leaseCounts() (open, total int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, isOpen := range l.open {
		if isOpen {
			open++
		}
	}
	return open, len(l.open)
}

// track will wrap a mock provider's RPC calls and will keep track of ephemeral values
// that are opened, closed, and used (to ensure the ephemeral resource is not being used after being closed).
func (l *ephemeralLeases) track(provider *stacks_testing_provider.MockProvider) {
	var credential string
	checkEph := func() (diags tfdiags.Diagnostics) {
		if credential != "" {
			diags = diags.Append(l.use(credential))
		}
		return diags
	}

	configureFn := provider.ConfigureProviderFn
	provider.ConfigureProviderFn = func(req providers.ConfigureProviderRequest) providers.ConfigureProviderResponse {
		if authn := req.Config.GetAttr("authentication"); authn.IsKnown() && !authn.IsNull() {
			credential = authn.AsString()
			if diags := checkEph(); diags.HasErrors() {
				return providers.ConfigureProviderResponse{Diagnostics: diags}
			}
			// Ephemeral values are unique when using [ephemeralLeases] and we don't really
			// care about the authentication check here, so we'll just force it to pass.
			provider.Authentication = credential
		}
		return configureFn(req)
	}

	planFn := provider.PlanResourceChangeFn
	provider.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		if diags := checkEph(); diags.HasErrors() {
			return providers.PlanResourceChangeResponse{Diagnostics: diags}
		}
		return planFn(req)
	}

	applyFn := provider.ApplyResourceChangeFn
	provider.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
		if diags := checkEph(); diags.HasErrors() {
			return providers.ApplyResourceChangeResponse{Diagnostics: diags}
		}
		return applyFn(req)
	}

	provider.OpenEphemeralResourceFn = func(req providers.OpenEphemeralResourceRequest) (resp providers.OpenEphemeralResourceResponse) {
		if resp.Diagnostics = checkEph(); resp.Diagnostics.HasErrors() {
			return resp
		}
		value := l.openNew()
		resp.Result = cty.ObjectVal(map[string]cty.Value{"value": cty.StringVal(value)})
		resp.Private = []byte(value)
		return resp
	}

	provider.CloseEphemeralResourceFn = func(req providers.CloseEphemeralResourceRequest) (resp providers.CloseEphemeralResourceResponse) {
		resp.Diagnostics = checkEph()
		if provider.CloseCalled {
			resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("ephemeral resource closed using a provider that was already terminated"))
		}
		l.close(string(req.Private))
		return resp
	}
}

func newEphemeralLeaseContext(t *testing.T, bundle string, leases *ephemeralLeases) TestContext {
	t.Helper()

	lock := depsfile.NewLocks()
	lock.SetProvider(
		addrs.NewDefaultProvider("testing"),
		providerreqs.MustParseVersion("0.1.0"),
		providerreqs.MustParseVersionConstraints("0.1.0"),
		providerreqs.PreferredHashes([]providerreqs.Hash{}),
	)
	return TestContext{
		config: loadMainBundleConfigForTest(t, bundle),
		providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
				provider := stacks_testing_provider.NewProvider(t)
				leases.track(provider)
				return provider, nil
			},
		},
		dependencyLocks: *lock,
	}
}
