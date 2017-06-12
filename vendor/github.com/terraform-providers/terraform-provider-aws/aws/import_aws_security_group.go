package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

// Security group import fans out to multiple resources due to the
// security group rules. Instead of creating one resource with nested
// rules, we use the best practices approach of one resource per rule.
func resourceAwsSecurityGroupImportState(
	d *schema.ResourceData,
	meta interface{}) ([]*schema.ResourceData, error) {
	conn := meta.(*AWSClient).ec2conn

	// First query the security group
	sgRaw, _, err := SGStateRefreshFunc(conn, d.Id())()
	if err != nil {
		return nil, err
	}
	if sgRaw == nil {
		return nil, fmt.Errorf("security group not found")
	}
	sg := sgRaw.(*ec2.SecurityGroup)

	// Start building our results
	results := make([]*schema.ResourceData, 1,
		1+len(sg.IpPermissions)+len(sg.IpPermissionsEgress))
	results[0] = d

	// Construct the rules
	permMap := map[string][]*ec2.IpPermission{
		"ingress": sg.IpPermissions,
		"egress":  sg.IpPermissionsEgress,
	}
	for ruleType, perms := range permMap {
		for _, perm := range perms {
			ds, err := resourceAwsSecurityGroupImportStatePerm(sg, ruleType, perm)
			if err != nil {
				return nil, err
			}
			results = append(results, ds...)
		}
	}

	return results, nil
}

func resourceAwsSecurityGroupImportStatePerm(sg *ec2.SecurityGroup, ruleType string, perm *ec2.IpPermission) ([]*schema.ResourceData, error) {
	/*
	   Create a seperate Security Group Rule for:
	   * The collection of IpRanges (cidr_blocks)
	   * The collection of Ipv6Ranges (ipv6_cidr_blocks)
	   * Each individual UserIdGroupPair (source_security_group_id)

	   If, for example, a security group has rules for:
	   * 2 IpRanges
	   * 2 Ipv6Ranges
	   * 2 UserIdGroupPairs

	   This would generate 4 security group rules:
	   * 1 for the collection of IpRanges
	   * 1 for the collection of Ipv6Ranges
	   * 1 for the first UserIdGroupPair
	   * 1 for the second UserIdGroupPair
	*/
	var result []*schema.ResourceData

	if perm.IpRanges != nil {
		p := &ec2.IpPermission{
			FromPort:      perm.FromPort,
			IpProtocol:    perm.IpProtocol,
			PrefixListIds: perm.PrefixListIds,
			ToPort:        perm.ToPort,
			IpRanges:      perm.IpRanges,
		}

		r, err := resourceAwsSecurityGroupImportStatePermPair(sg, ruleType, p)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}

	if perm.Ipv6Ranges != nil {
		p := &ec2.IpPermission{
			FromPort:      perm.FromPort,
			IpProtocol:    perm.IpProtocol,
			PrefixListIds: perm.PrefixListIds,
			ToPort:        perm.ToPort,
			Ipv6Ranges:    perm.Ipv6Ranges,
		}

		r, err := resourceAwsSecurityGroupImportStatePermPair(sg, ruleType, p)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}

	if len(perm.UserIdGroupPairs) > 0 {
		for _, pair := range perm.UserIdGroupPairs {
			p := &ec2.IpPermission{
				FromPort:         perm.FromPort,
				IpProtocol:       perm.IpProtocol,
				PrefixListIds:    perm.PrefixListIds,
				ToPort:           perm.ToPort,
				UserIdGroupPairs: []*ec2.UserIdGroupPair{pair},
			}

			r, err := resourceAwsSecurityGroupImportStatePermPair(sg, ruleType, p)
			if err != nil {
				return nil, err
			}
			result = append(result, r)
		}
	}

	if len(result) == 0 && len(perm.PrefixListIds) > 0 {
		p := &ec2.IpPermission{
			FromPort:      perm.FromPort,
			IpProtocol:    perm.IpProtocol,
			PrefixListIds: perm.PrefixListIds,
			ToPort:        perm.ToPort,
		}

		r, err := resourceAwsSecurityGroupImportStatePermPair(sg, ruleType, p)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}

	return result, nil
}

func resourceAwsSecurityGroupImportStatePermPair(sg *ec2.SecurityGroup, ruleType string, perm *ec2.IpPermission) (*schema.ResourceData, error) {
	// Construct the rule. We do this by populating the absolute
	// minimum necessary for Refresh on the rule to work. This
	// happens to be a lot of fields since they're almost all needed
	// for de-dupping.
	sgId := sg.GroupId
	id := ipPermissionIDHash(*sgId, ruleType, perm)
	ruleResource := resourceAwsSecurityGroupRule()
	d := ruleResource.Data(nil)
	d.SetId(id)
	d.SetType("aws_security_group_rule")
	d.Set("security_group_id", sgId)
	d.Set("type", ruleType)

	// 'self' is false by default. Below, we range over the group ids and set true
	// if the parent sg id is found
	d.Set("self", false)

	if len(perm.UserIdGroupPairs) > 0 {
		s := perm.UserIdGroupPairs[0]

		// Check for Pair that is the same as the Security Group, to denote self.
		// Otherwise, mark the group id in source_security_group_id
		isVPC := sg.VpcId != nil && *sg.VpcId != ""
		if isVPC {
			if *s.GroupId == *sg.GroupId {
				d.Set("self", true)
				// prune the self reference from the UserIdGroupPairs, so we don't
				// have duplicate sg ids (both self and in source_security_group_id)
				perm.UserIdGroupPairs = append(perm.UserIdGroupPairs[:0], perm.UserIdGroupPairs[0+1:]...)
			}
		} else {
			if *s.GroupName == *sg.GroupName {
				d.Set("self", true)
				// prune the self reference from the UserIdGroupPairs, so we don't
				// have duplicate sg ids (both self and in source_security_group_id)
				perm.UserIdGroupPairs = append(perm.UserIdGroupPairs[:0], perm.UserIdGroupPairs[0+1:]...)
			}
		}
	}

	if err := setFromIPPerm(d, sg, perm); err != nil {
		return nil, errwrap.Wrapf("Error importing AWS Security Group: {{err}}", err)
	}

	return d, nil
}
