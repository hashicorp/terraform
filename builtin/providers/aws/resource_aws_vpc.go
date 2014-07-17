package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/ec2"
)

func resource_aws_vpc_create(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	// Merge the diff so that we have all the proper attributes
	s = s.MergeDiff(d)

	// Create the VPC
	createOpts := &ec2.CreateVpc{
		CidrBlock: s.Attributes["cidr_block"],
	}
	log.Printf("[DEBUG] VPC create config: %#v", createOpts)
	vpcResp, err := ec2conn.CreateVpc(createOpts)
	if err != nil {
		return nil, fmt.Errorf("Error creating VPC: %s", err)
	}

	// Get the ID and store it
	vpc := &vpcResp.VPC
	log.Printf("[INFO] VPC ID: %s", vpc.VpcId)
	s.ID = vpc.VpcId

	// Wait for the VPC to become available
	log.Printf(
		"[DEBUG] Waiting for VPC (%s) to become available",
		s.ID)
	stateConf := &resource.StateChangeConf{
		Pending: []string{"pending"},
		Target:  "available",
		Refresh: VPCStateRefreshFunc(ec2conn, s.ID),
		Timeout: 10 * time.Minute,
	}
	vpcRaw, err := stateConf.WaitForState()
	if err != nil {
		return s, fmt.Errorf(
			"Error waiting for VPC (%s) to become available: %s",
			s.ID, err)
	}

	// Update our attributes and return
	return resource_aws_vpc_update_state(s, vpcRaw.(*ec2.VPC))
}

func resource_aws_vpc_update(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	// This should never be called because we have no update-able
	// attributes
	panic("Update for VPC is not supported")

	return nil, nil
}

func resource_aws_vpc_destroy(
	s *terraform.ResourceState,
	meta interface{}) error {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	log.Printf("[INFO] Deleting VPC: %s", s.ID)
	if _, err := ec2conn.DeleteVpc(s.ID); err != nil {
		ec2err, ok := err.(*ec2.Error)
		if ok && ec2err.Code == "InvalidVpcID.NotFound" {
			return nil
		}

		return fmt.Errorf("Error deleting VPC: %s", err)
	}

	return nil
}

func resource_aws_vpc_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	vpcRaw, _, err := VPCStateRefreshFunc(ec2conn, s.ID)()
	if err != nil {
		return s, err
	}
	if vpcRaw == nil {
		return nil, nil
	}

	vpc := vpcRaw.(*ec2.VPC)
	return resource_aws_vpc_update_state(s, vpc)
}

func resource_aws_vpc_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {
	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"cidr_block": diff.AttrTypeCreate,
		},
	}

	return b.Diff(s, c)
}

func resource_aws_vpc_update_state(
	s *terraform.ResourceState,
	vpc *ec2.VPC) (*terraform.ResourceState, error) {
	s.Attributes["cidr_block"] = vpc.CidrBlock
	return s, nil
}

// VPCStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// a VPC.
func VPCStateRefreshFunc(conn *ec2.EC2, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.DescribeVpcs([]string{id}, ec2.NewFilter())
		if err != nil {
			if ec2err, ok := err.(*ec2.Error); ok && ec2err.Code == "InvalidVpcID.NotFound" {
				resp = nil
			} else {
				log.Printf("Error on VPCStateRefresh: %s", err)
				return nil, "", err
			}
		}

		if resp == nil {
			// Sometimes AWS just has consistency issues and doesn't see
			// our instance yet. Return an empty state.
			return nil, "", nil
		}

		vpc := &resp.VPCs[0]
		return vpc, vpc.State, nil
	}
}
