package stackeval

//lint:file-ignore U1000 This package is still WIP so not everything is here yet.

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

	// validating captures the data needed when validating configuration,
	// in which case we consider _only_ the configuration and don't take
	// into account any existing state or specific input variable values.
	validating *mainValidating

	// planning captures the data needed when creating or applying a plan,
	// but which need not be populated when only using the validation-related
	// functionality of this package.
	planning *mainPlanning

	// applying captures the data needed when applying a plan. This can
	// only be populated if "planning" is also populated, because the process
	// of applying includes re-evaluation of plans based on new information
	// gathered during the apply process.
	applying *mainApplying

	// The remaining fields memoize other objects we might create in response
	// to method calls. Must lock "mu" before interacting with them.
	mu              sync.Mutex
	mainStackConfig *StackConfig
}

var _ namedPromiseReporter = (*Main)(nil)

type mainValidating struct {
	opts ValidateOpts
}

type mainPlanning struct {
	opts PlanOpts
}

type mainApplying struct {
	opts ApplyOpts
}

func NewForValidating(config *stackconfig.Config, opts ValidateOpts) *Main {
	return &Main{
		config: config,
		validating: &mainValidating{
			opts: opts,
		},
	}
}

func NewForPlanning(config *stackconfig.Config, opts PlanOpts) *Main {
	// TODO: This function should also take an optional prior state, but we
	// don't actually have a type for that yet.

	return &Main{
		config: config,
		planning: &mainPlanning{
			opts: opts,
		},
	}
}

// Validating returns true if the receiving [Main] is configured for validating.
//
// If this returns false then validation methods may panic or return strange
// results.
func (m *Main) Validating() bool {
	return m.validating != nil
}

// Planning returns true if the receiving [Main] is configured for planning.
//
// If this returns false then planning methods may panic or return strange
// results.
func (m *Main) Planning() bool {
	return m.planning != nil
}

// Applying returns true if the receiving [Main] is configured for applying.
//
// If this returns false then applying methods may panic or return strange
// results.
func (m *Main) Applying() bool {
	return m.applying != nil && m.Planning()
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
