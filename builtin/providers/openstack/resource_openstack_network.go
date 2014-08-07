package openstack

import (
	"github.com/haklop/gophercloud-extensions/network"
	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/terraform"
	"github.com/racker/perigee"
	"github.com/rackspace/gophercloud"
	"log"
)

func resource_openstack_network_create(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {

	p := meta.(*ResourceProvider)
	networksApi, err := p.getNetworkApi()
	if err != nil {
		return nil, err
	}

	// Merge the diff into the state so that we have all the attributes
	// properly.
	rs := s.MergeDiff(d)

	access := p.client.AccessProvider.(*gophercloud.Access)

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

	return rs, err
}

func resource_openstack_network_destroy(
	s *terraform.ResourceState,
	meta interface{}) error {

	log.Printf("[DEBUG] Destroy network: %s", s.ID)

	p := meta.(*ResourceProvider)
	networksApi, err := p.getNetworkApi()
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
	networksApi, err := p.getNetworkApi()
	if err != nil {
		return nil, err
	}

	n, err := networksApi.GetNetwork(s.ID)
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

	s.Attributes["name"] = n.Name

	return s, nil
}

func resource_openstack_network_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {

	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"name": diff.AttrTypeUpdate,
		},

		ComputedAttrs: []string{
			"id",
		},
	}

	return b.Diff(s, c)
}

func resource_openstack_network_update(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {

	p := meta.(*ResourceProvider)
	networksApi, err := p.getNetworkApi()
	if err != nil {
		return nil, err
	}

	// Merge the diff into the state so that we have all the attributes
	// properly.
	rs := s.MergeDiff(d)

	if attr, ok := d.Attributes["name"]; ok {
		_, err := networksApi.UpdateNetwork(rs.ID, network.UpdatedNetwork{
			Name: attr.New,
		})

		if err != nil {
			return nil, err
		}

		rs.Attributes["name"] = attr.New
	}

	return rs, nil
}
