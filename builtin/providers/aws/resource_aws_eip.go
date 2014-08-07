package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/config"
	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/ec2"
)

func resource_aws_eip_create(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	// Merge the diff into the state so that we have all the attributes
	// properly.
	rs := s.MergeDiff(d)

	// By default, we're not in a VPC
	vpc := false
	domainOpt := ""
	if rs.Attributes["vpc"] == "true" {
		vpc = true
		domainOpt = "vpc"
	}

	allocOpts := ec2.AllocateAddress{
		Domain: domainOpt,
	}

	log.Printf("[DEBUG] EIP create configuration: %#v", allocOpts)
	allocResp, err := ec2conn.AllocateAddress(&allocOpts)
	if err != nil {
		return nil, fmt.Errorf("Error creating EIP: %s", err)
	}

	// Assign the eips (unique) allocation id for use later
	// the EIP api has a conditional unique ID (really), so
	// if we're in a VPC we need to save the ID as such, otherwise
	// it defaults to using the public IP
	log.Printf("[DEBUG] EIP Allocate: %#v", allocResp)
	if allocResp.AllocationId != "" {
		rs.ID = allocResp.AllocationId
		rs.Attributes["vpc"] = "true"

	} else {
		rs.ID = allocResp.PublicIp
	}

	log.Printf("[INFO] EIP ID: %s (vpc: %v)", rs.ID, vpc)

	return resource_aws_eip_update(rs, d, meta)
}

func resource_aws_eip_update(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	// Merge the diff into the state so that we have all the attributes
	// properly.
	rs := s.MergeDiff(d)

	vpc := strings.Contains(rs.ID, "eipalloc")

	// If we have an instance to register, do it
	instanceId := rs.Attributes["instance"]

	// Only register with an instance if we have one
	if instanceId != "" {
		assocOpts := ec2.AssociateAddress{
			InstanceId: instanceId,
			PublicIp:   rs.ID,
		}

		// more unique ID conditionals
		if vpc {
			assocOpts = ec2.AssociateAddress{
				InstanceId:   instanceId,
				AllocationId: rs.ID,
				PublicIp:     "",
			}
		}

		log.Printf("[DEBUG] EIP associate configuration: %#v (vpc: %v)", assocOpts, vpc)
		_, err := ec2conn.AssociateAddress(&assocOpts)
		if err != nil {
			return rs, fmt.Errorf("Failure associating instances: %s", err)
		}
	}

	address, err := resource_aws_eip_retrieve_address(rs.ID, vpc, ec2conn)
	if err != nil {
		return rs, err
	}

	return resource_aws_eip_update_state(rs, address)
}

func resource_aws_eip_destroy(
	s *terraform.ResourceState,
	meta interface{}) error {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	var err error
	if strings.Contains(s.ID, "eipalloc") {
		log.Printf("[DEBUG] EIP release (destroy) address allocation: %v", s.ID)
		_, err = ec2conn.ReleaseAddress(s.ID)
		return err
	} else {
		log.Printf("[DEBUG] EIP release (destroy) address: %v", s.ID)
		_, err = ec2conn.ReleasePublicAddress(s.ID)
		return err
	}

	return nil
}

func resource_aws_eip_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	vpc := false
	if s.Attributes["vpc"] == "true" {
		vpc = true
	}

	address, err := resource_aws_eip_retrieve_address(s.ID, vpc, ec2conn)

	if err != nil {
		return s, err
	}

	return resource_aws_eip_update_state(s, address)
}

func resource_aws_eip_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {

	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"vpc":      diff.AttrTypeCreate,
			"instance": diff.AttrTypeUpdate,
		},

		ComputedAttrs: []string{
			"public_ip",
			"private_ip",
		},
	}

	return b.Diff(s, c)
}

func resource_aws_eip_update_state(
	s *terraform.ResourceState,
	address *ec2.Address) (*terraform.ResourceState, error) {

	s.Attributes["private_ip"] = address.PrivateIpAddress
	s.Attributes["public_ip"] = address.PublicIp
	s.Attributes["instance"] = address.InstanceId

	return s, nil
}

// Returns a single address by its ID
func resource_aws_eip_retrieve_address(id string, vpc bool, ec2conn *ec2.EC2) (*ec2.Address, error) {
	// Get the full address description for saving to state for
	// use in other resources
	assocIds := []string{}
	publicIps := []string{}
	if vpc {
		assocIds = []string{id}
	} else {
		publicIps = []string{id}
	}

	log.Printf("[DEBUG] EIP describe configuration: %#v, %#v (vpc: %v)", assocIds, publicIps, vpc)

	describeAddresses, err := ec2conn.Addresses(publicIps, assocIds, nil)

	if err != nil {
		return nil, fmt.Errorf("Error retrieving EIP: %s", err)
	}

	// Verify AWS returned our EIP
	if len(describeAddresses.Addresses) != 1 ||
		describeAddresses.Addresses[0].AllocationId != id ||
		describeAddresses.Addresses[0].PublicIp != id {
		if err != nil {
			return nil, fmt.Errorf("Unable to find EIP: %#v", describeAddresses.Addresses)
		}
	}

	address := describeAddresses.Addresses[0]

	return &address, nil
}

func resource_aws_eip_validation() *config.Validator {
	return &config.Validator{
		Optional: []string{
			"vpc",
			"instance",
		},
	}
}
