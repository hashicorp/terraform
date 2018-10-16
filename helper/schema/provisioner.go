package schema

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/terraform"
)

// Provisioner represents a resource provisioner in Terraform and properly
// implements all of the ResourceProvisioner API.
//
// This higher level structure makes it much easier to implement a new or
// custom provisioner for Terraform.
//
// The function callbacks for this structure are all passed a context object.
// This context object has a number of pre-defined values that can be accessed
// via the global functions defined in context.go.
type Provisioner struct {
	// ConnSchema is the schema for the connection settings for this
	// provisioner.
	//
	// The keys of this map are the configuration keys, and the value is
	// the schema describing the value of the configuration.
	//
	// NOTE: The value of connection keys can only be strings for now.
	ConnSchema map[string]*Schema

	// Schema is the schema for the usage of this provisioner.
	//
	// The keys of this map are the configuration keys, and the value is
	// the schema describing the value of the configuration.
	Schema map[string]*Schema

	// ApplyFunc is the function for executing the provisioner. This is required.
	// It is given a context. See the Provisioner struct docs for more
	// information.
	ApplyFunc func(ctx context.Context) error

	// ValidateFunc is a function for extended validation. This is optional
	// and should be used when individual field validation is not enough.
	ValidateFunc func(*terraform.ResourceConfig) ([]string, []error)

	stopCtx       context.Context
	stopCtxCancel context.CancelFunc
	stopOnce      sync.Once
}

// Keys that can be used to access data in the context parameters for
// Provisioners.
var (
	connDataInvalid = contextKey("data invalid")

	// This returns a *ResourceData for the connection information.
	// Guaranteed to never be nil.
	ProvConnDataKey = contextKey("provider conn data")

	// This returns a *ResourceData for the config information.
	// Guaranteed to never be nil.
	ProvConfigDataKey = contextKey("provider config data")

	// This returns a terraform.UIOutput. Guaranteed to never be nil.
	ProvOutputKey = contextKey("provider output")

	// This returns the raw InstanceState passed to Apply. Guaranteed to
	// be set, but may be nil.
	ProvRawStateKey = contextKey("provider raw state")
)

// InternalValidate should be called to validate the structure
// of the provisioner.
//
// This should be called in a unit test to verify before release that this
// structure is properly configured for use.
func (p *Provisioner) InternalValidate() error {
	if p == nil {
		return errors.New("provisioner is nil")
	}

	var validationErrors error
	{
		sm := schemaMap(p.ConnSchema)
		if err := sm.InternalValidate(sm); err != nil {
			validationErrors = multierror.Append(validationErrors, err)
		}
	}

	{
		sm := schemaMap(p.Schema)
		if err := sm.InternalValidate(sm); err != nil {
			validationErrors = multierror.Append(validationErrors, err)
		}
	}

	if p.ApplyFunc == nil {
		validationErrors = multierror.Append(validationErrors, fmt.Errorf(
			"ApplyFunc must not be nil"))
	}

	return validationErrors
}

// StopContext returns a context that checks whether a provisioner is stopped.
func (p *Provisioner) StopContext() context.Context {
	p.stopOnce.Do(p.stopInit)
	return p.stopCtx
}

func (p *Provisioner) stopInit() {
	p.stopCtx, p.stopCtxCancel = context.WithCancel(context.Background())
}

// Stop implementation of terraform.ResourceProvisioner interface.
func (p *Provisioner) Stop() error {
	p.stopOnce.Do(p.stopInit)
	p.stopCtxCancel()
	return nil
}

// GetConfigSchema implementation of terraform.ResourceProvisioner interface.
func (p *Provisioner) GetConfigSchema() (*configschema.Block, error) {
	return schemaMap(p.Schema).CoreConfigSchema(), nil
}

// Apply implementation of terraform.ResourceProvisioner interface.
func (p *Provisioner) Apply(
	o terraform.UIOutput,
	s *terraform.InstanceState,
	c *terraform.ResourceConfig) error {
	var connData, configData *ResourceData

	{
		// We first need to turn the connection information into a
		// terraform.ResourceConfig so that we can use that type to more
		// easily build a ResourceData structure. We do this by simply treating
		// the conn info as configuration input.
		raw := make(map[string]interface{})
		if s != nil {
			for k, v := range s.Ephemeral.ConnInfo {
				raw[k] = v
			}
		}

		c, err := config.NewRawConfig(raw)
		if err != nil {
			return err
		}

		sm := schemaMap(p.ConnSchema)
		diff, err := sm.Diff(nil, terraform.NewResourceConfig(c), nil, nil, true)
		if err != nil {
			return err
		}
		connData, err = sm.Data(nil, diff)
		if err != nil {
			return err
		}
	}

	{
		// Build the configuration data. Doing this requires making a "diff"
		// even though that's never used. We use that just to get the correct types.
		configMap := schemaMap(p.Schema)
		diff, err := configMap.Diff(nil, c, nil, nil, true)
		if err != nil {
			return err
		}
		configData, err = configMap.Data(nil, diff)
		if err != nil {
			return err
		}
	}

	// Build the context and call the function
	ctx := p.StopContext()
	ctx = context.WithValue(ctx, ProvConnDataKey, connData)
	ctx = context.WithValue(ctx, ProvConfigDataKey, configData)
	ctx = context.WithValue(ctx, ProvOutputKey, o)
	ctx = context.WithValue(ctx, ProvRawStateKey, s)
	return p.ApplyFunc(ctx)
}

// Validate implements the terraform.ResourceProvisioner interface.
func (p *Provisioner) Validate(c *terraform.ResourceConfig) (ws []string, es []error) {
	if err := p.InternalValidate(); err != nil {
		return nil, []error{fmt.Errorf(
			"Internal validation of the provisioner failed! This is always a bug\n"+
				"with the provisioner itself, and not a user issue. Please report\n"+
				"this bug:\n\n%s", err)}
	}

	if p.Schema != nil {
		w, e := schemaMap(p.Schema).Validate(c)
		ws = append(ws, w...)
		es = append(es, e...)
	}

	if p.ValidateFunc != nil {
		w, e := p.ValidateFunc(c)
		ws = append(ws, w...)
		es = append(es, e...)
	}

	return ws, es
}
