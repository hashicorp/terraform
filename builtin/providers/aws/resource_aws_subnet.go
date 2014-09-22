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

func resource_aws_subnet_create(
	s *terraform.InstanceState,
	d *terraform.InstanceDiff,
	meta interface{}) (*terraform.InstanceState, error) {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	// Merge the diff so that we have all the proper attributes
	s = s.MergeDiff(d)

	// Create the Subnet
	createOpts := &ec2.CreateSubnet{
		AvailabilityZone: s.Attributes["availability_zone"],
		CidrBlock:        s.Attributes["cidr_block"],
		VpcId:            s.Attributes["vpc_id"],
	}
	log.Printf("[DEBUG] Subnet create config: %#v", createOpts)
	resp, err := ec2conn.CreateSubnet(createOpts)
	if err != nil {
		return nil, fmt.Errorf("Error creating subnet: %s", err)
	}

	// Get the ID and store it
	subnet := &resp.Subnet
	s.ID = subnet.SubnetId
	log.Printf("[INFO] Subnet ID: %s", s.ID)

	// Wait for the Subnet to become available
	log.Printf(
		"[DEBUG] Waiting for subnet (%s) to become available",
		s.ID)
	stateConf := &resource.StateChangeConf{
		Pending: []string{"pending"},
		Target:  "available",
		Refresh: SubnetStateRefreshFunc(ec2conn, s.ID),
		Timeout: 10 * time.Minute,
	}
	subnetRaw, err := stateConf.WaitForState()
	if err != nil {
		return s, fmt.Errorf(
			"Error waiting for subnet (%s) to become available: %s",
			s.ID, err)
	}

	// Map public ip on launch must be set in another API call
	if attr := s.Attributes["map_public_ip_on_launch"]; attr == "true" {
		modifyOpts := &ec2.ModifySubnetAttribute{
			SubnetId:            s.ID,
			MapPublicIpOnLaunch: true,
		}
		log.Printf("[DEBUG] Subnet modify attributes: %#v", modifyOpts)
		_, err := ec2conn.ModifySubnetAttribute(modifyOpts)
		if err != nil {
			return nil, fmt.Errorf("Error modify subnet attributes: %s", err)
		}
	}

	// Update our attributes and return
	return resource_aws_subnet_update_state(s, subnetRaw.(*ec2.Subnet))
}

func resource_aws_subnet_update(
	s *terraform.InstanceState,
	d *terraform.InstanceDiff,
	meta interface{}) (*terraform.InstanceState, error) {
	// This should never be called because we have no update-able
	// attributes
	panic("Update for subnet is not supported")
}

func resource_aws_subnet_destroy(
	s *terraform.InstanceState,
	meta interface{}) error {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	log.Printf("[INFO] Deleting Subnet: %s", s.ID)
	if _, err := ec2conn.DeleteSubnet(s.ID); err != nil {
		ec2err, ok := err.(*ec2.Error)
		if ok && ec2err.Code == "InvalidSubnetID.NotFound" {
			return nil
		}

		return fmt.Errorf("Error deleting subnet: %s", err)
	}

	// Wait for the Subnet to actually delete
	log.Printf("[DEBUG] Waiting for subnet (%s) to delete", s.ID)
	stateConf := &resource.StateChangeConf{
		Pending: []string{"available", "pending"},
		Target:  "",
		Refresh: SubnetStateRefreshFunc(ec2conn, s.ID),
		Timeout: 10 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf(
			"Error waiting for subnet (%s) to destroy: %s",
			s.ID, err)
	}

	return nil
}

func resource_aws_subnet_refresh(
	s *terraform.InstanceState,
	meta interface{}) (*terraform.InstanceState, error) {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	subnetRaw, _, err := SubnetStateRefreshFunc(ec2conn, s.ID)()
	if err != nil {
		return s, err
	}
	if subnetRaw == nil {
		return nil, nil
	}

	subnet := subnetRaw.(*ec2.Subnet)
	return resource_aws_subnet_update_state(s, subnet)
}

func resource_aws_subnet_diff(
	s *terraform.InstanceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.InstanceDiff, error) {
	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"availability_zone":       diff.AttrTypeCreate,
			"cidr_block":              diff.AttrTypeCreate,
			"vpc_id":                  diff.AttrTypeCreate,
			"map_public_ip_on_launch": diff.AttrTypeCreate,
		},

		ComputedAttrs: []string{
			"availability_zone",
		},
	}

	return b.Diff(s, c)
}

func resource_aws_subnet_update_state(
	s *terraform.InstanceState,
	subnet *ec2.Subnet) (*terraform.InstanceState, error) {
	s.Attributes["availability_zone"] = subnet.AvailabilityZone
	s.Attributes["cidr_block"] = subnet.CidrBlock
	s.Attributes["vpc_id"] = subnet.VpcId
	if subnet.MapPublicIpOnLaunch {
		s.Attributes["map_public_ip_on_launch"] = "true"
	}

	return s, nil
}

// SubnetStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// a Subnet.
func SubnetStateRefreshFunc(conn *ec2.EC2, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.DescribeSubnets([]string{id}, ec2.NewFilter())
		if err != nil {
			if ec2err, ok := err.(*ec2.Error); ok && ec2err.Code == "InvalidSubnetID.NotFound" {
				resp = nil
			} else {
				log.Printf("Error on SubnetStateRefresh: %s", err)
				return nil, "", err
			}
		}

		if resp == nil {
			// Sometimes AWS just has consistency issues and doesn't see
			// our instance yet. Return an empty state.
			return nil, "", nil
		}

		subnet := &resp.Subnets[0]
		return subnet, subnet.State, nil
	}
}
