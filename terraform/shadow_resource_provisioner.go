package terraform

import (
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/helper/shadow"
)

// shadowResourceProvisioner implements ResourceProvisioner for the shadow
// eval context defined in eval_context_shadow.go.
//
// This is used to verify behavior with a real provisioner. This shouldn't
// be used directly.
type shadowResourceProvisioner interface {
	ResourceProvisioner
	Shadow
}

// newShadowResourceProvisioner creates a new shadowed ResourceProvisioner.
func newShadowResourceProvisioner(
	p ResourceProvisioner) (ResourceProvisioner, shadowResourceProvisioner) {
	// Create the shared data
	shared := shadowResourceProvisionerShared{
		Validate: shadow.ComparedValue{
			Func: shadowResourceProvisionerValidateCompare,
		},
	}

	// Create the real provisioner that does actual work
	real := &shadowResourceProvisionerReal{
		ResourceProvisioner: p,
		Shared:              &shared,
	}

	// Create the shadow that watches the real value
	shadow := &shadowResourceProvisionerShadow{
		Shared: &shared,
	}

	return real, shadow
}

// shadowResourceProvisionerReal is the real resource provisioner. Function calls
// to this will perform real work. This records the parameters and return
// values and call order for the shadow to reproduce.
type shadowResourceProvisionerReal struct {
	ResourceProvisioner

	Shared *shadowResourceProvisionerShared
}

func (p *shadowResourceProvisionerReal) Close() error {
	var result error
	if c, ok := p.ResourceProvisioner.(ResourceProvisionerCloser); ok {
		result = c.Close()
	}

	p.Shared.CloseErr.SetValue(result)
	return result
}

func (p *shadowResourceProvisionerReal) Validate(c *ResourceConfig) ([]string, []error) {
	warns, errs := p.ResourceProvisioner.Validate(c)
	p.Shared.Validate.SetValue(&shadowResourceProvisionerValidate{
		Config:     c,
		ResultWarn: warns,
		ResultErr:  errs,
	})

	return warns, errs
}

func (p *shadowResourceProvisionerReal) Apply(
	output UIOutput, s *InstanceState, c *ResourceConfig) error {
	err := p.ResourceProvisioner.Apply(output, s, c)

	// Write the result, grab a lock for writing. This should nver
	// block long since the operations below don't block.
	p.Shared.ApplyLock.Lock()
	defer p.Shared.ApplyLock.Unlock()

	key := s.ID
	raw, ok := p.Shared.Apply.ValueOk(key)
	if !ok {
		// Setup a new value
		raw = &shadow.ComparedValue{
			Func: shadowResourceProvisionerApplyCompare,
		}

		// Set it
		p.Shared.Apply.SetValue(key, raw)
	}

	compareVal, ok := raw.(*shadow.ComparedValue)
	if !ok {
		// Just log and return so that we don't cause the real side
		// any side effects.
		log.Printf("[ERROR] unknown value in 'apply': %#v", raw)
		return err
	}

	// Write the resulting value
	compareVal.SetValue(&shadowResourceProvisionerApply{
		Config:    c,
		ResultErr: err,
	})

	return err
}

func (p *shadowResourceProvisionerReal) Stop() error {
	return p.ResourceProvisioner.Stop()
}

// shadowResourceProvisionerShadow is the shadow resource provisioner. Function
// calls never affect real resources. This is paired with the "real" side
// which must be called properly to enable recording.
type shadowResourceProvisionerShadow struct {
	Shared *shadowResourceProvisionerShared

	Error     error // Error is the list of errors from the shadow
	ErrorLock sync.Mutex
}

type shadowResourceProvisionerShared struct {
	// NOTE: Anytime a value is added here, be sure to add it to
	// the Close() method so that it is closed.

	CloseErr  shadow.Value
	Validate  shadow.ComparedValue
	Apply     shadow.KeyedValue
	ApplyLock sync.Mutex // For writing only
}

func (p *shadowResourceProvisionerShared) Close() error {
	closers := []io.Closer{
		&p.CloseErr,
	}

	for _, c := range closers {
		// This should never happen, but we don't panic because a panic
		// could affect the real behavior of Terraform and a shadow should
		// never be able to do that.
		if err := c.Close(); err != nil {
			return err
		}
	}

	return nil
}

func (p *shadowResourceProvisionerShadow) CloseShadow() error {
	err := p.Shared.Close()
	if err != nil {
		err = fmt.Errorf("close error: %s", err)
	}

	return err
}

func (p *shadowResourceProvisionerShadow) ShadowError() error {
	return p.Error
}

func (p *shadowResourceProvisionerShadow) Close() error {
	v := p.Shared.CloseErr.Value()
	if v == nil {
		return nil
	}

	return v.(error)
}

func (p *shadowResourceProvisionerShadow) Validate(c *ResourceConfig) ([]string, []error) {
	// Get the result of the validate call
	raw := p.Shared.Validate.Value(c)
	if raw == nil {
		return nil, nil
	}

	result, ok := raw.(*shadowResourceProvisionerValidate)
	if !ok {
		p.ErrorLock.Lock()
		defer p.ErrorLock.Unlock()
		p.Error = multierror.Append(p.Error, fmt.Errorf(
			"Unknown 'validate' shadow value: %#v", raw))
		return nil, nil
	}

	// We don't need to compare configurations because we key on the
	// configuration so just return right away.
	return result.ResultWarn, result.ResultErr
}

func (p *shadowResourceProvisionerShadow) Apply(
	output UIOutput, s *InstanceState, c *ResourceConfig) error {
	// Get the value based on the key
	key := s.ID
	raw := p.Shared.Apply.Value(key)
	if raw == nil {
		return nil
	}

	compareVal, ok := raw.(*shadow.ComparedValue)
	if !ok {
		p.ErrorLock.Lock()
		defer p.ErrorLock.Unlock()
		p.Error = multierror.Append(p.Error, fmt.Errorf(
			"Unknown 'apply' shadow value: %#v", raw))
		return nil
	}

	// With the compared value, we compare against our config
	raw = compareVal.Value(c)
	if raw == nil {
		return nil
	}

	result, ok := raw.(*shadowResourceProvisionerApply)
	if !ok {
		p.ErrorLock.Lock()
		defer p.ErrorLock.Unlock()
		p.Error = multierror.Append(p.Error, fmt.Errorf(
			"Unknown 'apply' shadow value: %#v", raw))
		return nil
	}

	return result.ResultErr
}

func (p *shadowResourceProvisionerShadow) Stop() error {
	// For the shadow, we always just return nil since a Stop indicates
	// that we were interrupted and shadows are disabled during interrupts
	// anyways.
	return nil
}

// The structs for the various function calls are put below. These structs
// are used to carry call information across the real/shadow boundaries.

type shadowResourceProvisionerValidate struct {
	Config     *ResourceConfig
	ResultWarn []string
	ResultErr  []error
}

type shadowResourceProvisionerApply struct {
	Config    *ResourceConfig
	ResultErr error
}

func shadowResourceProvisionerValidateCompare(k, v interface{}) bool {
	c, ok := k.(*ResourceConfig)
	if !ok {
		return false
	}

	result, ok := v.(*shadowResourceProvisionerValidate)
	if !ok {
		return false
	}

	return c.Equal(result.Config)
}

func shadowResourceProvisionerApplyCompare(k, v interface{}) bool {
	c, ok := k.(*ResourceConfig)
	if !ok {
		return false
	}

	result, ok := v.(*shadowResourceProvisionerApply)
	if !ok {
		return false
	}

	return c.Equal(result.Config)
}
