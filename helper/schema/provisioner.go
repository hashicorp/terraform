package schema

import (
	"errors"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/terraform"
)

type Provisioner struct {
	Schema       map[string]*Schema
	ValidateFunc ValidateFunc
	ApplyFunc    ApplyFunc
}

type ValidateFunc func(*ResourceData) ([]string, []error)
type ApplyFunc func(terraform.UIOutput, *ResourceData) error

// InternalValidate should be called to validate the structure
// of the provisioner.
//
// This should be called in a unit test for any provisioner to verify
// before release that a provisioner is properly configured for use with
// this library.
func (p *Provisioner) InternalValidate() error {
	if p == nil {
		return errors.New("provisioner is nil")
	}

	var validationErrors error
	sm := schemaMap(p.Schema)
	if err := sm.InternalValidate(sm); err != nil {
		validationErrors = multierror.Append(validationErrors, err)
	}

	return validationErrors
}

func (p *Provisioner) Validate(config *terraform.ResourceConfig) ([]string, []error) {
	if err := p.InternalValidate(); err != nil {
		return nil, []error{fmt.Errorf(
			"Internal validation of the provisioner failed! This is always a bug\n"+
				"with the provisioner itself, and not a user issue. Please report\n"+
				"this bug:\n\n%s", err)}
	}
	w := []string{}
	e := []error{}
	if p.Schema != nil {
		w2, e2 := schemaMap(p.Schema).Validate(config)
		w = append(w, w2...)
		e = append(e, e2...)
	}
	if p.ValidateFunc != nil {
		data := &ResourceData{
			schema: p.Schema,
			config: config,
		}
		w2, e2 := p.ValidateFunc(data)
		w = append(w, w2...)
		e = append(e, e2...)
	}
	return w, e
}

func (p *Provisioner) Apply(ui terraform.UIOutput, state *terraform.InstanceState, config *terraform.ResourceConfig) error {
	if p.ApplyFunc == nil {
		panic("ApplyFunc should be specified in provisioner")
	}
	data := &ResourceData{
		schema: p.Schema,
		config: config,
		state:  state,
	}
	return p.ApplyFunc(ui, data)
}

func (p *Provisioner) TestResourceData(config *terraform.ResourceConfig) *ResourceData {
	return &ResourceData{
		schema: p.Schema,
		config: config,
	}
}
