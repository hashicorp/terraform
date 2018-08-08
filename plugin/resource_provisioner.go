package plugin

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/terraform"
)

// ResourceProvisionerPlugin is the plugin.Plugin implementation.
type ResourceProvisionerPlugin struct {
	F func() terraform.ResourceProvisioner
}

func (p *ResourceProvisionerPlugin) Server(b *plugin.MuxBroker) (interface{}, error) {
	return &ResourceProvisionerServer{Broker: b, Provisioner: p.F()}, nil
}

func (p *ResourceProvisionerPlugin) Client(
	b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &ResourceProvisioner{Broker: b, Client: c}, nil
}

// ResourceProvisioner is an implementation of terraform.ResourceProvisioner
// that communicates over RPC.
type ResourceProvisioner struct {
	Broker *plugin.MuxBroker
	Client *rpc.Client
}

func (p *ResourceProvisioner) GetConfigSchema() (*configschema.Block, error) {
	panic("not implemented")
	return nil, nil
}

func (p *ResourceProvisioner) Validate(c *terraform.ResourceConfig) ([]string, []error) {
	var resp ResourceProvisionerValidateResponse
	args := ResourceProvisionerValidateArgs{
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

func (p *ResourceProvisioner) Apply(
	output terraform.UIOutput,
	s *terraform.InstanceState,
	c *terraform.ResourceConfig) error {
	id := p.Broker.NextId()
	go p.Broker.AcceptAndServe(id, &UIOutputServer{
		UIOutput: output,
	})

	var resp ResourceProvisionerApplyResponse
	args := &ResourceProvisionerApplyArgs{
		OutputId: id,
		State:    s,
		Config:   c,
	}

	err := p.Client.Call("Plugin.Apply", args, &resp)
	if err != nil {
		return err
	}
	if resp.Error != nil {
		err = resp.Error
	}

	return err
}

func (p *ResourceProvisioner) Stop() error {
	var resp ResourceProvisionerStopResponse
	err := p.Client.Call("Plugin.Stop", new(interface{}), &resp)
	if err != nil {
		return err
	}
	if resp.Error != nil {
		err = resp.Error
	}

	return err
}

func (p *ResourceProvisioner) Close() error {
	return p.Client.Close()
}

type ResourceProvisionerValidateArgs struct {
	Config *terraform.ResourceConfig
}

type ResourceProvisionerValidateResponse struct {
	Warnings []string
	Errors   []*plugin.BasicError
}

type ResourceProvisionerApplyArgs struct {
	OutputId uint32
	State    *terraform.InstanceState
	Config   *terraform.ResourceConfig
}

type ResourceProvisionerApplyResponse struct {
	Error *plugin.BasicError
}

type ResourceProvisionerStopResponse struct {
	Error *plugin.BasicError
}

// ResourceProvisionerServer is a net/rpc compatible structure for serving
// a ResourceProvisioner. This should not be used directly.
type ResourceProvisionerServer struct {
	Broker      *plugin.MuxBroker
	Provisioner terraform.ResourceProvisioner
}

func (s *ResourceProvisionerServer) Apply(
	args *ResourceProvisionerApplyArgs,
	result *ResourceProvisionerApplyResponse) error {
	conn, err := s.Broker.Dial(args.OutputId)
	if err != nil {
		*result = ResourceProvisionerApplyResponse{
			Error: plugin.NewBasicError(err),
		}
		return nil
	}
	client := rpc.NewClient(conn)
	defer client.Close()

	output := &UIOutput{Client: client}

	err = s.Provisioner.Apply(output, args.State, args.Config)
	*result = ResourceProvisionerApplyResponse{
		Error: plugin.NewBasicError(err),
	}
	return nil
}

func (s *ResourceProvisionerServer) Validate(
	args *ResourceProvisionerValidateArgs,
	reply *ResourceProvisionerValidateResponse) error {
	warns, errs := s.Provisioner.Validate(args.Config)
	berrs := make([]*plugin.BasicError, len(errs))
	for i, err := range errs {
		berrs[i] = plugin.NewBasicError(err)
	}
	*reply = ResourceProvisionerValidateResponse{
		Warnings: warns,
		Errors:   berrs,
	}
	return nil
}

func (s *ResourceProvisionerServer) Stop(
	_ interface{},
	reply *ResourceProvisionerStopResponse) error {
	err := s.Provisioner.Stop()
	*reply = ResourceProvisionerStopResponse{
		Error: plugin.NewBasicError(err),
	}

	return nil
}
