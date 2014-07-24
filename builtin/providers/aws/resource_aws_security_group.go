package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/flatmap"
	"github.com/hashicorp/terraform/helper/config"
	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
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
		Name: rs.Attributes["name"],
	}

	if rs.Attributes["vpc_id"] != "" {
		securityGroupOpts.VpcId = rs.Attributes["vpc_id"]
	}

	if rs.Attributes["description"] != "" {
		securityGroupOpts.Description = rs.Attributes["description"]
	}

	log.Printf("[DEBUG] Security Group create configuration: %#v", securityGroupOpts)
	createResp, err := ec2conn.CreateSecurityGroup(securityGroupOpts)
	if err != nil {
		return nil, fmt.Errorf("Error creating Security Group: %s", err)
	}

	rs.ID = createResp.Id
	group := createResp.SecurityGroup

	log.Printf("[INFO] Security Group ID: %s", rs.ID)

	// Wait for the security group to truly exist
	log.Printf(
		"[DEBUG] Waiting for SG (%s) to exist",
		s.ID)
	stateConf := &resource.StateChangeConf{
		Pending: []string{""},
		Target:  "exists",
		Refresh: SGStateRefreshFunc(ec2conn, rs.ID),
		Timeout: 1 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return s, fmt.Errorf(
			"Error waiting for SG (%s) to become available: %s",
			rs.ID, err)
	}

	// Expand the "ingress" array to goamz compat []ec2.IPPerm
	ingressRules := []ec2.IPPerm{}
	v, ok := flatmap.Expand(rs.Attributes, "ingress").([]interface{})
	if ok {
		ingressRules, err = expandIPPerms(v)
		if err != nil {
			return rs, err
		}
	}

	if len(ingressRules) > 0 {
		_, err = ec2conn.AuthorizeSecurityGroup(group, ingressRules)
		if err != nil {
			return rs, fmt.Errorf("Error authorizing security group ingress rules: %s", err)
		}
	}

	return resource_aws_security_group_refresh(rs, meta)
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

	sgRaw, _, err := SGStateRefreshFunc(ec2conn, s.ID)()
	if err != nil {
		return s, err
	}
	if sgRaw == nil {
		return nil, nil
	}

	return resource_aws_security_group_update_state(
		s, sgRaw.(*ec2.SecurityGroupInfo))
}

func resource_aws_security_group_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {

	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"name":        diff.AttrTypeCreate,
			"description": diff.AttrTypeUpdate,
			"ingress":     diff.AttrTypeUpdate,
			"vpc_id":      diff.AttrTypeCreate,
		},

		ComputedAttrs: []string{
			"owner_id",
			"vpc_id",
		},
	}

	return b.Diff(s, c)
}

func resource_aws_security_group_update_state(
	s *terraform.ResourceState,
	sg *ec2.SecurityGroupInfo) (*terraform.ResourceState, error) {

	s.Attributes["description"] = sg.Description
	s.Attributes["name"] = sg.Name
	s.Attributes["vpc_id"] = sg.VpcId
	s.Attributes["owner_id"] = sg.OwnerId

	// Flatten our ingress values
	toFlatten := make(map[string]interface{})
	toFlatten["ingress"] = flattenIPPerms(sg.IPPerms)

	for k, v := range flatmap.Flatten(toFlatten) {
		s.Attributes[k] = v
	}

	s.Dependencies = nil
	if s.Attributes["vpc_id"] != "" {
		s.Dependencies = append(s.Dependencies,
			terraform.ResourceDependency{ID: s.Attributes["vpc_id"]},
		)
	}

	return s, nil
}

func resource_aws_security_group_validation() *config.Validator {
	return &config.Validator{
		Required: []string{
			"name",
			"ingress.*",
			"ingress.*.from_port",
			"ingress.*.to_port",
			"ingress.*.protocol",
		},
		Optional: []string{
			"description",
			"vpc_id",
			"owner_id",
			"ingress.*.cidr_blocks.*",
			"ingress.*.security_groups.*",
		},
	}
}

// SGStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// a security group.
func SGStateRefreshFunc(conn *ec2.EC2, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		sgs := []ec2.SecurityGroup{ec2.SecurityGroup{Id: id}}
		resp, err := conn.SecurityGroups(sgs, nil)
		if err != nil {
			if ec2err, ok := err.(*ec2.Error); ok {
				if ec2err.Code == "InvalidSecurityGroupID.NotFound" ||
					ec2err.Code == "InvalidGroup.NotFound" {
					resp = nil
					err = nil
				}
			}

			if err != nil {
				log.Printf("Error on SGStateRefresh: %s", err)
				return nil, "", err
			}
		}

		if resp == nil {
			return nil, "", nil
		}

		group := &resp.Groups[0]
		return group, "exists", nil
	}
}
