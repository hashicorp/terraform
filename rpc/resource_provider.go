package rpc

import (
	"net/rpc"

	"github.com/hashicorp/terraform/terraform"
)

// ResourceProvider is an implementation of terraform.ResourceProvider
// that communicates over RPC.
type ResourceProvider struct {
	Broker *muxBroker
	Client *rpc.Client
	Name   string
}

func (p *ResourceProvider) Input(
	input terraform.UIInput,
	c *terraform.ResourceConfig) (*terraform.ResourceConfig, error) {
	id := p.Broker.NextId()
	go acceptAndServe(p.Broker, id, "UIInput", &UIInputServer{
		UIInput: input,
	})

	var resp ResourceProviderInputResponse
	args := ResourceProviderInputArgs{
		InputId: id,
		Config:  c,
	}

	err := p.Client.Call(p.Name+".Input", &args, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		err = resp.Error
		return nil, err
	}

	return resp.Config, nil
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
	info *terraform.InstanceInfo,
	s *terraform.InstanceState,
	d *terraform.InstanceDiff) (*terraform.InstanceState, error) {
	var resp ResourceProviderApplyResponse
	args := &ResourceProviderApplyArgs{
		Info:  info,
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
	info *terraform.InstanceInfo,
	s *terraform.InstanceState,
	c *terraform.ResourceConfig) (*terraform.InstanceDiff, error) {
	var resp ResourceProviderDiffResponse
	args := &ResourceProviderDiffArgs{
		Info:   info,
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
	info *terraform.InstanceInfo,
	s *terraform.InstanceState) (*terraform.InstanceState, error) {
	var resp ResourceProviderRefreshResponse
	args := &ResourceProviderRefreshArgs{
		Info:  info,
		State: s,
	}

	err := p.Client.Call(p.Name+".Refresh", args, &resp)
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

func (p *ResourceProvider) Close() error {
	return p.Client.Close()
}

// ResourceProviderServer is a net/rpc compatible structure for serving
// a ResourceProvider. This should not be used directly.
type ResourceProviderServer struct {
	Broker   *muxBroker
	Provider terraform.ResourceProvider
}

type ResourceProviderConfigureResponse struct {
	Error *BasicError
}

type ResourceProviderInputArgs struct {
	InputId uint32
	Config  *terraform.ResourceConfig
}

type ResourceProviderInputResponse struct {
	Config *terraform.ResourceConfig
	Error  *BasicError
}

type ResourceProviderApplyArgs struct {
	Info  *terraform.InstanceInfo
	State *terraform.InstanceState
	Diff  *terraform.InstanceDiff
}

type ResourceProviderApplyResponse struct {
	State *terraform.InstanceState
	Error *BasicError
}

type ResourceProviderDiffArgs struct {
	Info   *terraform.InstanceInfo
	State  *terraform.InstanceState
	Config *terraform.ResourceConfig
}

type ResourceProviderDiffResponse struct {
	Diff  *terraform.InstanceDiff
	Error *BasicError
}

type ResourceProviderRefreshArgs struct {
	Info  *terraform.InstanceInfo
	State *terraform.InstanceState
}

type ResourceProviderRefreshResponse struct {
	State *terraform.InstanceState
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

func (s *ResourceProviderServer) Input(
	args *ResourceProviderInputArgs,
	reply *ResourceProviderInputResponse) error {
	conn, err := s.Broker.Dial(args.InputId)
	if err != nil {
		*reply = ResourceProviderInputResponse{
			Error: NewBasicError(err),
		}
		return nil
	}
	client := rpc.NewClient(conn)
	defer client.Close()

	input := &UIInput{
		Client: client,
		Name:   "UIInput",
	}

	config, err := s.Provider.Input(input, args.Config)
	*reply = ResourceProviderInputResponse{
		Config: config,
		Error:  NewBasicError(err),
	}

	return nil
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
	state, err := s.Provider.Apply(args.Info, args.State, args.Diff)
	*result = ResourceProviderApplyResponse{
		State: state,
		Error: NewBasicError(err),
	}
	return nil
}

func (s *ResourceProviderServer) Diff(
	args *ResourceProviderDiffArgs,
	result *ResourceProviderDiffResponse) error {
	diff, err := s.Provider.Diff(args.Info, args.State, args.Config)
	*result = ResourceProviderDiffResponse{
		Diff:  diff,
		Error: NewBasicError(err),
	}
	return nil
}

func (s *ResourceProviderServer) Refresh(
	args *ResourceProviderRefreshArgs,
	result *ResourceProviderRefreshResponse) error {
	newState, err := s.Provider.Refresh(args.Info, args.State)
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
