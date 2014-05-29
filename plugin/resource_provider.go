package plugin

import (
	"os/exec"

	tfrpc "github.com/hashicorp/terraform/rpc"
	"github.com/hashicorp/terraform/terraform"
)

// ResourceProviderFactory returns a Terraform ResourceProviderFactory
// that executes a plugin and connects to it.
func ResourceProviderFactory(cmd *exec.Cmd) terraform.ResourceProviderFactory {
	return func() (terraform.ResourceProvider, error) {
		config := &ClientConfig{
			Cmd:     cmd,
			Managed: true,
		}

		client := NewClient(config)
		rpcClient, err := client.Client()
		if err != nil {
			return nil, err
		}

		rpcName, err := client.Service()
		if err != nil {
			return nil, err
		}

		return &tfrpc.ResourceProvider{
			Client: rpcClient,
			Name:   rpcName,
		}, nil
	}
}
