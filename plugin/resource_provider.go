package plugin

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform/terraform"
)

// ResourceProviderPlugin is the plugin.Plugin implementation.
type ResourceProviderPlugin struct {
	F func() terraform.ResourceProvider
}

func (p *ResourceProviderPlugin) Server(b *plugin.MuxBroker) (interface{}, error) {
	return &ResourceProviderServer{Broker: b, Provider: p.F()}, nil
}

func (p *ResourceProviderPlugin) Client(
	b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &ResourceProvider{Broker: b, Client: c}, nil
}

// ResourceProvider is an implementation of terraform.ResourceProvider
// that communicates over RPC.
type ResourceProvider struct {
	Broker *plugin.MuxBroker
	Client *rpc.Client
}

func (p *ResourceProvider) Stop() error {
	var resp ResourceProviderStopResponse
	err := p.Client.Call("Plugin.Stop", new(interface{}), &resp)
	if err != nil {
		return err
	}
	if resp.Error != nil {
		err = resp.Error
	}

	return err
}

func (p *ResourceProvider) GetSchema(req *terraform.ProviderSchemaRequest) (*terraform.ProviderSchema, error) {
	var result ResourceProviderGetSchemaResponse
	args := &ResourceProviderGetSchemaArgs{
		Req: req,
	}

	err := p.Client.Call("Plugin.GetSchema", args, &result)
	if err != nil {
		return nil, err
	}

	if result.Error != nil {
		err = result.Error
	}

	return result.Schema, err
}

func (p *ResourceProvider) Input(
	input terraform.UIInput,
	c *terraform.ResourceConfig) (*terraform.ResourceConfig, error) {
	id := p.Broker.NextId()
	go p.Broker.AcceptAndServe(id, &UIInputServer{
		UIInput: input,
	})

	var resp ResourceProviderInputResponse
	args := ResourceProviderInputArgs{
		InputId: id,
		Config:  c,
	}

	err := p.Client.Call("Plugin.Input", &args, &resp)
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

	err := p.Client.Call("Plugin.Validate", &args, &resp)
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

	err := p.Client.Call("Plugin.ValidateResource", &args, &resp)
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
	err := p.Client.Call("Plugin.Configure", c, &resp)
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

	err := p.Client.Call("Plugin.Apply", args, &resp)
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
	err := p.Client.Call("Plugin.Diff", args, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		err = resp.Error
	}

	return resp.Diff, err
}

