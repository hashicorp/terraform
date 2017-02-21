package terraform

import (
	"fmt"
	"log"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/helper/shadow"
)

// shadowResourceProvider implements ResourceProvider for the shadow
// eval context defined in eval_context_shadow.go.
//
// This is used to verify behavior with a real provider. This shouldn't
// be used directly.
type shadowResourceProvider interface {
	ResourceProvider
	Shadow
}

// newShadowResourceProvider creates a new shadowed ResourceProvider.
//
// This will assume a well behaved real ResourceProvider. For example,
// it assumes that the `Resources` call underneath doesn't change values
// since once it is called on the real provider, it will be cached and
// returned in the shadow since number of calls to that shouldn't affect
// actual behavior.
//
// However, with calls like Apply, call order is taken into account,
// parameters are checked for equality, etc.
func newShadowResourceProvider(p ResourceProvider) (ResourceProvider, shadowResourceProvider) {
	// Create the shared data
	shared := shadowResourceProviderShared{}

	// Create the real provider that does actual work
	real := &shadowResourceProviderReal{
		ResourceProvider: p,
		Shared:           &shared,
	}

	// Create the shadow that watches the real value
	shadow := &shadowResourceProviderShadow{
		Shared: &shared,

		resources:   p.Resources(),
		dataSources: p.DataSources(),
	}

	return real, shadow
}

// shadowResourceProviderReal is the real resource provider. Function calls
// to this will perform real work. This records the parameters and return
// values and call order for the shadow to reproduce.
type shadowResourceProviderReal struct {
	ResourceProvider

	Shared *shadowResourceProviderShared
}

func (p *shadowResourceProviderReal) Close() error {
	var result error
	if c, ok := p.ResourceProvider.(ResourceProviderCloser); ok {
		result = c.Close()
	}

	p.Shared.CloseErr.SetValue(result)
	return result
}

func (p *shadowResourceProviderReal) Input(
	input UIInput, c *ResourceConfig) (*ResourceConfig, error) {
	cCopy := c.DeepCopy()

	result, err := p.ResourceProvider.Input(input, c)
	p.Shared.Input.SetValue(&shadowResourceProviderInput{
		Config:    cCopy,
		Result:    result.DeepCopy(),
		ResultErr: err,
	})

	return result, err
}

func (p *shadowResourceProviderReal) Validate(c *ResourceConfig) ([]string, []error) {
	warns, errs := p.ResourceProvider.Validate(c)
	p.Shared.Validate.SetValue(&shadowResourceProviderValidate{
		Config:     c.DeepCopy(),
		ResultWarn: warns,
		ResultErr:  errs,
	})

	return warns, errs
}

func (p *shadowResourceProviderReal) Configure(c *ResourceConfig) error {
	cCopy := c.DeepCopy()

	err := p.ResourceProvider.Configure(c)
	p.Shared.Configure.SetValue(&shadowResourceProviderConfigure{
		Config: cCopy,
		Result: err,
	})

	return err
}

func (p *shadowResourceProviderReal) Stop() error {
	return p.ResourceProvider.Stop()
}

func (p *shadowResourceProviderReal) ValidateResource(
	t string, c *ResourceConfig) ([]string, []error) {
	key := t
	configCopy := c.DeepCopy()

	// Real operation
	warns, errs := p.ResourceProvider.ValidateResource(t, c)

	// Initialize to ensure we always have a wrapper with a lock
	p.Shared.ValidateResource.Init(
		key, &shadowResourceProviderValidateResourceWrapper{})

	// Get the result
	raw := p.Shared.ValidateResource.Value(key)
	wrapper, ok := raw.(*shadowResourceProviderValidateResourceWrapper)
	if !ok {
		// If this fails then we just continue with our day... the shadow
		// will fail to but there isn't much we can do.
		log.Printf(
			"[ERROR] unknown value in ValidateResource shadow value: %#v", raw)
		return warns, errs
	}

	// Lock the wrapper for writing and record our call
	wrapper.Lock()
	defer wrapper.Unlock()

	wrapper.Calls = append(wrapper.Calls, &shadowResourceProviderValidateResource{
		Config: configCopy,
		Warns:  warns,
		Errors: errs,
	})

	// With it locked, call SetValue again so that it triggers WaitForChange
	p.Shared.ValidateResource.SetValue(key, wrapper)

	// Return the result
	return warns, errs
}

