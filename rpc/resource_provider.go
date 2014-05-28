package rpc

import (
	"net/rpc"

	"github.com/hashicorp/terraform/terraform"
)

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
