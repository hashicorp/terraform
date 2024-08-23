// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ProviderFactories is a collection of factory functions for starting new
// instances of various providers.
type ProviderFactories map[addrs.Provider]providers.Factory

func (pf ProviderFactories) ProviderAvailable(providerAddr addrs.Provider) bool {
	_, available := pf[providerAddr]
	return available
}

// NewUnconfiguredClient launches a new instance of the requested provider,
// if available, and returns it in an unconfigured state.
//
// Callers that need a _configured_ provider can then call
// [providers.Interface.Configure] on the result to configure it, making it
// ready for the majority of operations that require a configured provider.
func (pf ProviderFactories) NewUnconfiguredClient(providerAddr addrs.Provider) (providers.Interface, error) {
	f, ok := pf[providerAddr]
	if !ok {
		return nil, fmt.Errorf("provider is not available in this execution context")
	}
	return f()
}

// rcProviderClient is a reference-counting abstraction to help with sharing
// instances of providers between multiple callers and shutting them down
// once all callers have finished with them.
//
// It encapsulates the problem of tracking the number of callers, instantiating
// a provider when necessary, and closing that provider once there are no
// active references remaining.
type rcProviderClient struct {
	// Factory is the function to use to create a new instance of the provider
	// when needed.
	//
	// This function should perform all of the steps that ought to happen
	// exactly once before the provider becomes useful to its possibly-many
	// constituents. In particular, if multiple callers are hoping to share
	// a single _configured_ provider then the factory function must be the
	// one to configure it, so that the callers don't all need to race to
	// be the one to do the one-time configuration themselves.
	Factory providers.Factory

	// must hold mu when interacting with the other fields below
	mu sync.Mutex

	callers int
	client  providers.Interface
}

func (rcpc *rcProviderClient) Borrow(ctx context.Context) (providers.Interface, error) {
	rcpc.mu.Lock()
	defer rcpc.mu.Unlock()

	rcpc.callers++

	var client providers.Interface
	if rcpc.client != nil {
		client = rcpc.client
	} else {
		var err error
		client, err = rcpc.Factory()
		if err != nil {
			return nil, err
		}
	}

	// each caller gets its own "closed" flag captured into its "close" closure,
	// so we can silently ignore duplicate calls to "close".
	var closed atomic.Bool
	close := func() error {
		if !closed.CompareAndSwap(false, true) {
			// We silently ignore redundant calls to close.
			// We intentionally don't panic here because we want to encourage
			// callers to call Close liberally in every possible return
			// path, rather than working hard to ensure only one call and
			// potentially ending up not calling it at all in some edge cases.
			return nil
		}

		// To reduce churn of provider clients when different callers are
		// taking turns to use them, we'll decrement the caller count immediately
		// but pause briefly before we check if the count has reached zero
		// so that another caller has a chance to acquire the same client
		// and thus avoid the overhead of starting it up again.
		rcpc.mu.Lock()
		rcpc.callers--
		rcpc.mu.Unlock()

		// We'll wait either for our anti-churn delay or until the context is
		// cancelled before we test whether we should shut down the client,
		// but that's an implementation detail the caller doesn't need to know
		// about and so we'll deal with that in a separate goroutine.
		go func() {
			// This time selection is essentially arbitrary; we're aiming to
			// find a happy compromise between using more RAM by keeping a
			// provider active a little longer vs. spending less time on
			// startup and shutdown overhead as usage of a provider passes
			// between different callers that are not necessarily synchronized
			// with one another.
			timer := time.NewTimer(1 * time.Second)
			select {
			case <-timer.C:
			case <-ctx.Done():
			}

			rcpc.mu.Lock()
			if rcpc.callers > 0 {
				rcpc.mu.Unlock()
				return // someone else requested the client in the meantime
			}
			if rcpc.client == nil {
				rcpc.mu.Unlock()
				return // someone else already closed the client in the meantime
			}

			// We'll take our own private copy of the client and nil out the
			// shared one so that we don't need to hold the mutex for the
			// entire (possibly-time-consuming) shutdown procedure.
			oldClient := rcpc.client
			rcpc.client = nil
			rcpc.mu.Unlock()
			// NOTE: MUST NOT access p.client or p.callerCount after this point

			err := oldClient.Close()
			if err != nil {
				// We don't really have any way to properly handle an error here,
				// but "Close" is typically just sending the child process a
				// kill signal and so if that fails there wouldn't be much we could
				// do to recover anyway.
				log.Printf("[ERROR] failed to shut down provider instance: %s", err)
			}
		}()

		return nil
	}

	// To honor the providers.Interface abstraction while still allowing
	// multiple callers to share a single client we wrap the real client
	// to intercept the Close method and treat it as decrementing the
	// reference count rather than closing the client directly.
	return providerClose{
		Interface: client,
		close:     close,
	}, nil
}

// providerClose is an implementation of providers.Interface that intercepts
// the "Close" operation and diverts it into a callback function. We use this
// so that multiple callers can share a single client in a coordinated way,
// so they can all call Close as normal without interfering with one another.
type providerClose struct {
	providers.Interface
	close func() error
}

var _ providers.Interface = providerClose{}

func (p providerClose) Close() error {
	return p.close()
}

func (p providerClose) ConfigureProvider(request providers.ConfigureProviderRequest) providers.ConfigureProviderResponse {
	// the real provider should either already have been configured by the time
	// we get here or should never get configured, so we should never see this
	// method called.
	return providers.ConfigureProviderResponse{
		Diagnostics: tfdiags.Diagnostics{
			tfdiags.AttributeValue(
				tfdiags.Error,
				"Called ConfigureProvider on an unconfigurable provider",
				"This provider should have already been configured, or should never be configured. This is a bug in Terraform - please report it.",
				nil, // nil attribute path means the overall configuration block
			),
		},
	}
}