func (p *shadowResourceProviderReal) Apply(
	info *InstanceInfo,
	state *InstanceState,
	diff *InstanceDiff) (*InstanceState, error) {
	// Thse have to be copied before the call since call can modify
	stateCopy := state.DeepCopy()
	diffCopy := diff.DeepCopy()

	result, err := p.ResourceProvider.Apply(info, state, diff)
	p.Shared.Apply.SetValue(info.uniqueId(), &shadowResourceProviderApply{
		State:     stateCopy,
		Diff:      diffCopy,
		Result:    result.DeepCopy(),
		ResultErr: err,
	})

	return result, err
}

func (p *shadowResourceProviderReal) Diff(
	info *InstanceInfo,
	state *InstanceState,
	desired *ResourceConfig) (*InstanceDiff, error) {
	// Thse have to be copied before the call since call can modify
	stateCopy := state.DeepCopy()
	desiredCopy := desired.DeepCopy()

	result, err := p.ResourceProvider.Diff(info, state, desired)
	p.Shared.Diff.SetValue(info.uniqueId(), &shadowResourceProviderDiff{
		State:     stateCopy,
		Desired:   desiredCopy,
		Result:    result.DeepCopy(),
		ResultErr: err,
	})

	return result, err
}

func (p *shadowResourceProviderReal) Refresh(
	info *InstanceInfo,
	state *InstanceState) (*InstanceState, error) {
	// Thse have to be copied before the call since call can modify
	stateCopy := state.DeepCopy()

	result, err := p.ResourceProvider.Refresh(info, state)
	p.Shared.Refresh.SetValue(info.uniqueId(), &shadowResourceProviderRefresh{
		State:     stateCopy,
		Result:    result.DeepCopy(),
		ResultErr: err,
	})

	return result, err
}

func (p *shadowResourceProviderReal) ValidateDataSource(
	t string, c *ResourceConfig) ([]string, []error) {
	key := t
	configCopy := c.DeepCopy()

	// Real operation
	warns, errs := p.ResourceProvider.ValidateDataSource(t, c)

	// Initialize
	p.Shared.ValidateDataSource.Init(
		key, &shadowResourceProviderValidateDataSourceWrapper{})

	// Get the result
	raw := p.Shared.ValidateDataSource.Value(key)
	wrapper, ok := raw.(*shadowResourceProviderValidateDataSourceWrapper)
	if !ok {
		// If this fails then we just continue with our day... the shadow
		// will fail to but there isn't much we can do.
		log.Printf(
			"[ERROR] unknown value in ValidateDataSource shadow value: %#v", raw)
		return warns, errs
	}

	// Lock the wrapper for writing and record our call
	wrapper.Lock()
	defer wrapper.Unlock()

	wrapper.Calls = append(wrapper.Calls, &shadowResourceProviderValidateDataSource{
		Config: configCopy,
		Warns:  warns,
		Errors: errs,
	})

	// Set it
	p.Shared.ValidateDataSource.SetValue(key, wrapper)

	// Return the result
	return warns, errs
}

func (p *shadowResourceProviderReal) ReadDataDiff(
	info *InstanceInfo,
	desired *ResourceConfig) (*InstanceDiff, error) {
	// These have to be copied before the call since call can modify
	desiredCopy := desired.DeepCopy()

	result, err := p.ResourceProvider.ReadDataDiff(info, desired)
	p.Shared.ReadDataDiff.SetValue(info.uniqueId(), &shadowResourceProviderReadDataDiff{
		Desired:   desiredCopy,
		Result:    result.DeepCopy(),
		ResultErr: err,
	})

	return result, err
}

func (p *shadowResourceProviderReal) ReadDataApply(
	info *InstanceInfo,
	diff *InstanceDiff) (*InstanceState, error) {
	// Thse have to be copied before the call since call can modify
	diffCopy := diff.DeepCopy()

	result, err := p.ResourceProvider.ReadDataApply(info, diff)
	p.Shared.ReadDataApply.SetValue(info.uniqueId(), &shadowResourceProviderReadDataApply{
		Diff:      diffCopy,
		Result:    result.DeepCopy(),
		ResultErr: err,
	})

	return result, err
}