func (p *ResourceProvider) ValidateDataSource(
	t string, c *terraform.ResourceConfig) ([]string, []error) {
	var resp ResourceProviderValidateResourceResponse
	args := ResourceProviderValidateResourceArgs{
		Config: c,
		Type:   t,
	}

	err := p.Client.Call("Plugin.ValidateDataSource", &args, &resp)
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

func (p *ResourceProvider) Refresh(
	info *terraform.InstanceInfo,
	s *terraform.InstanceState) (*terraform.InstanceState, error) {
	var resp ResourceProviderRefreshResponse
	args := &ResourceProviderRefreshArgs{
		Info:  info,
		State: s,
	}

	err := p.Client.Call("Plugin.Refresh", args, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		err = resp.Error
	}

	return resp.State, err
}

func (p *ResourceProvider) ImportState(
	info *terraform.InstanceInfo,
	id string) ([]*terraform.InstanceState, error) {
	var resp ResourceProviderImportStateResponse
	args := &ResourceProviderImportStateArgs{
		Info: info,
		Id:   id,
	}

	err := p.Client.Call("Plugin.ImportState", args, &resp)
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

	err := p.Client.Call("Plugin.Resources", new(interface{}), &result)
	if err != nil {
		// TODO: panic, log, what?
		return nil
	}

	return result
}

func (p *ResourceProvider) ReadDataDiff(
	info *terraform.InstanceInfo,
	c *terraform.ResourceConfig) (*terraform.InstanceDiff, error) {
	var resp ResourceProviderReadDataDiffResponse
	args := &ResourceProviderReadDataDiffArgs{
		Info:   info,
		Config: c,
	}

	err := p.Client.Call("Plugin.ReadDataDiff", args, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		err = resp.Error
	}

	return resp.Diff, err
}

func (p *ResourceProvider) ReadDataApply(
	info *terraform.InstanceInfo,
	d *terraform.InstanceDiff) (*terraform.InstanceState, error) {
	var resp ResourceProviderReadDataApplyResponse
	args := &ResourceProviderReadDataApplyArgs{
		Info: info,
		Diff: d,
	}

	err := p.Client.Call("Plugin.ReadDataApply", args, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		err = resp.Error
	}

	return resp.State, err
}

func (p *ResourceProvider) DataSources() []terraform.DataSource {
	var result []terraform.DataSource

	err := p.Client.Call("Plugin.DataSources", new(interface{}), &result)
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
	Broker   *plugin.MuxBroker
	Provider terraform.ResourceProvider
}

type ResourceProviderStopResponse struct {
	Error *plugin.BasicError
}

type ResourceProviderGetSchemaArgs struct {
	Req *terraform.ProviderSchemaRequest
}

type ResourceProviderGetSchemaResponse struct {
	Schema *terraform.ProviderSchema
	Error  *plugin.BasicError
}

type ResourceProviderConfigureResponse struct {
	Error *plugin.BasicError
}

type ResourceProviderInputArgs struct {
	InputId uint32
	Config  *terraform.ResourceConfig
}

type ResourceProviderInputResponse struct {
	Config *terraform.ResourceConfig
	Error  *plugin.BasicError
}

type ResourceProviderApplyArgs struct {
	Info  *terraform.InstanceInfo
	State *terraform.InstanceState
	Diff  *terraform.InstanceDiff
}

type ResourceProviderApplyResponse struct {
	State *terraform.InstanceState
	Error *plugin.BasicError
}

type ResourceProviderDiffArgs struct {
	Info   *terraform.InstanceInfo
	State  *terraform.InstanceState
	Config *terraform.ResourceConfig
}

type ResourceProviderDiffResponse struct {
	Diff  *terraform.InstanceDiff
	Error *plugin.BasicError
}

type ResourceProviderRefreshArgs struct {
	Info  *terraform.InstanceInfo
	State *terraform.InstanceState
}

type ResourceProviderRefreshResponse struct {
	State *terraform.InstanceState
	Error *plugin.BasicError
}

type ResourceProviderImportStateArgs struct {
	Info *terraform.InstanceInfo
	Id   string
}

type ResourceProviderImportStateResponse struct {
	State []*terraform.InstanceState
	Error *plugin.BasicError
}

type ResourceProviderReadDataApplyArgs struct {
	Info *terraform.InstanceInfo
	Diff *terraform.InstanceDiff
}

type ResourceProviderReadDataApplyResponse struct {
	State *terraform.InstanceState
	Error *plugin.BasicError
}

type ResourceProviderReadDataDiffArgs struct {
	Info   *terraform.InstanceInfo
	Config *terraform.ResourceConfig
}

type ResourceProviderReadDataDiffResponse struct {
	Diff  *terraform.InstanceDiff
	Error *plugin.BasicError
}

type ResourceProviderValidateArgs struct {
	Config *terraform.ResourceConfig
}

type ResourceProviderValidateResponse struct {
	Warnings []string
	Errors   []*plugin.BasicError
}

type ResourceProviderValidateResourceArgs struct {
	Config *terraform.ResourceConfig
	Type   string
}

type ResourceProviderValidateResourceResponse struct {
	Warnings []string
	Errors   []*plugin.BasicError
}

func (s *ResourceProviderServer) Stop(
	_ interface{},
	reply *ResourceProviderStopResponse) error {
	err := s.Provider.Stop()
	*reply = ResourceProviderStopResponse{
		Error: plugin.NewBasicError(err),
	}

	return nil
}

func (s *ResourceProviderServer) GetSchema(
	args *ResourceProviderGetSchemaArgs,
	result *ResourceProviderGetSchemaResponse,
) error {
	schema, err := s.Provider.GetSchema(args.Req)
	result.Schema = schema
	if err != nil {
		result.Error = plugin.NewBasicError(err)
	}
	return nil
}

func (s *ResourceProviderServer) Input(
	args *ResourceProviderInputArgs,
	reply *ResourceProviderInputResponse) error {
	conn, err := s.Broker.Dial(args.InputId)
	if err != nil {
		*reply = ResourceProviderInputResponse{
			Error: plugin.NewBasicError(err),
		}
		return nil
	}
	client := rpc.NewClient(conn)
	defer client.Close()

	input := &UIInput{Client: client}

	config, err := s.Provider.Input(input, args.Config)
	*reply = ResourceProviderInputResponse{
		Config: config,
		Error:  plugin.NewBasicError(err),
	}

	return nil
}

func (s *ResourceProviderServer) Validate(
	args *ResourceProviderValidateArgs,
	reply *ResourceProviderValidateResponse) error {
	warns, errs := s.Provider.Validate(args.Config)
	berrs := make([]*plugin.BasicError, len(errs))
	for i, err := range errs {
		berrs[i] = plugin.NewBasicError(err)
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
	berrs := make([]*plugin.BasicError, len(errs))
	for i, err := range errs {
		berrs[i] = plugin.NewBasicError(err)
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
		Error: plugin.NewBasicError(err),
	}
	return nil
}

func (s *ResourceProviderServer) Apply(
	args *ResourceProviderApplyArgs,
	result *ResourceProviderApplyResponse) error {
	state, err := s.Provider.Apply(args.Info, args.State, args.Diff)
	*result = ResourceProviderApplyResponse{
		State: state,
		Error: plugin.NewBasicError(err),
	}
	return nil
}

func (s *ResourceProviderServer) Diff(
	args *ResourceProviderDiffArgs,
	result *ResourceProviderDiffResponse) error {
	diff, err := s.Provider.Diff(args.Info, args.State, args.Config)
	*result = ResourceProviderDiffResponse{
		Diff:  diff,
		Error: plugin.NewBasicError(err),
	}
	return nil
}

func (s *ResourceProviderServer) Refresh(
	args *ResourceProviderRefreshArgs,
	result *ResourceProviderRefreshResponse) error {
	newState, err := s.Provider.Refresh(args.Info, args.State)
	*result = ResourceProviderRefreshResponse{
		State: newState,
		Error: plugin.NewBasicError(err),
	}
	return nil
}

func (s *ResourceProviderServer) ImportState(
	args *ResourceProviderImportStateArgs,
	result *ResourceProviderImportStateResponse) error {
	states, err := s.Provider.ImportState(args.Info, args.Id)
	*result = ResourceProviderImportStateResponse{
		State: states,
		Error: plugin.NewBasicError(err),
	}
	return nil
}

func (s *ResourceProviderServer) Resources(
	nothing interface{},
	result *[]terraform.ResourceType) error {
	*result = s.Provider.Resources()
	return nil
}

func (s *ResourceProviderServer) ValidateDataSource(
	args *ResourceProviderValidateResourceArgs,
	reply *ResourceProviderValidateResourceResponse) error {
	warns, errs := s.Provider.ValidateDataSource(args.Type, args.Config)
	berrs := make([]*plugin.BasicError, len(errs))
	for i, err := range errs {
		berrs[i] = plugin.NewBasicError(err)
	}
	*reply = ResourceProviderValidateResourceResponse{
		Warnings: warns,
		Errors:   berrs,
	}
	return nil
}

func (s *ResourceProviderServer) ReadDataDiff(
	args *ResourceProviderReadDataDiffArgs,
	result *ResourceProviderReadDataDiffResponse) error {
	diff, err := s.Provider.ReadDataDiff(args.Info, args.Config)
	*result = ResourceProviderReadDataDiffResponse{
		Diff:  diff,
		Error: plugin.NewBasicError(err),
	}
	return nil
}

func (s *ResourceProviderServer) ReadDataApply(
	args *ResourceProviderReadDataApplyArgs,
	result *ResourceProviderReadDataApplyResponse) error {
	newState, err := s.Provider.ReadDataApply(args.Info, args.Diff)
	*result = ResourceProviderReadDataApplyResponse{
		State: newState,
		Error: plugin.NewBasicError(err),
	}
	return nil
}

func (s *ResourceProviderServer) DataSources(
	nothing interface{},
	result *[]terraform.DataSource) error {
	*result = s.Provider.DataSources()
	return nil
}
