package stackeval

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
)

// Main is the central node of all data required for performing the major
// actions against a stack: validation, planning, and applying.
//
// This type delegates to various other types in this package to implement
// the real logic, with Main focused on enabling the collaboration between
// objects of those other types.
type Main struct {
	config *stackconfig.Config

	// planning captures the data needed when creating or applying a plan,
	// but which need not be populated when only using the validation-related
	// functionality of this package.
	planning *mainPlanning

	// The remaining fields memoize other objects we might create in response
	// to method calls. Must lock "mu" before interacting with them.
	mu              sync.Mutex
	mainStackConfig *StackConfig
}

var _ namedPromiseReporter = (*Main)(nil)

type mainPlanning struct {
	opts PlanOpts
}

type mainApplying struct {
	opts ApplyOpts
}

// MainStackConfig returns the [StackConfig] object representing the main
// stack, which is at the root of the configuration tree.
func (m *Main) MainStackConfig(ctx context.Context) *StackConfig {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.mainStackConfig == nil {
		m.mainStackConfig = newStackConfig(m, stackaddrs.RootStack, m.config.Root)
	}
	return m.mainStackConfig
}

// StackConfig returns the [StackConfig] object representing the stack with
// the given address, or nil if there is no such stack.
func (m *Main) StackConfig(ctx context.Context, addr stackaddrs.Stack) *StackConfig {
	ret := m.MainStackConfig(ctx)
	for _, step := range addr {
		ret = ret.ChildConfig(ctx, step)
		if ret == nil {
			return nil
		}
	}
	return ret
}

// mustStackConfig is like [Main.StackConfig] except that it panics if it
// does not find a stack configuration object matching the given address,
// for situations where the absense of a stack config represents a bug
// somewhere in Terraform, rather than incorrect user input.
func (m *Main) mustStackConfig(ctx context.Context, addr stackaddrs.Stack) *StackConfig {
	ret := m.StackConfig(ctx, addr)
	if ret == nil {
		panic(fmt.Sprintf("no configuration for %s", addr))
	}
	return ret
}

// StackCallConfig returns the [StackCallConfig] object representing the
// "stack" block in the configuration with the given address, or nil if there
// is no such block.
func (m *Main) StackCallConfig(ctx context.Context, addr stackaddrs.ConfigStackCall) *StackCallConfig {
	caller := m.StackConfig(ctx, addr.Stack)
	if caller == nil {
		return nil
	}
	return caller.StackCall(ctx, addr.Item)
}

// reportNamedPromises implements namedPromiseReporter.
func (m *Main) reportNamedPromises(cb func(id promising.PromiseID, name string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.mainStackConfig != nil {
		m.mainStackConfig.reportNamedPromises(cb)
	}
}