// shadowResourceProviderShadow is the shadow resource provider. Function
// calls never affect real resources. This is paired with the "real" side
// which must be called properly to enable recording.
type shadowResourceProviderShadow struct {
	Shared *shadowResourceProviderShared

	// Cached values that are expected to not change
	resources   []ResourceType
	dataSources []DataSource

	Error     error // Error is the list of errors from the shadow
	ErrorLock sync.Mutex
}

type shadowResourceProviderShared struct {
	// NOTE: Anytime a value is added here, be sure to add it to
	// the Close() method so that it is closed.

	CloseErr           shadow.Value
	Input              shadow.Value
	Validate           shadow.Value
	Configure          shadow.Value
	ValidateResource   shadow.KeyedValue
	Apply              shadow.KeyedValue
	Diff               shadow.KeyedValue
	Refresh            shadow.KeyedValue
	ValidateDataSource shadow.KeyedValue
	ReadDataDiff       shadow.KeyedValue
	ReadDataApply      shadow.KeyedValue
}

func (p *shadowResourceProviderShared) Close() error {
	return shadow.Close(p)
}

func (p *shadowResourceProviderShadow) CloseShadow() error {
	err := p.Shared.Close()
	if err != nil {
		err = fmt.Errorf("close error: %s", err)
	}

	return err
}

func (p *shadowResourceProviderShadow) ShadowError() error {
	return p.Error
}

func (p *shadowResourceProviderShadow) Resources() []ResourceType {
	return p.resources
}

func (p *shadowResourceProviderShadow) DataSources() []DataSource {
	return p.dataSources
}

func (p *shadowResourceProviderShadow) Close() error {
	v := p.Shared.CloseErr.Value()
	if v == nil {
		return nil
	}

	return v.(error)
}

func (p *shadowResourceProviderShadow) Input(
	input UIInput, c *ResourceConfig) (*ResourceConfig, error) {
	// Get the result of the input call
	raw := p.Shared.Input.Value()
	if raw == nil {
		return nil, nil
	}

	result, ok := raw.(*shadowResourceProviderInput)
	if !ok {
		p.ErrorLock.Lock()
		defer p.ErrorLock.Unlock()
		p.Error = multierror.Append(p.Error, fmt.Errorf(
			"Unknown 'input' shadow value: %#v", raw))
		return nil, nil
	}

	// Compare the parameters, which should be identical
	if !c.Equal(result.Config) {
		p.ErrorLock.Lock()
		p.Error = multierror.Append(p.Error, fmt.Errorf(
			"Input had unequal configurations (real, then shadow):\n\n%#v\n\n%#v",
			result.Config, c))
		p.ErrorLock.Unlock()
	}

	// Return the results
	return result.Result, result.ResultErr
}

func (p *shadowResourceProviderShadow) Validate(c *ResourceConfig) ([]string, []error) {
	// Get the result of the validate call
	raw := p.Shared.Validate.Value()
	if raw == nil {
		return nil, nil
	}

	result, ok := raw.(*shadowResourceProviderValidate)
	if !ok {
		p.ErrorLock.Lock()
		defer p.ErrorLock.Unlock()
		p.Error = multierror.Append(p.Error, fmt.Errorf(
			"Unknown 'validate' shadow value: %#v", raw))
		return nil, nil
	}

	// Compare the parameters, which should be identical
	if !c.Equal(result.Config) {
		p.ErrorLock.Lock()
		p.Error = multierror.Append(p.Error, fmt.Errorf(
			"Validate had unequal configurations (real, then shadow):\n\n%#v\n\n%#v",
			result.Config, c))
		p.ErrorLock.Unlock()
	}

	// Return the results
	return result.ResultWarn, result.ResultErr
}

