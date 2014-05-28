package rpc

import (
	"net/rpc"

	"github.com/hashicorp/terraform/terraform"
)

type ResourceProvider struct {
	Client *rpc.Client
}

func (p *ResourceProvider) Configure(c map[string]interface{}) ([]string, error) {
	var resp ResourceProviderConfigureResponse
	err := p.Client.Call("ResourceProvider.Configure", c, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Warnings, resp.Error
}

type ResourceProviderServer struct {
	Provider terraform.ResourceProvider
}

type ResourceProviderConfigureResponse struct {
	Warnings []string
	Error    error
}

func (s *ResourceProviderServer) Configure(
	config map[string]interface{},
	reply *ResourceProviderConfigureResponse) error {
	warnings, err := s.Provider.Configure(config)
	*reply = ResourceProviderConfigureResponse{
		Warnings: warnings,
		Error:    err,
	}
	return nil
}
