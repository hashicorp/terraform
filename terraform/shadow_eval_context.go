package terraform

import (
	"errors"
	"fmt"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/helper/shadow"
)

// ShadowEvalContext is an EvalContext that is used to "shadow" a real
// eval context for comparing whether two separate graph executions result
// in the same output.
//
// This eval context will never communicate with a real provider and will
// never modify real state.
type ShadowEvalContext interface {
	EvalContext

	// Close should be called when the _real_ EvalContext operations
	// are complete. This will immediately end any blocks calls and record
	// any errors.
	//
	// The returned error is the result of the shadow run. If it is nil,
	// then the shadow run seemingly completed successfully. You should
	// still compare the resulting states, diffs from both the real and shadow
	// contexts to verify equivalent end state.
	//
	// If the error is non-nil, then an error occurred during the execution
	// itself. In this scenario, you should not compare diffs/states since
	// they can't be considered accurate since operations during execution
	// failed.
	CloseShadow() error
}

// NewShadowEvalContext creates a new shadowed EvalContext. This returns
// the real EvalContext that should be used with the real evaluation and
// will communicate with real providers and write real state as well as
// the ShadowEvalContext that should be used with the test graph.
//
// This should be called before the ctx is ever used in order to ensure
// a consistent shadow state.
func NewShadowEvalContext(ctx EvalContext) (EvalContext, ShadowEvalContext) {
	var shared shadowEvalContextShared
	real := &shadowEvalContextReal{
		EvalContext: ctx,
		Shared:      &shared,
	}

	// Copy the diff. We do this using some weird scoping so that the
	// "diff" (real) value never leaks out and can be used.
	var diffCopy *Diff
	{
		diff, lock := ctx.Diff()
		if lock != nil {
			lock.RLock()
			diffCopy = diff
			// TODO: diffCopy = diff.DeepCopy()
			lock.RUnlock()
		}
	}

	// Copy the state. We do this using some weird scoping so that the
	// "state" (real) value never leaks out and can be used.
	var stateCopy *State
	{
		state, lock := ctx.State()
		if lock != nil {
			lock.RLock()
			stateCopy = state.DeepCopy()
			lock.RUnlock()
		}
	}

	// Build the shadow copy. For safety, we don't even give the shadow
	// copy a reference to the real context. This means that it would be
	// very difficult (impossible without some real obvious mistakes) for
	// the shadow context to do "real" work.
	shadow := &shadowEvalContextShadow{
		Shared: &shared,

		PathValue:  ctx.Path(),
		StateValue: stateCopy,
		StateLock:  new(sync.RWMutex),
		DiffValue:  diffCopy,
		DiffLock:   new(sync.RWMutex),
	}

	return real, shadow
}

var (
	// errShadow is the error returned by the shadow context when
	// things go wrong. This should be ignored and the error result from
	// Close should be checked instead since that'll contain more detailed
	// error.
	errShadow = errors.New("shadow error")
)

// shadowEvalContextReal is the EvalContext that does real work.
type shadowEvalContextReal struct {
	EvalContext

	Shared *shadowEvalContextShared
}

func (c *shadowEvalContextReal) InitProvider(n string) (ResourceProvider, error) {
	// Initialize the real provider
	p, err := c.EvalContext.InitProvider(n)

	// Create the shadow
	var real ResourceProvider
	var shadow shadowResourceProvider
	if err == nil {
		real, shadow = newShadowResourceProvider(p)
	}

	// Store the result
	c.Shared.Providers.SetValue(n, &shadowEvalContextInitProvider{
		Shadow:    shadow,
		ResultErr: err,
	})

	return real, err
}

// shadowEvalContextShadow is the EvalContext that shadows the real one
// and leans on that for data.
type shadowEvalContextShadow struct {
	Shared *shadowEvalContextShared

	PathValue  []string
	Providers  map[string]ResourceProvider
	DiffValue  *Diff
	DiffLock   *sync.RWMutex
	StateValue *State
	StateLock  *sync.RWMutex

	// The collection of errors that were found during the shadow run
	Error     error
	ErrorLock sync.Mutex

	// Fields relating to closing the context. Closing signals that
	// the execution of the real context completed.
	closeLock sync.Mutex
	closed    bool
	closeCh   chan struct{}
}

// Shared is the shared state between the shadow and real contexts when
// a shadow context is active. This is used by the real context to setup
// some state, trigger condition variables, etc.
type shadowEvalContextShared struct {
	Providers shadow.KeyedValue
}

func (c *shadowEvalContextShadow) CloseShadow() error {
	// TODO: somehow shut this thing down
	return c.Error
}

func (c *shadowEvalContextShadow) Path() []string {
	return c.PathValue
}