func (p *shadowResourceProviderShadow) Configure(c *ResourceConfig) error {
	// Get the result of the call
	raw := p.Shared.Configure.Value()
	if raw == nil {
		return nil
	}

	result, ok := raw.(*shadowResourceProviderConfigure)
	if !ok {
		p.ErrorLock.Lock()
		defer p.ErrorLock.Unlock()
		p.Error = multierror.Append(p.Error, fmt.Errorf(
			"Unknown 'configure' shadow value: %#v", raw))
		return nil
	}

	// Compare the parameters, which should be identical
	if !c.Equal(result.Config) {
		p.ErrorLock.Lock()
		p.Error = multierror.Append(p.Error, fmt.Errorf(
			"Configure had unequal configurations (real, then shadow):\n\n%#v\n\n%#v",
			result.Config, c))
		p.ErrorLock.Unlock()
	}

	// Return the results
	return result.Result
}

// Stop returns immediately.
func (p *shadowResourceProviderShadow) Stop() error {
	return nil
}

func (p *shadowResourceProviderShadow) ValidateResource(t string, c *ResourceConfig) ([]string, []error) {
	// Unique key
	key := t

	// Get the initial value
	raw := p.Shared.ValidateResource.Value(key)

	// Find a validation with our configuration
	var result *shadowResourceProviderValidateResource
	for {
		// Get the value
		if raw == nil {
			p.ErrorLock.Lock()
			defer p.ErrorLock.Unlock()
			p.Error = multierror.Append(p.Error, fmt.Errorf(
				"Unknown 'ValidateResource' call for %q:\n\n%#v",
				key, c))
			return nil, nil
		}

		wrapper, ok := raw.(*shadowResourceProviderValidateResourceWrapper)
		if !ok {
			p.ErrorLock.Lock()
			defer p.ErrorLock.Unlock()
			p.Error = multierror.Append(p.Error, fmt.Errorf(
				"Unknown 'ValidateResource' shadow value for %q: %#v", key, raw))
			return nil, nil
		}

		// Look for the matching call with our configuration
		wrapper.RLock()
		for _, call := range wrapper.Calls {
			if call.Config.Equal(c) {
				result = call
				break
			}
		}
		wrapper.RUnlock()

		// If we found a result, exit
		if result != nil {
			break
		}

		// Wait for a change so we can get the wrapper again
		raw = p.Shared.ValidateResource.WaitForChange(key)
	}

	return result.Warns, result.Errors
}

func (p *shadowResourceProviderShadow) Apply(
	info *InstanceInfo,
	state *InstanceState,
	diff *InstanceDiff) (*InstanceState, error) {
	// Unique key
	key := info.uniqueId()
	raw := p.Shared.Apply.Value(key)
	if raw == nil {
		p.ErrorLock.Lock()
		defer p.ErrorLock.Unlock()
		p.Error = multierror.Append(p.Error, fmt.Errorf(
			"Unknown 'apply' call for %q:\n\n%#v\n\n%#v",
			key, state, diff))
		return nil, nil
	}

	result, ok := raw.(*shadowResourceProviderApply)
	if !ok {
		p.ErrorLock.Lock()
		defer p.ErrorLock.Unlock()
		p.Error = multierror.Append(p.Error, fmt.Errorf(
			"Unknown 'apply' shadow value for %q: %#v", key, raw))
		return nil, nil
	}

	// Compare the parameters, which should be identical
	if !state.Equal(result.State) {
		p.ErrorLock.Lock()
		p.Error = multierror.Append(p.Error, fmt.Errorf(
			"Apply %q: state had unequal states (real, then shadow):\n\n%#v\n\n%#v",
			key, result.State, state))
		p.ErrorLock.Unlock()
	}

	if !diff.Equal(result.Diff) {
		p.ErrorLock.Lock()
		p.Error = multierror.Append(p.Error, fmt.Errorf(
			"Apply %q: unequal diffs (real, then shadow):\n\n%#v\n\n%#v",
			key, result.Diff, diff))
		p.ErrorLock.Unlock()
	}

	return result.Result, result.ResultErr
}

