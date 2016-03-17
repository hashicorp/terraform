package plugin

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform/terraform"
)

// UIInput is an implementatin of terraform.UIInput that communicates
// over RPC.
type UIInput struct {
	Client *rpc.Client
}

func (i *UIInput) Input(opts *terraform.InputOpts) (string, error) {
	var resp UIInputInputResponse
	err := i.Client.Call("Plugin.Input", opts, &resp)
	if err != nil {
		return "", err
	}
	if resp.Error != nil {
		err = resp.Error
		return "", err
	}

	return resp.Value, nil
}

type UIInputInputResponse struct {
	Value string
	Error *plugin.BasicError
}

// UIInputServer is a net/rpc compatible structure for serving
// a UIInputServer. This should not be used directly.
type UIInputServer struct {
	UIInput terraform.UIInput
}

func (s *UIInputServer) Input(
	opts *terraform.InputOpts,
	reply *UIInputInputResponse) error {
	value, err := s.UIInput.Input(opts)
	*reply = UIInputInputResponse{
		Value: value,
		Error: plugin.NewBasicError(err),
	}

	return nil
}
