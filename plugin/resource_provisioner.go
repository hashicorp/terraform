package plugin

import (
	"os/exec"

	tfrpc "github.com/hashicorp/terraform/rpc"
	"github.com/hashicorp/terraform/terraform"
)

// ResourceProvisionerFactory returns a Terraform ResourceProvisionerFactory
// that executes a plugin and connects to it.
func ResourceProvisionerFactory(cmd *exec.Cmd) terraform.ResourceProvisionerFactory {
	return func() (terraform.ResourceProvisioner, error) {
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

		return &tfrpc.ResourceProvisioner{
			Client: rpcClient,
			Name:   rpcName,
		}, nil
	}
}
