package openstack

import (
	"github.com/haklop/gophercloud-extensions/network"
	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/terraform"
	"github.com/racker/perigee"
	"log"
	"strconv"
)

func resource_openstack_subnet_create(
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

	newSubnet := network.NewSubnet{
		NetworkId: rs.Attributes["network_id"],
		Name:      rs.Attributes["name"],
		Cidr:      rs.Attributes["cidr"],
	}

	enableDhcp, err := strconv.ParseBool(rs.Attributes["enable_dhcp"])
	newSubnet.EnableDhcp = enableDhcp

	ipVersion, err := strconv.Atoi(rs.Attributes["ip_version"])
	newSubnet.IPVersion = ipVersion

	createdSubnet, err := networksApi.CreateSubnet(newSubnet)

	log.Printf("[DEBUG] Create subnet: %s", createdSubnet.Id)

	rs.ID = createdSubnet.Id
	rs.Attributes["id"] = createdSubnet.Id

	return rs, err
}

func resource_openstack_subnet_destroy(
	s *terraform.ResourceState,
	meta interface{}) error {

	p := meta.(*ResourceProvider)
	client := p.client

	networksApi, err := getNetworkApi(client.AccessProvider)
	if err != nil {
		return err
	}

	err = networksApi.DeleteSubnet(s.ID)

	log.Printf("[DEBUG] Destroy subnet: %s", s.ID)

	return err
}

func resource_openstack_subnet_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {

	log.Printf("[DEBUG] Retrieve information about subnet: %s", s.ID)

	p := meta.(*ResourceProvider)
	client := p.client

	networksApi, err := getNetworkApi(client.AccessProvider)
	if err != nil {
		return nil, err
	}

	_, err = networksApi.GetSubnet(s.ID)
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

	return s, nil
}

func resource_openstack_subnet_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {

	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"name":        diff.AttrTypeCreate,
			"cidr":        diff.AttrTypeCreate,
			"ip_version":  diff.AttrTypeCreate,
			"enable_dhcp": diff.AttrTypeCreate,
			"network_id":  diff.AttrTypeCreate,
		},

		ComputedAttrs: []string{
			"id",
		},
	}

	return b.Diff(s, c)
}
