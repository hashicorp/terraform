package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/ec2"
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
	sgId := d.Id()

	// Start building our results
	results := make([]*schema.ResourceData, 1,
		1+len(sg.IpPermissions)+len(sg.IpPermissionsEgress))
	results[0] = d

	// Construct the rules
	ruleResource := resourceAwsSecurityGroupRule()
	permMap := map[string][]*ec2.IpPermission{
		"ingress": sg.IpPermissions,
		"egress":  sg.IpPermissionsEgress,
	}
	for ruleType, perms := range permMap {
		for _, perm := range perms {
			// Construct the rule. We do this by populating the absolute
			// minimum necessary for Refresh on the rule to work.
			id := ipPermissionIDHash(sgId, ruleType, perm)
			data := ruleResource.Data(nil)
			data.SetId(id)
			data.SetType("aws_security_group_rule")
			data.Set("security_group_id", sgId)
			data.Set("type", ruleType)
			results = append(results, data)
		}
	}

	return results, nil
}
