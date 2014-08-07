package openstack

import (
	"github.com/haklop/gophercloud-extensions/network"
	"github.com/hashicorp/terraform/flatmap"
	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/terraform"
	"github.com/racker/perigee"
	"github.com/rackspace/gophercloud"
	"strconv"
)

func resource_openstack_security_group_create(
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

	rules := []network.SecurityGroupRule{}
	v, ok := flatmap.Expand(rs.Attributes, "rule").([]interface{})
	if ok {
		rules, err = expandRules(v)
		if err != nil {
			return rs, err
		}

		for _, rule := range rules {
			rule.SecurityGroupId = rs.ID
			rule.TenantId = access.Token.Tenant.Id

			// TODO store rules id for allowed updates
			_, err := networksApi.CreateSecurityGroupRule(rule)
			if err != nil {
				return nil, err
			}
		}

	}

	return rs, err
}

func resource_openstack_security_group_destroy(
	s *terraform.ResourceState,
	meta interface{}) error {

	p := meta.(*ResourceProvider)
	networksApi, err := p.getNetworkApi()
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
	networksApi, err := p.getNetworkApi()
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
			"rule":        diff.AttrTypeCreate,
		},

		ComputedAttrs: []string{
			"id",
		},
	}

	return b.Diff(s, c)
}

func expandRules(configured []interface{}) ([]network.SecurityGroupRule, error) {
	rules := make([]network.SecurityGroupRule, 0, len(configured))

	for _, rule := range configured {
		raw := rule.(map[string]interface{})

		newRule := network.SecurityGroupRule{}

		if attr, ok := raw["direction"].(string); ok {
			newRule.Direction = attr
		}

		if attr, ok := raw["remote_ip_prefix"].(string); ok {
			newRule.RemoteIpPrefix = attr
		}

		if attr, ok := raw["port_range_min"].(string); ok {
			minPort, err := strconv.Atoi(attr)
			if err != nil {
				return nil, err
			}
			newRule.PortRangeMin = minPort
		}

		if attr, ok := raw["port_range_max"].(string); ok {
			maxPort, err := strconv.Atoi(attr)
			if err != nil {
				return nil, err
			}
			newRule.PortRangeMax = maxPort
		}

		if attr, ok := raw["protocol"].(string); ok {
			newRule.Protocol = attr
		}

		rules = append(rules, newRule)
	}

	return rules, nil
}
