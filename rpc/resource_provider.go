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

func (p *ResourceProvider) Configure(c map[string]interface{}) ([]string, error) {
	var resp ResourceProviderConfigureResponse
	err := p.Client.Call(p.Name+".Configure", c, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		err = resp.Error
	}

	return resp.Warnings, err
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
	Warnings []string
	Error    *BasicError
}

func (s *ResourceProviderServer) Configure(
	config map[string]interface{},
	reply *ResourceProviderConfigureResponse) error {
	warnings, err := s.Provider.Configure(config)
	*reply = ResourceProviderConfigureResponse{
		Warnings: warnings,
		Error:    NewBasicError(err),
	}
	return nil
}

func (s *ResourceProviderServer) Resources(
	nothing interface{},
	result *[]terraform.ResourceType) error {
	*result = s.Provider.Resources()
	return nil
}
