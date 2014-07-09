package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/ec2"
)

func resource_aws_route_table_create(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	// Merge the diff so that we have all the proper attributes
	s = s.MergeDiff(d)

	// Create the Subnet
	createOpts := &ec2.CreateRouteTable{
		VpcId: s.Attributes["vpc_id"],
	}
	log.Printf("[DEBUG] RouteTable create config: %#v", createOpts)
	resp, err := ec2conn.CreateRouteTable(createOpts)
	if err != nil {
		return nil, fmt.Errorf("Error creating route table: %s", err)
	}

	// Get the ID and store it
	rt := &resp.RouteTable
	s.ID = rt.RouteTableId
	log.Printf("[INFO] Route Table ID: %s", s.ID)

	// Update our attributes and return
	return resource_aws_route_table_update_state(s, rt)
}

func resource_aws_route_table_update(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	panic("Update for route table is not supported")

	return nil, nil
}

func resource_aws_route_table_destroy(
	s *terraform.ResourceState,
	meta interface{}) error {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	log.Printf("[INFO] Deleting Route Table: %s", s.ID)
	if _, err := ec2conn.DeleteRouteTable(s.ID); err != nil {
		ec2err, ok := err.(*ec2.Error)
		if ok && ec2err.Code == "InvalidRouteTableID.NotFound" {
			return nil
		}

		return fmt.Errorf("Error deleting route table: %s", err)
	}

	return nil
}

func resource_aws_route_table_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	resp, err := ec2conn.DescribeRouteTables([]string{s.ID}, ec2.NewFilter())
	if err != nil {
		if ec2err, ok := err.(*ec2.Error); ok && ec2err.Code == "InvalidRouteTableID.NotFound" {
			return nil, nil
		}

		log.Printf("[ERROR] Error searching for route table: %s", err)
		return s, err
	}

	if len(resp.RouteTables) == 0 {
		return nil, nil
	}

	rt := &resp.RouteTables[0]
	return resource_aws_route_table_update_state(s, rt)
}

func resource_aws_route_table_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {
	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"vpc_id": diff.AttrTypeCreate,
			"route":  diff.AttrTypeUpdate,
		},
	}

	return b.Diff(s, c)
}

func resource_aws_route_table_update_state(
	s *terraform.ResourceState,
	rt *ec2.RouteTable) (*terraform.ResourceState, error) {
	s.Attributes["vpc_id"] = rt.VpcId

	// We belong to a VPC
	s.Dependencies = []terraform.ResourceDependency{
		terraform.ResourceDependency{ID: rt.VpcId},
	}

	return s, nil
}
