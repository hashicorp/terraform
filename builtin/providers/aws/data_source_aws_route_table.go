package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsRouteTable() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsRouteTableRead,

		Schema: map[string]*schema.Schema{
			"subnet_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"vpc_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"filter": ec2CustomFiltersSchema(),
			"tags":   tagsSchemaComputed(),
			"routes": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cidr_block": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"gateway_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"instance_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"nat_gateway_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"vpc_peering_connection_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"network_interface_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"associations": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"route_table_association_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"route_table_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"subnet_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"main": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceAwsRouteTableRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	req := &ec2.DescribeRouteTablesInput{}

	req.Filters = buildEC2AttributeFilterList(
		map[string]string{
			"vpc-id":                d.Get("vpc_id").(string),
			"association.subnet-id": d.Get("subnet_id").(string),
		},
	)
	req.Filters = append(req.Filters, buildEC2TagFilterList(
		tagsFromMap(d.Get("tags").(map[string]interface{})),
	)...)
	req.Filters = append(req.Filters, buildEC2CustomFilterList(
		d.Get("filter").(*schema.Set),
	)...)
	if len(req.Filters) == 0 {
		return fmt.Errorf("At least One filter should be setted")
	}

	log.Printf("[DEBUG] Describe Route Tables %v\n", req)
	resp, err := conn.DescribeRouteTables(req)
	if err != nil {
		return err
	}
	if resp == nil || len(resp.RouteTables) == 0 {
		return fmt.Errorf("no matching Route Table found")
	}
	if len(resp.RouteTables) > 1 {
		return fmt.Errorf("multiple Route Table matched; use additional constraints to reduce matches to a single Route Table")
	}

	rt := resp.RouteTables[0]

	d.SetId(*rt.RouteTableId)
	d.Set("vpc_id", rt.VpcId)
	d.Set("tags", tagsToMap(rt.Tags))

	routes := make([]map[string]interface{}, 0, len(rt.Routes))
	// Loop through the routes and add them to the set
	for _, r := range rt.Routes {
		if r.GatewayId != nil && *r.GatewayId == "local" {
			continue
		}

		if r.Origin != nil && *r.Origin == "EnableVgwRoutePropagation" {
			continue
		}

		if r.DestinationPrefixListId != nil {
			// Skipping because VPC endpoint routes are handled separately
			// See aws_vpc_endpoint
			continue
		}

		m := make(map[string]interface{})

		if r.DestinationCidrBlock != nil {
			m["cidr_block"] = *r.DestinationCidrBlock
		}
		if r.GatewayId != nil {
			m["gateway_id"] = *r.GatewayId
		}
		if r.NatGatewayId != nil {
			m["nat_gateway_id"] = *r.NatGatewayId
		}
		if r.InstanceId != nil {
			m["instance_id"] = *r.InstanceId
		}
		if r.VpcPeeringConnectionId != nil {
			m["vpc_peering_connection_id"] = *r.VpcPeeringConnectionId
		}
		if r.NetworkInterfaceId != nil {
			m["network_interface_id"] = *r.NetworkInterfaceId
		}

		routes = append(routes, m)
	}
	d.Set("routes", routes)

	associations := make([]map[string]interface{}, 0, len(rt.Associations))
	// Loop through the routes and add them to the set
	for _, a := range rt.Associations {

		m := make(map[string]interface{})
		m["route_table_id"] = *a.RouteTableId
		m["route_table_association_id"] = *a.RouteTableAssociationId
		m["subnet_id"] = *a.SubnetId
		m["main"] = *a.Main
		associations = append(associations, m)
	}
	d.Set("associations", associations)

	return nil
}
