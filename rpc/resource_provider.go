package rpc

import (
	"net/rpc"

	"github.com/hashicorp/terraform/terraform"
)

// ResourceProvider is an implementation of terraform.ResourceProvider
// that communicates over RPC.
type ResourceProvider struct {
	Client *rpc.Client
	Name   string
}

func (p *ResourceProvider) Validate(c *terraform.ResourceConfig) ([]string, []error) {
	var resp ResourceProviderValidateResponse
	args := ResourceProviderValidateArgs{
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

func (p *ResourceProvider) ValidateResource(
	t string, c *terraform.ResourceConfig) ([]string, []error) {
	var resp ResourceProviderValidateResourceResponse
	args := ResourceProviderValidateResourceArgs{
		Config: c,
		Type:   t,
	}

	err := p.Client.Call(p.Name+".ValidateResource", &args, &resp)
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

func (p *ResourceProvider) Configure(c *terraform.ResourceConfig) error {
	var resp ResourceProviderConfigureResponse
	err := p.Client.Call(p.Name+".Configure", c, &resp)
	if err != nil {
		return err
	}
	if resp.Error != nil {
		err = resp.Error
	}

	return err
}

func (p *ResourceProvider) Apply(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff) (*terraform.ResourceState, error) {
	var resp ResourceProviderApplyResponse
	args := &ResourceProviderApplyArgs{
		State: s,
		Diff:  d,
	}

	err := p.Client.Call(p.Name+".Apply", args, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		err = resp.Error
	}

	return resp.State, err
}

func (p *ResourceProvider) Diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig) (*terraform.ResourceDiff, error) {
	var resp ResourceProviderDiffResponse
	args := &ResourceProviderDiffArgs{
		State:  s,
		Config: c,
	}
	err := p.Client.Call(p.Name+".Diff", args, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		err = resp.Error
	}

	return resp.Diff, err
}

func (p *ResourceProvider) Refresh(
	s *terraform.ResourceState) (*terraform.ResourceState, error) {
	var resp ResourceProviderRefreshResponse
	err := p.Client.Call(p.Name+".Refresh", s, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		err = resp.Error
	}

	return resp.State, err
}

func (p *ResourceProvider) Resources() []terraform.ResourceType {
	var result []terraform.ResourceType

	err := p.Client.Call(p.Name+".Resources", new(interface{}), &result)
	if err != nil {
		// TODO: panic, log, what?
		return nil
	}

	return result
}

// ResourceProviderServer is a net/rpc compatible structure for serving
// a ResourceProvider. This should not be used directly.
type ResourceProviderServer struct {
	Provider terraform.ResourceProvider
}

type ResourceProviderConfigureResponse struct {
	Error *BasicError
}

type ResourceProviderApplyArgs struct {
	State *terraform.ResourceState
	Diff  *terraform.ResourceDiff
}

type ResourceProviderApplyResponse struct {
	State *terraform.ResourceState
	Error *BasicError
}

type ResourceProviderDiffArgs struct {
	State  *terraform.ResourceState
	Config *terraform.ResourceConfig
}

type ResourceProviderDiffResponse struct {
	Diff  *terraform.ResourceDiff
	Error *BasicError
}

type ResourceProviderRefreshResponse struct {
	State *terraform.ResourceState
	Error *BasicError
}

type ResourceProviderValidateArgs struct {
	Config *terraform.ResourceConfig
}

type ResourceProviderValidateResponse struct {
	Warnings []string
	Errors   []*BasicError
}

type ResourceProviderValidateResourceArgs struct {
	Config *terraform.ResourceConfig
	Type   string
}

type ResourceProviderValidateResourceResponse struct {
	Warnings []string
	Errors   []*BasicError
}

func (s *ResourceProviderServer) Validate(
	args *ResourceProviderValidateArgs,
	reply *ResourceProviderValidateResponse) error {
	warns, errs := s.Provider.Validate(args.Config)
	berrs := make([]*BasicError, len(errs))
	for i, err := range errs {
		berrs[i] = NewBasicError(err)
	}
	*reply = ResourceProviderValidateResponse{
		Warnings: warns,
		Errors:   berrs,
	}
	return nil
}

func (s *ResourceProviderServer) ValidateResource(
	args *ResourceProviderValidateResourceArgs,
	reply *ResourceProviderValidateResourceResponse) error {
	warns, errs := s.Provider.ValidateResource(args.Type, args.Config)
	berrs := make([]*BasicError, len(errs))
	for i, err := range errs {
		berrs[i] = NewBasicError(err)
	}
	*reply = ResourceProviderValidateResourceResponse{
		Warnings: warns,
		Errors:   berrs,
	}
	return nil
}

func (s *ResourceProviderServer) Configure(
	config *terraform.ResourceConfig,
	reply *ResourceProviderConfigureResponse) error {
	err := s.Provider.Configure(config)
	*reply = ResourceProviderConfigureResponse{
		Error: NewBasicError(err),
	}
	return nil
}

func (s *ResourceProviderServer) Apply(
	args *ResourceProviderApplyArgs,
	result *ResourceProviderApplyResponse) error {
	state, err := s.Provider.Apply(args.State, args.Diff)
	*result = ResourceProviderApplyResponse{
		State: state,
		Error: NewBasicError(err),
	}
	return nil
}

func (s *ResourceProviderServer) Diff(
	args *ResourceProviderDiffArgs,
	result *ResourceProviderDiffResponse) error {
	diff, err := s.Provider.Diff(args.State, args.Config)
	*result = ResourceProviderDiffResponse{
		Diff:  diff,
		Error: NewBasicError(err),
	}
	return nil
}

func (s *ResourceProviderServer) Refresh(
	state *terraform.ResourceState,
	result *ResourceProviderRefreshResponse) error {
	newState, err := s.Provider.Refresh(state)
	*result = ResourceProviderRefreshResponse{
		State: newState,
		Error: NewBasicError(err),
	}
	return nil
}

func (s *ResourceProviderServer) Resources(
	nothing interface{},
	result *[]terraform.ResourceType) error {
	*result = s.Provider.Resources()
	return nil
}
