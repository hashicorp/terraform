package openstack

import (
	"github.com/haklop/gophercloud-extensions/network"
	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/terraform"
	"github.com/racker/perigee"
	"log"
)

func resource_openstack_lbaas_create(
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

	pool, err := networksApi.CreatePool(network.NewPool{
		Name:        rs.Attributes["name"],
		SubnetId:    rs.Attributes["subnet_id"],
		LoadMethod:  rs.Attributes["lb_method"],
		Protocol:    rs.Attributes["protocol"],
		Description: rs.Attributes["description"],
	})
	if err != nil {
		return nil, err
	}

	rs.ID = pool.Id

	return rs, err
}

func resource_openstack_lbaas_destroy(
	s *terraform.ResourceState,
	meta interface{}) error {

	p := meta.(*ResourceProvider)
	networksApi, err := p.getNetworkApi()
	if err != nil {
		return err
	}

	err = networksApi.DeletePool(s.ID)

	return err
}

func resource_openstack_lbaas_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {

	log.Printf("[DEBUG] Retrieve information about pool: %s", s.ID)

	p := meta.(*ResourceProvider)
	networksApi, err := p.getNetworkApi()
	if err != nil {
		return nil, err
	}

	pool, err := networksApi.GetPool(s.ID)
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

	s.Attributes["name"] = pool.Name
	s.Attributes["description"] = pool.Description
	s.Attributes["lb_method"] = pool.LoadMethod

	return s, nil
}

func resource_openstack_lbaas_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {

	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"name":        diff.AttrTypeUpdate,
			"subnet_id":   diff.AttrTypeCreate,
			"protocol":    diff.AttrTypeCreate,
			"lb_method":   diff.AttrTypeUpdate,
			"description": diff.AttrTypeUpdate,
		},

		ComputedAttrs: []string{
			"id",
		},
	}

	return b.Diff(s, c)
}

func resource_openstack_lbaas_update(
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

	updatedPool := network.Pool{
		Id: rs.ID,
	}

	if attr, ok := d.Attributes["name"]; ok {
		updatedPool.Name = attr.New
		rs.Attributes["name"] = attr.New
	}

	if attr, ok := d.Attributes["lb_method"]; ok {
		updatedPool.LoadMethod = attr.New
		rs.Attributes["lb_method"] = attr.New
	}

	if attr, ok := d.Attributes["description"]; ok {
		updatedPool.Description = attr.New
		rs.Attributes["description"] = attr.New
	}

	_, err = networksApi.UpdatePool(updatedPool)

	return rs, err
}
