package rpc

import (
	"net/rpc"

	"github.com/hashicorp/terraform/terraform"
)

// ResourceProvisioner is an implementation of terraform.ResourceProvisioner
// that communicates over RPC.
type ResourceProvisioner struct {
	Client *rpc.Client
	Name   string
}

func (p *ResourceProvisioner) Validate(c *terraform.ResourceConfig) ([]string, []error) {
	var resp ResourceProvisionerValidateResponse
	args := ResourceProvisionerValidateArgs{
		Config: c,
	}

	err := p.Client.Call(p.Name+".Validate", &args, &resp)
	if err != nil {
		return nil, []error{err}
	}

	var errs []error
	if len(resp.Errors) > 0 {
		errs = make([]error, len(resp.Errors))
		for i, err := range resp.Errors {
			errs[i] = err
		}
	}

	return resp.Warnings, errs
}

func (p *ResourceProvisioner) Apply(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig) error {
	var resp ResourceProvisionerApplyResponse
	args := &ResourceProvisionerApplyArgs{
		State:  s,
		Config: c,
	}

	err := p.Client.Call(p.Name+".Apply", args, &resp)
	if err != nil {
		return err
	}
	if resp.Error != nil {
		err = resp.Error
	}

	return err
}

type ResourceProvisionerValidateArgs struct {
	Config *terraform.ResourceConfig
}

type ResourceProvisionerValidateResponse struct {
	Warnings []string
	Errors   []*BasicError
}

type ResourceProvisionerApplyArgs struct {
	State  *terraform.ResourceState
	Config *terraform.ResourceConfig
}

type ResourceProvisionerApplyResponse struct {
	Error *BasicError
}

// ResourceProvisionerServer is a net/rpc compatible structure for serving
// a ResourceProvisioner. This should not be used directly.
type ResourceProvisionerServer struct {
	Provisioner terraform.ResourceProvisioner
}

func (s *ResourceProvisionerServer) Apply(
	args *ResourceProvisionerApplyArgs,
	result *ResourceProvisionerApplyResponse) error {
	err := s.Provisioner.Apply(args.State, args.Config)
	*result = ResourceProvisionerApplyResponse{
		Error: NewBasicError(err),
	}
	return nil
}

func (s *ResourceProvisionerServer) Validate(
	args *ResourceProvisionerValidateArgs,
	reply *ResourceProvisionerValidateResponse) error {
	warns, errs := s.Provisioner.Validate(args.Config)
	berrs := make([]*BasicError, len(errs))
	for i, err := range errs {
		berrs[i] = NewBasicError(err)
	}
	*reply = ResourceProvisionerValidateResponse{
		Warnings: warns,
		Errors:   berrs,
	}
	return nil
}
