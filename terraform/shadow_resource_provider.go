package terraform

import (
	"fmt"
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

	// CloseShadow should be called when the _real_ side is complete.
	// This will immediately end any blocked calls and return any errors.
	//
	// Any operations on the shadow provider after this is undefined. It
	// could be fine, it could result in crashes, etc. Do not use the
	// shadow after this is called.
	CloseShadow() error
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
	result, err := p.ResourceProvider.Input(input, c)
	p.Shared.Input.SetValue(&shadowResourceProviderInput{
		Config:    c.DeepCopy(),
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
	err := p.ResourceProvider.Configure(c)
	p.Shared.Configure.SetValue(&shadowResourceProviderConfigure{
		Config: c.DeepCopy(),
		Result: err,
	})

	return err
}

func (p *shadowResourceProviderReal) Apply(
	info *InstanceInfo,
	state *InstanceState,
	diff *InstanceDiff) (*InstanceState, error) {
	result, err := p.ResourceProvider.Apply(info, state, diff)
	p.Shared.Apply.SetValue(info.HumanId(), &shadowResourceProviderApply{
		State:     state,
		Diff:      diff,
		Result:    result,
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
	CloseErr  shadow.Value
	Input     shadow.Value
	Validate  shadow.Value
	Configure shadow.Value
	Apply     shadow.KeyedValue
}

func (p *shadowResourceProviderShadow) CloseShadow() error {
	// For now, just return the error. What we need to do in the future
	// is marked the provider as "closed" so that any subsequent calls
	// will fail out.
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

func (p *shadowResourceProviderShadow) Apply(
	info *InstanceInfo,
	state *InstanceState,
	diff *InstanceDiff) (*InstanceState, error) {
	// Unique key
	key := info.HumanId()
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
			"Unknown 'apply' shadow value: %#v", raw))
		return nil, nil
	}

	// Compare the parameters, which should be identical
	if !state.Equal(result.State) {
		p.ErrorLock.Lock()
		p.Error = multierror.Append(p.Error, fmt.Errorf(
			"State had unequal states (real, then shadow):\n\n%#v\n\n%#v",
			result.State, state))
		p.ErrorLock.Unlock()
	}

	// TODO: compare diffs

	return result.Result, result.ResultErr
}

// TODO
// TODO
// TODO
// TODO
// TODO

func (p *shadowResourceProviderShadow) ValidateResource(t string, c *ResourceConfig) ([]string, []error) {
	return nil, nil
}

func (p *shadowResourceProviderShadow) Diff(
	info *InstanceInfo,
	state *InstanceState,
	desired *ResourceConfig) (*InstanceDiff, error) {
	return nil, nil
}

func (p *shadowResourceProviderShadow) Refresh(
	info *InstanceInfo,
	s *InstanceState) (*InstanceState, error) {
	return nil, nil
}

func (p *shadowResourceProviderShadow) ImportState(info *InstanceInfo, id string) ([]*InstanceState, error) {
	return nil, nil
}

func (p *shadowResourceProviderShadow) ValidateDataSource(t string, c *ResourceConfig) ([]string, []error) {
	return nil, nil
}

func (p *shadowResourceProviderShadow) ReadDataDiff(
	info *InstanceInfo,
	desired *ResourceConfig) (*InstanceDiff, error) {
	return nil, nil
}

func (p *shadowResourceProviderShadow) ReadDataApply(
	info *InstanceInfo,
	d *InstanceDiff) (*InstanceState, error) {
	return nil, nil
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

type shadowResourceProviderApply struct {
	State     *InstanceState
	Diff      *InstanceDiff
	Result    *InstanceState
	ResultErr error
}
