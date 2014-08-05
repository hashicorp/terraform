package openstack

import (
	"github.com/haklop/gophercloud-extensions/network"
	"github.com/hashicorp/terraform/flatmap"
	"github.com/hashicorp/terraform/helper/diff"
	//"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	//"github.com/racker/perigee"
	//"github.com/rackspace/gophercloud"
	"log"
)

func resource_openstack_router_create(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {

	p := meta.(*ResourceProvider)
	client := p.client

	// Merge the diff into the state so that we have all the attributes
	// properly.
	rs := s.MergeDiff(d)

	networksApi, err := getNetworkApi(client.AccessProvider)
	if err != nil {
		return nil, err
	}

	if _, ok := rs.Attributes["external_gateway"]; ok {
		if _, ok := rs.Attributes["external_id"]; !ok {

			// TODO find first public network available by requesting GET /networks on Neutron API
			externalId := "first public network available"
			rs.Attributes["external_id"] = externalId
		}
	}

	externalGateway := network.ExternalGateway{
		NetworkId: rs.Attributes["external_id"],
	}

	newRouter, err := networksApi.CreateRouter(network.NewRouter{
		Name:            rs.Attributes["name"],
		AdminStateUp:    true,
		ExternalGateway: externalGateway,
	})
	if err != nil {
		return nil, err
	}

	log.Printf("[DEBUG] New router created: %#v", newRouter)

	rs.Attributes["id"] = newRouter.Id
	rs.ID = newRouter.Id

	// TODO wait for router status

	// Attach subnets
	if raw := flatmap.Expand(rs.Attributes, "subnets"); raw != nil {
		if entries, ok := raw.([]interface{}); ok {
			for _, entry := range entries {
				value, ok := entry.(string)
				if !ok {
					continue
				}

				_, err := networksApi.AddRouterInterface(rs.ID, value)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return rs, nil
}

func resource_openstack_router_update(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {

	return nil, nil
}

func resource_openstack_router_destroy(
	s *terraform.ResourceState,
	meta interface{}) error {

	return nil
}

func resource_openstack_router_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {

	// TODO refresh ports

	return s, nil
}

func resource_openstack_router_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {

	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"name":             diff.AttrTypeCreate,
			"external_gateway": diff.AttrTypeCreate,
			"external_id":      diff.AttrTypeCreate,
			"subnets":          diff.AttrTypeCreate,
		},

		ComputedAttrs: []string{
			"id",
		},
	}

	return b.Diff(s, c)
}
