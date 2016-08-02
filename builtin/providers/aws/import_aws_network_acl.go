package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

// Network ACLs import their rules and associations
func resourceAwsNetworkAclImportState(
	d *schema.ResourceData,
	meta interface{}) ([]*schema.ResourceData, error) {
	conn := meta.(*AWSClient).ec2conn

	// First query the resource itself
	resp, err := conn.DescribeNetworkAcls(&ec2.DescribeNetworkAclsInput{
		NetworkAclIds: []*string{aws.String(d.Id())},
	})
	if err != nil {
		return nil, err
	}
	if resp == nil || len(resp.NetworkAcls) < 1 || resp.NetworkAcls[0] == nil {
		return nil, fmt.Errorf("network ACL %s is not found", d.Id())
	}
	acl := resp.NetworkAcls[0]

	// Start building our results
	results := make([]*schema.ResourceData, 1,
		2+len(acl.Associations)+len(acl.Entries))
	results[0] = d

	/*
		{
			// Construct the entries
			subResource := resourceAwsNetworkAclRule()
			for _, entry := range acl.Entries {
				// Minimal data for route
				d := subResource.Data(nil)
				d.SetType("aws_network_acl_rule")
				d.Set("network_acl_id", acl.NetworkAclId)
				d.Set("rule_number", entry.RuleNumber)
				d.Set("egress", entry.Egress)
				d.Set("protocol", entry.Protocol)
				d.SetId(networkAclIdRuleNumberEgressHash(
					d.Get("network_acl_id").(string),
					d.Get("rule_number").(int),
					d.Get("egress").(bool),
					d.Get("protocol").(string)))
				results = append(results, d)
			}
		}

			{
				// Construct the associations
				subResource := resourceAwsRouteTableAssociation()
				for _, assoc := range table.Associations {
					if *assoc.Main {
						// Ignore
						continue
					}

					// Minimal data for route
					d := subResource.Data(nil)
					d.SetType("aws_route_table_association")
					d.Set("route_table_id", assoc.RouteTableId)
					d.SetId(*assoc.RouteTableAssociationId)
					results = append(results, d)
				}
			}

			{
				// Construct the main associations. We could do this above but
				// I keep this as a separate section since it is a separate resource.
				subResource := resourceAwsMainRouteTableAssociation()
				for _, assoc := range table.Associations {
					if !*assoc.Main {
						// Ignore
						continue
					}

					// Minimal data for route
					d := subResource.Data(nil)
					d.SetType("aws_main_route_table_association")
					d.Set("route_table_id", id)
					d.Set("vpc_id", table.VpcId)
					d.SetId(*assoc.RouteTableAssociationId)
					results = append(results, d)
				}
			}
	*/

	return results, nil
}
