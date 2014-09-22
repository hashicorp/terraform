package aws

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/ec2"
)

func resource_aws_vpc_create(
	s *terraform.InstanceState,
	d *terraform.InstanceDiff,
	meta interface{}) (*terraform.InstanceState, error) {
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

	if attr, ok := d.Attributes["enable_dns_support"]; ok {
		options := new(ec2.ModifyVpcAttribute)

		options.EnableDnsSupport = attr.New != "" && attr.New != "false"
		options.SetEnableDnsSupport = true

		s.Attributes["enable_dns_support"] = strconv.FormatBool(options.EnableDnsSupport)

		log.Printf("[INFO] Modifying vpc attributes for %s: %#v", s.ID, options)

		if _, err := ec2conn.ModifyVpcAttribute(s.ID, options); err != nil {
			return s, err
		}
	}

	if attr, ok := d.Attributes["enable_dns_hostnames"]; ok {
		options := new(ec2.ModifyVpcAttribute)

		options.EnableDnsHostnames = attr.New != "" && attr.New != "false"
		options.SetEnableDnsHostnames = true

		s.Attributes["enable_dns_hostnames"] = strconv.FormatBool(options.EnableDnsHostnames)

		log.Printf("[INFO] Modifying enable_dns_hostnames vpc attribute for %s: %#v", s.ID, options)

		if _, err := ec2conn.ModifyVpcAttribute(s.ID, options); err != nil {
			return s, err
		}
	}

	// Update our attributes and return
	return resource_aws_vpc_update_state(s, vpcRaw.(*ec2.VPC))
}

func resource_aws_vpc_update(
	s *terraform.InstanceState,
	d *terraform.InstanceDiff,
	meta interface{}) (*terraform.InstanceState, error) {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn
	rs := s.MergeDiff(d)

	log.Printf("[DEBUG] attributes: %#v", d.Attributes)

	if attr, ok := d.Attributes["enable_dns_support"]; ok {
		options := new(ec2.ModifyVpcAttribute)

		options.EnableDnsSupport = attr.New != "" && attr.New != "false"
		options.SetEnableDnsSupport = true

		rs.Attributes["enable_dns_support"] = strconv.FormatBool(options.EnableDnsSupport)

		log.Printf("[INFO] Modifying enable_dns_support vpc attribute for %s: %#v", s.ID, options)

		if _, err := ec2conn.ModifyVpcAttribute(s.ID, options); err != nil {
			return s, err
		}
	}

	if attr, ok := d.Attributes["enable_dns_hostnames"]; ok {
		options := new(ec2.ModifyVpcAttribute)

		options.EnableDnsHostnames = attr.New != "" && attr.New != "false"
		options.SetEnableDnsHostnames = true

		rs.Attributes["enable_dns_hostnames"] = strconv.FormatBool(options.EnableDnsHostnames)

		log.Printf("[INFO] Modifying enable_dns_hostnames vpc attribute for %s: %#v", s.ID, options)

		if _, err := ec2conn.ModifyVpcAttribute(s.ID, options); err != nil {
			return s, err
		}
	}

	return rs, nil
}

func resource_aws_vpc_destroy(
	s *terraform.InstanceState,
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
	s *terraform.InstanceState,
	meta interface{}) (*terraform.InstanceState, error) {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	vpcRaw, _, err := VPCStateRefreshFunc(ec2conn, s.ID)()
	if err != nil {
		return s, err
	}
	if vpcRaw == nil {
		return nil, nil
	}

	if dnsSupportResp, err := ec2conn.VpcAttribute(s.ID, "enableDnsSupport"); err != nil {
		return s, err
	} else {
		s.Attributes["enable_dns_support"] = strconv.FormatBool(dnsSupportResp.EnableDnsSupport)
	}

	if dnsHostnamesResp, err := ec2conn.VpcAttribute(s.ID, "enableDnsHostnames"); err != nil {
		return s, err
	} else {
		s.Attributes["enable_dns_hostnames"] = strconv.FormatBool(dnsHostnamesResp.EnableDnsHostnames)
	}

	return resource_aws_vpc_update_state(s, vpcRaw.(*ec2.VPC))
}

func resource_aws_vpc_diff(
	s *terraform.InstanceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.InstanceDiff, error) {
	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"cidr_block":           diff.AttrTypeCreate,
			"enable_dns_support":   diff.AttrTypeUpdate,
			"enable_dns_hostnames": diff.AttrTypeUpdate,
		},

		ComputedAttrs: []string{
			"enable_dns_support",
			"enable_dns_hostnames",
		},
	}

	return b.Diff(s, c)
}

func resource_aws_vpc_update_state(
	s *terraform.InstanceState,
	vpc *ec2.VPC) (*terraform.InstanceState, error) {
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
