package openstack

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/servergroups"
)

const (
	softAntiAffinityPolicy = "soft-anti-affinity"
	softAffinityPolicy     = "soft-affinity"
)

// ServerGroupCreateOpts is a custom ServerGroup struct to include the
// ValueSpecs field.
type ComputeServerGroupV2CreateOpts struct {
	servergroups.CreateOpts
	ValueSpecs map[string]string `json:"value_specs,omitempty"`
}

// ToServerGroupCreateMap casts a CreateOpts struct to a map.
// It overrides routers.ToServerGroupCreateMap to add the ValueSpecs field.
func (opts ComputeServerGroupV2CreateOpts) ToServerGroupCreateMap() (map[string]interface{}, error) {
	return BuildRequest(opts, "server_group")
}

func expandComputeServerGroupV2Policies(client *gophercloud.ServiceClient, raw []interface{}) []string {
	policies := make([]string, len(raw))
	for i, v := range raw {
		policy := v.(string)
		policies[i] = policy

		// Set microversion for new policies.
		if policy == softAntiAffinityPolicy || policy == softAffinityPolicy {
			client.Microversion = "2.15"
		}
	}

	return policies
}
