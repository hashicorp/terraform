package openstack

import (
	"github.com/haklop/gophercloud-extensions/network"
	"github.com/hashicorp/terraform/flatmap"
	"github.com/hashicorp/terraform/helper/diff"
	//"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/racker/perigee"
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

	newRouter := network.NewRouter{
		Name:         rs.Attributes["name"],
		AdminStateUp: true,
	}

	if _, ok := rs.Attributes["external_gateway"]; ok {
		if _, ok := rs.Attributes["external_id"]; !ok {

			// TODO find first public network available by requesting GET /networks on Neutron API
			externalId := "first public network available"
			rs.Attributes["external_id"] = externalId
		}

		externalGateway := network.ExternalGateway{
			NetworkId: rs.Attributes["external_id"],
		}

		newRouter.ExternalGateway = externalGateway
	} else {
		rs.Attributes["external_id"] = ""
	}

	createdRouter, err := networksApi.CreateRouter(newRouter)
	if err != nil {
		return nil, err
	}

	log.Printf("[DEBUG] New router created: %#v", createdRouter)

	rs.Attributes["id"] = createdRouter.Id
	rs.ID = createdRouter.Id

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

	p := meta.(*ResourceProvider)
	client := p.client

	// Merge the diff into the state so that we have all the attributes
	// properly.
	rs := s.MergeDiff(d)

	networksApi, err := getNetworkApi(client.AccessProvider)
	if err != nil {
		return nil, err
	}

	log.Printf("[DEBUG] Updating router: %#v", s.ID)

	if attr, ok := d.Attributes["name"]; ok {
		_, err := networksApi.UpdateRouter(network.Router{
			Id:   rs.ID,
			Name: attr.New,
		})

		if err != nil {
			return nil, err
		}

		rs.Attributes["name"] = attr.New
	}

	return rs, nil
}

func resource_openstack_router_destroy(
	s *terraform.ResourceState,
	meta interface{}) error {

	p := meta.(*ResourceProvider)
	client := p.client

	networksApi, err := getNetworkApi(client.AccessProvider)
	if err != nil {
		return err
	}

	ports, err := networksApi.GetPorts()
	if err != nil {
		return err
	}

	for _, port := range ports {
		if port.DeviceId == s.ID {
			err = networksApi.RemoveRouterInterface(s.ID, port.PortId)
		}
	}

	err = networksApi.DeleteRouter(s.ID)
	if err != nil {
		httpError, ok := err.(*perigee.UnexpectedResponseCodeError)
		if !ok {
			return err
		}

		if httpError.Actual == 404 {
			return nil
		}
	}
	return err
}

func resource_openstack_router_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {

	p := meta.(*ResourceProvider)
	client := p.client

	networksApi, err := getNetworkApi(client.AccessProvider)
	if err != nil {
		return nil, err
	}

	router, err := networksApi.GetRouter(s.ID)
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

	s.Attributes["name"] = router.Name
	s.Attributes["external_id"] = router.ExternalGateway.NetworkId

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
			"external_id",
		},
	}

	return b.Diff(s, c)
}
