package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

// Route table import also imports all the rules
func resourceAwsRouteTableImportState(
	d *schema.ResourceData,
	meta interface{}) ([]*schema.ResourceData, error) {
	conn := meta.(*AWSClient).ec2conn

	// First query the resource itself
	id := d.Id()
	resp, err := conn.DescribeRouteTables(&ec2.DescribeRouteTablesInput{
		RouteTableIds: []*string{&id},
	})
	if err != nil {
		return nil, err
	}
	if len(resp.RouteTables) < 1 || resp.RouteTables[0] == nil {
		return nil, fmt.Errorf("route table %s is not found", id)
	}
	table := resp.RouteTables[0]

	// Start building our results
	results := make([]*schema.ResourceData, 1,
		2+len(table.Associations)+len(table.Routes))
	results[0] = d

	{
		// Construct the routes
		subResource := resourceAwsRoute()
		for _, route := range table.Routes {
			// Ignore the local/default route
			if route.GatewayId != nil && *route.GatewayId == "local" {
				continue
			}

			if route.Origin != nil && *route.Origin == "EnableVgwRoutePropagation" {
				continue
			}

			if route.DestinationPrefixListId != nil {
				// Skipping because VPC endpoint routes are handled separately
				// See aws_vpc_endpoint
				continue
			}

			// Minimal data for route
			d := subResource.Data(nil)
			d.SetType("aws_route")
			d.Set("route_table_id", id)
			d.Set("destination_cidr_block", route.DestinationCidrBlock)
			d.Set("destination_ipv6_cidr_block", route.DestinationIpv6CidrBlock)
			d.SetId(routeIDHash(d, route))
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

	return results, nil
}
