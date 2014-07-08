package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/flatmap"
	"github.com/mitchellh/goamz/ec2"
)

func resource_aws_security_group_create(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	// Merge the diff into the state so that we have all the attributes
	// properly.
	rs := s.MergeDiff(d)

	securityGroupOpts := ec2.SecurityGroup{
		Name:        rs.Attributes["name"],
		Description: rs.Attributes["description"],
		VpcId:       rs.Attributes["vpc_id"],
	}

	log.Printf("[DEBUG] Security Group create configuration: %#v", securityGroupOpts)
	createResp, err := ec2conn.CreateSecurityGroup(securityGroupOpts)
	if err != nil {
		return nil, fmt.Errorf("Error creating Security Group: %s", err)
	}

	rs.ID = createResp.Id
	group := createResp.SecurityGroup

	log.Printf("[INFO] Security Group ID: %s", rs.ID)

	// Expand the "ingress" array to goamz compat []ec2.IPPerm
	v := flatmap.Expand(rs.Attributes, "ingress").([]interface{})
	ingressRules := expandIPPerms(v)

	// Expand the "egress" array to goamz compat []ec2.IPPerm
	v = flatmap.Expand(rs.Attributes, "egress").([]interface{})
	egressRules := expandIPPerms(v)

	if len(egressRules) > 0 {
		_, err = ec2conn.AuthorizeSecurityGroupEgress(group, egressRules)
		if err != nil {
			return rs, fmt.Errorf("Error authorizing security group egress rules: %s", err)
		}
	}

	if len(egressRules) > 0 {
		_, err = ec2conn.AuthorizeSecurityGroup(group, ingressRules)
		if err != nil {
			return rs, fmt.Errorf("Error authorizing security group ingress rules: %s", err)
		}
	}

	sg, err := resource_aws_security_group_retrieve(rs.ID, ec2conn)
	if err != nil {
		return rs, err
	}

	return resource_aws_security_group_update_state(rs, sg)
}

func resource_aws_security_group_update(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {

	rs := s.MergeDiff(d)
	log.Printf("ResourceDiff: %s", d)
	log.Printf("ResourceState: %s", s)
	log.Printf("Merged: %s", rs)

	return nil, fmt.Errorf("Did not update")
}

func resource_aws_security_group_destroy(
	s *terraform.ResourceState,
	meta interface{}) error {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	log.Printf("[DEBUG] Security Group destroy: %v", s.ID)

	_, err := ec2conn.DeleteSecurityGroup(ec2.SecurityGroup{Id: s.ID})

	if err != nil {
		ec2err, ok := err.(*ec2.Error)
		if ok && ec2err.Code == "InvalidGroup.NotFound" {
			return nil
		}
	}

	return err
}

func resource_aws_security_group_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	sg, err := resource_aws_security_group_retrieve(s.ID, ec2conn)

	if err != nil {
		return s, err
	}

	return resource_aws_security_group_update_state(s, sg)
}

func resource_aws_security_group_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {

	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"name":        diff.AttrTypeCreate,
			"description": diff.AttrTypeCreate,
			"vpc_id":      diff.AttrTypeUpdate,
		},

		ComputedAttrs: []string{
			"owner_id",
		},
	}

	return b.Diff(s, c)
}

func resource_aws_security_group_update_state(
	s *terraform.ResourceState,
	sg *ec2.SecurityGroupInfo) (*terraform.ResourceState, error) {

	log.Println(sg)

	s.Attributes["description"] = sg.Description
	s.Attributes["name"] = sg.Name
	s.Attributes["vpc_id"] = sg.VpcId
	s.Attributes["owner_id"] = sg.OwnerId

	return s, nil
}

// Returns a single sg by it's ID
func resource_aws_security_group_retrieve(id string, ec2conn *ec2.EC2) (*ec2.SecurityGroupInfo, error) {
	sgs := []ec2.SecurityGroup{
		ec2.SecurityGroup{
			Id: id,
		},
	}

	log.Printf("[DEBUG] Security Group describe configuration: %#v", sgs)

	describeGroups, err := ec2conn.SecurityGroups(sgs, nil)

	if err != nil {
		return nil, fmt.Errorf("Error retrieving security groups: %s", err)
	}

	// Verify AWS returned our sg
	if len(describeGroups.Groups) != 1 ||
		describeGroups.Groups[0].Id != id {
		if err != nil {
			return nil, fmt.Errorf("Unable to find security group: %#v", describeGroups.Groups)
		}
	}

	sg := describeGroups.Groups[0]

	return &sg, nil
}
