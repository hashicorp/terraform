package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/ec2"
)

func resource_aws_route_table_association_create(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn
	rs := s.MergeDiff(d)

	log.Printf(
		"[INFO] Creating route table association: %s => %s",
		rs.Attributes["subnet_id"],
		rs.Attributes["route_table_id"])
	resp, err := ec2conn.AssociateRouteTable(
		rs.Attributes["route_table_id"],
		rs.Attributes["subnet_id"])
	if err != nil {
		return nil, err
	}

	// Set the ID and return
	rs.ID = resp.AssociationId
	log.Printf("[INFO] Association ID: %s", rs.ID)

	rs.Dependencies = []terraform.ResourceDependency{
		terraform.ResourceDependency{ID: rs.Attributes["route_table_id"]},
	}

	return rs, nil
}

func resource_aws_route_table_association_update(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	rs := s.MergeDiff(d)
	log.Printf(
		"[INFO] Replacing route table association: %s => %s",
		rs.Attributes["subnet_id"],
		rs.Attributes["route_table_id"])
	resp, err := ec2conn.ReassociateRouteTable(
		rs.ID,
		rs.Attributes["route_table_id"])
	if err != nil {
		ec2err, ok := err.(*ec2.Error)
		if ok && ec2err.Code == "InvalidAssociationID.NotFound" {
			// Not found, so just create a new one
			return resource_aws_route_table_association_create(s, d, meta)
		}

		return s, err
	}

	// Update the ID
	rs.ID = resp.AssociationId
	log.Printf("[INFO] Association ID: %s", rs.ID)

	rs.Dependencies = []terraform.ResourceDependency{
		terraform.ResourceDependency{ID: rs.Attributes["route_table_id"]},
	}

	return rs, nil
}

func resource_aws_route_table_association_destroy(
	s *terraform.ResourceState,
	meta interface{}) error {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	log.Printf("[INFO] Deleting route table association: %s", s.ID)
	if _, err := ec2conn.DisassociateRouteTable(s.ID); err != nil {
		ec2err, ok := err.(*ec2.Error)
		if ok && ec2err.Code == "InvalidAssociationID.NotFound" {
			return nil
		}

		return fmt.Errorf("Error deleting route table association: %s", err)
	}

	return nil
}

func resource_aws_route_table_association_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	// Get the routing table that this association belongs to
	rtRaw, _, err := RouteTableStateRefreshFunc(
		ec2conn, s.Attributes["route_table_id"])()
	if err != nil {
		return s, err
	}
	if rtRaw == nil {
		return nil, nil
	}
	rt := rtRaw.(*ec2.RouteTable)

	// Inspect that the association exists
	found := false
	for _, a := range rt.Associations {
		if a.AssociationId == s.ID {
			found = true
			s.Attributes["subnet_id"] = a.SubnetId
			break
		}
	}
	if !found {
		return nil, nil
	}

	return s, nil
}

func resource_aws_route_table_association_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {
	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"subnet_id":      diff.AttrTypeCreate,
			"route_table_id": diff.AttrTypeUpdate,
		},
	}

	return b.Diff(s, c)
}
