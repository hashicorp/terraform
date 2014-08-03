package openstack

import (
	"github.com/haklop/gophercloud-extensions/network"
	"github.com/hashicorp/terraform/flatmap"
	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/terraform"
	"github.com/racker/perigee"
	"github.com/rackspace/gophercloud"
	"log"
	"strconv"
)

func resource_openstack_network_create(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {

	p := meta.(*ResourceProvider)
	client := p.client
	access := p.client.AccessProvider.(*gophercloud.Access)

	// Merge the diff into the state so that we have all the attributes
	// properly.
	rs := s.MergeDiff(d)

	networksApi, err := getNetworkApi(client.AccessProvider)
	if err != nil {
		return nil, err
	}

	newNetwork, err := networksApi.CreateNetwork(network.NewNetwork{
		Name:         rs.Attributes["name"],
		AdminStateUp: true,
		TenantId:     access.Token.Tenant.Id,
	})
	if err != nil {
		return nil, err
	}

	rs.ID = newNetwork.Id
	rs.Attributes["id"] = newNetwork.Id

	log.Printf("[DEBUG] Network create configuration: %#v", newNetwork)

	subnets := []network.Subnet{}
	v, ok := flatmap.Expand(rs.Attributes, "subnet").([]interface{})
	if ok {
		subnets, err = expandSubnets(v)
		if err != nil {
			return rs, err
		}

		for _, subnet := range subnets {
			subnet.NetworkId = rs.ID
			subnet.TenantId = access.Token.Tenant.Id

			// TODO store subnet id for allowed updates
			newSubnet, err := networksApi.CreateSubnet(subnet)
			if err != nil {
				return nil, err
			}

			log.Printf("[DEBUG] Subnet create configuration: %#v", newSubnet)
		}

	}

	return rs, err
}

func resource_openstack_network_destroy(
	s *terraform.ResourceState,
	meta interface{}) error {

	log.Printf("[DEBUG] Destroy network: %s", s.ID)

	p := meta.(*ResourceProvider)
	client := p.client

	networksApi, err := getNetworkApi(client.AccessProvider)
	if err != nil {
		return err
	}

	err = networksApi.DeleteNetwork(s.ID)

	return err
}

func resource_openstack_network_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {

	log.Printf("[DEBUG] Retrieve information about network: %s", s.ID)

	p := meta.(*ResourceProvider)
	client := p.client

	networksApi, err := getNetworkApi(client.AccessProvider)
	if err != nil {
		return nil, err
	}

	_, err = networksApi.GetNetwork(s.ID)
	if err != nil {
		httpError, ok := err.(*perigee.UnexpectedResponseCodeError)
		if !ok {
			return nil, err
		}

		if httpError.Actual == 404 {
			return nil, nil
		}

		return nil, err
	}

	// TODO check subnets

	return s, nil
}

func resource_openstack_network_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {

	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"name":   diff.AttrTypeCreate,
			"subnet": diff.AttrTypeCreate,
		},

		ComputedAttrs: []string{
			"id",
		},
	}

	return b.Diff(s, c)
}

func expandSubnets(configured []interface{}) ([]network.Subnet, error) {
	subnets := make([]network.Subnet, 0, len(configured))

	for _, subnet := range configured {
		raw := subnet.(map[string]interface{})

		newSubnet := network.Subnet{}

		if attr, ok := raw["cidr"].(string); ok {
			newSubnet.Cidr = attr
		}

		if attr, ok := raw["name"].(string); ok {
			newSubnet.Name = attr
		}

		if attr, ok := raw["enable_dhcp"].(bool); ok {
			newSubnet.EnableDhcp = attr
		}

		if attr, ok := raw["ip_version"].(string); ok {
			ipVersion, err := strconv.Atoi(attr)
			if err != nil {
				return nil, err
			}
			newSubnet.IPVersion = ipVersion
		}

		subnets = append(subnets, newSubnet)
	}

	return subnets, nil
}

func getNetworkApi(accessProvider gophercloud.AccessProvider) (network.NetworkProvider, error) {
	api, err := network.NetworksApi(accessProvider, gophercloud.ApiCriteria{
		Name:      "neutron",
		UrlChoice: gophercloud.PublicURL,
	})

	return api, err
}
