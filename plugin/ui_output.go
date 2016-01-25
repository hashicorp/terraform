package rpc

import (
	"net/rpc"

	"github.com/hashicorp/terraform/terraform"
)

// UIOutput is an implementatin of terraform.UIOutput that communicates
// over RPC.
type UIOutput struct {
	Client *rpc.Client
	Name   string
}

func (o *UIOutput) Output(v string) {
	o.Client.Call(o.Name+".Output", v, new(interface{}))
}

// UIOutputServer is the RPC server for serving UIOutput.
type UIOutputServer struct {
	UIOutput terraform.UIOutput
}

func (s *UIOutputServer) Output(
	v string,
	reply *interface{}) error {
	s.UIOutput.Output(v)
	return nil
}