func (p *shadowResourceProviderShadow) Diff(
	info *InstanceInfo,
	state *InstanceState,
	desired *ResourceConfig) (*InstanceDiff, error) {
	// Unique key
	key := info.uniqueId()
	raw := p.Shared.Diff.Value(key)
	if raw == nil {
		p.ErrorLock.Lock()
		defer p.ErrorLock.Unlock()
		p.Error = multierror.Append(p.Error, fmt.Errorf(
			"Unknown 'diff' call for %q:\n\n%#v\n\n%#v",
			key, state, desired))
		return nil, nil
	}

	result, ok := raw.(*shadowResourceProviderDiff)
	if !ok {
		p.ErrorLock.Lock()
		defer p.ErrorLock.Unlock()
		p.Error = multierror.Append(p.Error, fmt.Errorf(
			"Unknown 'diff' shadow value for %q: %#v", key, raw))
		return nil, nil
	}

	// Compare the parameters, which should be identical
	if !state.Equal(result.State) {
		p.ErrorLock.Lock()
		p.Error = multierror.Append(p.Error, fmt.Errorf(
			"Diff %q had unequal states (real, then shadow):\n\n%#v\n\n%#v",
			key, result.State, state))
		p.ErrorLock.Unlock()
	}
	if !desired.Equal(result.Desired) {
		p.ErrorLock.Lock()
		p.Error = multierror.Append(p.Error, fmt.Errorf(
			"Diff %q had unequal states (real, then shadow):\n\n%#v\n\n%#v",
			key, result.Desired, desired))
		p.ErrorLock.Unlock()
	}

	return result.Result, result.ResultErr
}

func (p *shadowResourceProviderShadow) Refresh(
	info *InstanceInfo,
	state *InstanceState) (*InstanceState, error) {
	// Unique key
	key := info.uniqueId()
	raw := p.Shared.Refresh.Value(key)
	if raw == nil {
		p.ErrorLock.Lock()
		defer p.ErrorLock.Unlock()
		p.Error = multierror.Append(p.Error, fmt.Errorf(
			"Unknown 'refresh' call for %q:\n\n%#v",
			key, state))
		return nil, nil
	}

	result, ok := raw.(*shadowResourceProviderRefresh)
	if !ok {
		p.ErrorLock.Lock()
		defer p.ErrorLock.Unlock()
		p.Error = multierror.Append(p.Error, fmt.Errorf(
			"Unknown 'refresh' shadow value: %#v", raw))
		return nil, nil
	}

	// Compare the parameters, which should be identical
	if !state.Equal(result.State) {
		p.ErrorLock.Lock()
		p.Error = multierror.Append(p.Error, fmt.Errorf(
			"Refresh %q had unequal states (real, then shadow):\n\n%#v\n\n%#v",
			key, result.State, state))
		p.ErrorLock.Unlock()
	}

	return result.Result, result.ResultErr
}

func (p *shadowResourceProviderShadow) ValidateDataSource(
	t string, c *ResourceConfig) ([]string, []error) {
	// Unique key
	key := t

	// Get the initial value
	raw := p.Shared.ValidateDataSource.Value(key)

	// Find a validation with our configuration
	var result *shadowResourceProviderValidateDataSource
	for {
		// Get the value
		if raw == nil {
			p.ErrorLock.Lock()
			defer p.ErrorLock.Unlock()
			p.Error = multierror.Append(p.Error, fmt.Errorf(
				"Unknown 'ValidateDataSource' call for %q:\n\n%#v",
				key, c))
			return nil, nil
		}

		wrapper, ok := raw.(*shadowResourceProviderValidateDataSourceWrapper)
		if !ok {
			p.ErrorLock.Lock()
			defer p.ErrorLock.Unlock()
			p.Error = multierror.Append(p.Error, fmt.Errorf(
				"Unknown 'ValidateDataSource' shadow value: %#v", raw))
			return nil, nil
		}

		// Look for the matching call with our configuration
		wrapper.RLock()
		for _, call := range wrapper.Calls {
			if call.Config.Equal(c) {
				result = call
				break
			}
		}
		wrapper.RUnlock()

		// If we found a result, exit
		if result != nil {
			break
		}

		// Wait for a change so we can get the wrapper again
		raw = p.Shared.ValidateDataSource.WaitForChange(key)
	}

	return result.Warns, result.Errors
}