func (c *shadowEvalContextShadow) Hook(f func(Hook) (HookAction, error)) error {
	// Don't do anything on hooks. Mission critical behavior should not
	// depend on hooks and at the time of writing it does not depend on
	// hooks. In the future we could also test hooks but not now.
	return nil
}

func (c *shadowEvalContextShadow) InitProvider(n string) (ResourceProvider, error) {
	// Wait for the provider value
	raw := c.Shared.Providers.Value(n)
	if raw == nil {
		return nil, c.err(fmt.Errorf(
			"Unknown 'InitProvider' call for %q", n))
	}

	result, ok := raw.(*shadowEvalContextInitProvider)
	if !ok {
		return nil, c.err(fmt.Errorf(
			"Unknown 'InitProvider' shadow value: %#v", raw))
	}

	result.Lock()
	defer result.Unlock()

	if result.Init {
		// Record the error but continue...
		c.err(fmt.Errorf(
			"InitProvider: provider %q already initialized", n))
	}

	result.Init = true
	return result.Shadow, result.ResultErr
}

func (c *shadowEvalContextShadow) Provider(n string) ResourceProvider {
	// Wait for the provider value
	raw := c.Shared.Providers.Value(n)
	if raw == nil {
		c.err(fmt.Errorf(
			"Unknown 'Provider' call for %q", n))
		return nil
	}

	result, ok := raw.(*shadowEvalContextInitProvider)
	if !ok {
		c.err(fmt.Errorf(
			"Unknown 'Provider' shadow value: %#v", raw))
		return nil
	}

	result.Lock()
	defer result.Unlock()

	if !result.Init {
		// Record the error but continue...
		c.err(fmt.Errorf(
			"Provider: provider %q requested but not initialized", n))
	}

	return result.Shadow
}

func (c *shadowEvalContextShadow) CloseProvider(n string) error {
	// Wait for the provider value
	raw := c.Shared.Providers.Value(n)
	if raw == nil {
		c.err(fmt.Errorf(
			"Unknown 'CloseProvider' call for %q", n))
		return nil
	}

	result, ok := raw.(*shadowEvalContextInitProvider)
	if !ok {
		c.err(fmt.Errorf(
			"Unknown 'CloseProvider' shadow value: %#v", raw))
		return nil
	}

	result.Lock()
	defer result.Unlock()

	if !result.Init {
		// Record the error but continue...
		c.err(fmt.Errorf(
			"CloseProvider: provider %q requested but not initialized", n))
	} else if result.Closed {
		c.err(fmt.Errorf(
			"CloseProvider: provider %q requested but already closed", n))
	}

	result.Closed = true
	return nil
}

func (c *shadowEvalContextShadow) Diff() (*Diff, *sync.RWMutex) {
	return c.DiffValue, c.DiffLock
}

func (c *shadowEvalContextShadow) State() (*State, *sync.RWMutex) {
	return c.StateValue, c.StateLock
}

func (c *shadowEvalContextShadow) err(err error) error {
	c.ErrorLock.Lock()
	defer c.ErrorLock.Unlock()
	c.Error = multierror.Append(c.Error, err)
	return err
}

// TODO: All the functions below are EvalContext functions that must be impl.

func (c *shadowEvalContextShadow) Input() UIInput                                  { return nil }
func (c *shadowEvalContextShadow) ConfigureProvider(string, *ResourceConfig) error { return nil }
func (c *shadowEvalContextShadow) SetProviderConfig(string, *ResourceConfig) error { return nil }
func (c *shadowEvalContextShadow) ParentProviderConfig(string) *ResourceConfig     { return nil }
func (c *shadowEvalContextShadow) ProviderInput(string) map[string]interface{}     { return nil }
func (c *shadowEvalContextShadow) SetProviderInput(string, map[string]interface{}) {}

func (c *shadowEvalContextShadow) InitProvisioner(string) (ResourceProvisioner, error) {
	return nil, nil
}
func (c *shadowEvalContextShadow) Provisioner(string) ResourceProvisioner { return nil }
func (c *shadowEvalContextShadow) CloseProvisioner(string) error          { return nil }

func (c *shadowEvalContextShadow) Interpolate(*config.RawConfig, *Resource) (*ResourceConfig, error) {
	return nil, nil
}
func (c *shadowEvalContextShadow) SetVariables(string, map[string]interface{}) {}

// The structs for the various function calls are put below. These structs
// are used to carry call information across the real/shadow boundaries.

type shadowEvalContextInitProvider struct {
	Shadow    shadowResourceProvider
	ResultErr error

	sync.Mutex      // Must be held to modify the field below
	Init       bool // Keeps track of whether it has been initialized in the shadow
	Closed     bool // Keeps track of whether this provider is closed
}
