package openstack

import (
	"github.com/haklop/gophercloud-extensions/network"
	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/terraform"
	"github.com/racker/perigee"
	"github.com/rackspace/gophercloud"
)

func resource_openstack_security_group_create(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {

	p := meta.(*ResourceProvider)
	client := p.client
	access := p.client.AccessProvider.(*gophercloud.Access)

	// Merge the diff into the state so that we have all the attributes
	// properly.
	rs := s.MergeDiff(d)

	networksApi, err := network.NetworksApi(client.AccessProvider, gophercloud.ApiCriteria{
		Name:      "neutron",
		UrlChoice: gophercloud.PublicURL,
	})
	if err != nil {
		return nil, err
	}

	newSecurityGroup, err := networksApi.CreateSecurityGroup(network.NewSecurityGroup{
		Name:        rs.Attributes["name"],
		Description: rs.Attributes["description"],
		TenantId:    access.Token.Tenant.Id,
	})
	if err != nil {
		return nil, err
	}

	rs.ID = newSecurityGroup.Id
	rs.Attributes["id"] = newSecurityGroup.Id

	return rs, err
}

func resource_openstack_security_group_destroy(
	s *terraform.ResourceState,
	meta interface{}) error {

	p := meta.(*ResourceProvider)
	client := p.client

	networksApi, err := network.NetworksApi(client.AccessProvider, gophercloud.ApiCriteria{
		Name:      "neutron",
		UrlChoice: gophercloud.PublicURL,
	})
	if err != nil {
		return err
	}

	err = networksApi.DeleteSecurityGroup(s.ID)

	return err
}

func resource_openstack_security_group_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {

	p := meta.(*ResourceProvider)
	client := p.client

	networksApi, err := network.NetworksApi(client.AccessProvider, gophercloud.ApiCriteria{
		Name:      "neutron",
		UrlChoice: gophercloud.PublicURL,
	})
	if err != nil {
		return nil, err
	}

	_, err = networksApi.GetSecurityGroup(s.ID)
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

	// TODO check rules

	return s, nil
}

func resource_openstack_security_group_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {

	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"name":        diff.AttrTypeCreate,
			"description": diff.AttrTypeCreate,
		},

		ComputedAttrs: []string{
			"id",
		},
	}

	return b.Diff(s, c)
}
