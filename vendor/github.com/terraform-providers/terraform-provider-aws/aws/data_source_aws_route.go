package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsRoute() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsRouteRead,

		Schema: map[string]*schema.Schema{
			"route_table_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"destination_cidr_block": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"destination_ipv6_cidr_block": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"egress_only_gateway_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"gateway_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"instance_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"nat_gateway_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"vpc_peering_connection_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"network_interface_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsRouteRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	req := &ec2.DescribeRouteTablesInput{}
	rtbId := d.Get("route_table_id")
	cidr := d.Get("destination_cidr_block")
	ipv6Cidr := d.Get("destination_ipv6_cidr_block")

	req.Filters = buildEC2AttributeFilterList(
		map[string]string{
			"route-table-id":                    rtbId.(string),
			"route.destination-cidr-block":      cidr.(string),
			"route.destination-ipv6-cidr-block": ipv6Cidr.(string),
		},
	)

	log.Printf("[DEBUG] Reading Route Table: %s", req)
	resp, err := conn.DescribeRouteTables(req)
	if err != nil {
		return err
	}
	if resp == nil || len(resp.RouteTables) == 0 {
		return fmt.Errorf("Your query returned no results. Please change your search criteria and try again.")
	}
	if len(resp.RouteTables) > 1 {
		return fmt.Errorf("Your query returned more than one route table. Please change your search criteria and try again.")
	}

	results := getRoutes(resp.RouteTables[0], d)

	if len(results) == 0 {
		return fmt.Errorf("No routes matching supplied arguments found in table(s)")
	}
	if len(results) > 1 {
		return fmt.Errorf("Multiple routes matched; use additional constraints to reduce matches to a single route")
	}
	route := results[0]

	d.SetId(resourceAwsRouteID(d, route)) // using function from "resource_aws_route.go"
	d.Set("destination_cidr_block", route.DestinationCidrBlock)
	d.Set("destination_ipv6_cidr_block", route.DestinationIpv6CidrBlock)
	d.Set("egress_only_gateway_id", route.EgressOnlyInternetGatewayId)
	d.Set("gateway_id", route.GatewayId)
	d.Set("instance_id", route.InstanceId)
	d.Set("nat_gateway_id", route.NatGatewayId)
	d.Set("vpc_peering_connection_id", route.VpcPeeringConnectionId)
	d.Set("network_interface_id", route.NetworkInterfaceId)

	return nil
}

func getRoutes(table *ec2.RouteTable, d *schema.ResourceData) []*ec2.Route {
	ec2Routes := table.Routes
	routes := make([]*ec2.Route, 0, len(ec2Routes))
	// Loop through the routes and add them to the set
	for _, r := range ec2Routes {

		if r.Origin != nil && *r.Origin == "EnableVgwRoutePropagation" {
			continue
		}

		if r.DestinationPrefixListId != nil {
			// Skipping because VPC endpoint routes are handled separately
			// See aws_vpc_endpoint
			continue
		}

		if v, ok := d.GetOk("destination_cidr_block"); ok {
			if r.DestinationCidrBlock == nil || *r.DestinationCidrBlock != v.(string) {
				continue
			}
		}

		if v, ok := d.GetOk("destination_ipv6_cidr_block"); ok {
			if r.DestinationIpv6CidrBlock == nil || *r.DestinationIpv6CidrBlock != v.(string) {
				continue
			}
		}

		if v, ok := d.GetOk("egress_only_gateway_id"); ok {
			if r.EgressOnlyInternetGatewayId == nil || *r.EgressOnlyInternetGatewayId != v.(string) {
				continue
			}
		}

		if v, ok := d.GetOk("gateway_id"); ok {
			if r.GatewayId == nil || *r.GatewayId != v.(string) {
				continue
			}
		}

		if v, ok := d.GetOk("instance_id"); ok {
			if r.InstanceId == nil || *r.InstanceId != v.(string) {
				continue
			}
		}

		if v, ok := d.GetOk("nat_gateway_id"); ok {
			if r.NatGatewayId == nil || *r.NatGatewayId != v.(string) {
				continue
			}
		}

		if v, ok := d.GetOk("vpc_peering_connection_id"); ok {
			if r.VpcPeeringConnectionId == nil || *r.VpcPeeringConnectionId != v.(string) {
				continue
			}
		}

		if v, ok := d.GetOk("network_interface_id"); ok {
			if r.NetworkInterfaceId == nil || *r.NetworkInterfaceId != v.(string) {
				continue
			}
		}
		routes = append(routes, r)
	}
	return routes
}