func (p *shadowResourceProviderShadow) ReadDataDiff(
	info *InstanceInfo,
	desired *ResourceConfig) (*InstanceDiff, error) {
	// Unique key
	key := info.uniqueId()
	raw := p.Shared.ReadDataDiff.Value(key)
	if raw == nil {
		p.ErrorLock.Lock()
		defer p.ErrorLock.Unlock()
		p.Error = multierror.Append(p.Error, fmt.Errorf(
			"Unknown 'ReadDataDiff' call for %q:\n\n%#v",
			key, desired))
		return nil, nil
	}

	result, ok := raw.(*shadowResourceProviderReadDataDiff)
	if !ok {
		p.ErrorLock.Lock()
		defer p.ErrorLock.Unlock()
		p.Error = multierror.Append(p.Error, fmt.Errorf(
			"Unknown 'ReadDataDiff' shadow value for %q: %#v", key, raw))
		return nil, nil
	}

	// Compare the parameters, which should be identical
	if !desired.Equal(result.Desired) {
		p.ErrorLock.Lock()
		p.Error = multierror.Append(p.Error, fmt.Errorf(
			"ReadDataDiff %q had unequal configs (real, then shadow):\n\n%#v\n\n%#v",
			key, result.Desired, desired))
		p.ErrorLock.Unlock()
	}

	return result.Result, result.ResultErr
}

func (p *shadowResourceProviderShadow) ReadDataApply(
	info *InstanceInfo,
	d *InstanceDiff) (*InstanceState, error) {
	// Unique key
	key := info.uniqueId()
	raw := p.Shared.ReadDataApply.Value(key)
	if raw == nil {
		p.ErrorLock.Lock()
		defer p.ErrorLock.Unlock()
		p.Error = multierror.Append(p.Error, fmt.Errorf(
			"Unknown 'ReadDataApply' call for %q:\n\n%#v",
			key, d))
		return nil, nil
	}

	result, ok := raw.(*shadowResourceProviderReadDataApply)
	if !ok {
		p.ErrorLock.Lock()
		defer p.ErrorLock.Unlock()
		p.Error = multierror.Append(p.Error, fmt.Errorf(
			"Unknown 'ReadDataApply' shadow value for %q: %#v", key, raw))
		return nil, nil
	}

	// Compare the parameters, which should be identical
	if !d.Equal(result.Diff) {
		p.ErrorLock.Lock()
		p.Error = multierror.Append(p.Error, fmt.Errorf(
			"ReadDataApply: unequal diffs (real, then shadow):\n\n%#v\n\n%#v",
			result.Diff, d))
		p.ErrorLock.Unlock()
	}

	return result.Result, result.ResultErr
}

func (p *shadowResourceProviderShadow) ImportState(info *InstanceInfo, id string) ([]*InstanceState, error) {
	panic("import not supported by shadow graph")
}

// The structs for the various function calls are put below. These structs
// are used to carry call information across the real/shadow boundaries.

type shadowResourceProviderInput struct {
	Config    *ResourceConfig
	Result    *ResourceConfig
	ResultErr error
}

type shadowResourceProviderValidate struct {
	Config     *ResourceConfig
	ResultWarn []string
	ResultErr  []error
}

type shadowResourceProviderConfigure struct {
	Config *ResourceConfig
	Result error
}

type shadowResourceProviderValidateResourceWrapper struct {
	sync.RWMutex

	Calls []*shadowResourceProviderValidateResource
}

type shadowResourceProviderValidateResource struct {
	Config *ResourceConfig
	Warns  []string
	Errors []error
}

type shadowResourceProviderApply struct {
	State     *InstanceState
	Diff      *InstanceDiff
	Result    *InstanceState
	ResultErr error
}

type shadowResourceProviderDiff struct {
	State     *InstanceState
	Desired   *ResourceConfig
	Result    *InstanceDiff
	ResultErr error
}

type shadowResourceProviderRefresh struct {
	State     *InstanceState
	Result    *InstanceState
	ResultErr error
}

type shadowResourceProviderValidateDataSourceWrapper struct {
	sync.RWMutex

	Calls []*shadowResourceProviderValidateDataSource
}

type shadowResourceProviderValidateDataSource struct {
	Config *ResourceConfig
	Warns  []string
	Errors []error
}

type shadowResourceProviderReadDataDiff struct {
	Desired   *ResourceConfig
	Result    *InstanceDiff
	ResultErr error
}

type shadowResourceProviderReadDataApply struct {
	Diff      *InstanceDiff
	Result    *InstanceState
	ResultErr error
}
