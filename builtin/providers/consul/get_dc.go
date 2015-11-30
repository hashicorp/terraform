package consul

import (
	"fmt"

	consulapi "github.com/hashicorp/consul/api"
)

// getDC is used to get the datacenter of the local agent
func getDC(client *consulapi.Client) (string, error) {
	info, err := client.Agent().Self()
	if err != nil {
		return "", fmt.Errorf("Failed to get datacenter from Consul agent: %v", err)
	}
	dc := info["Config"]["Datacenter"].(string)
	return dc, nil
}
