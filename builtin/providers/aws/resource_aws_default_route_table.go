package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsDefaultRouteTable() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDefaultRouteTableCreate,
		Read:   resourceAwsDefaultRouteTableRead,
		Update: resourceAwsRouteTableUpdate,
		Delete: resourceAwsDefaultRouteTableDelete,

		Schema: map[string]*schema.Schema{
			"default_route_table_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"vpc_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"propagating_vgws": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"route": &schema.Schema{
				Type:     schema.TypeSet,
				Computed: true,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cidr_block": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"gateway_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"instance_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"nat_gateway_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"vpc_peering_connection_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"network_interface_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
				Set: resourceAwsRouteTableHash,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsDefaultRouteTableCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	filter1 := &ec2.Filter{
		Name:   aws.String("association.main"),
		Values: []*string{aws.String("true")},
	}
	filter2 := &ec2.Filter{
		Name:   aws.String("vpc-id"),
		Values: []*string{aws.String(d.Get("vpc_id").(string))},
	}

	findOpts := &ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{filter1, filter2},
	}

	resp, err := conn.DescribeRouteTables(findOpts)
	if err != nil {
		return err
	}

	if len(resp.RouteTables) < 1 || resp.RouteTables[0] == nil {
		return fmt.Errorf("Default Route table not found")
	}

	rt := resp.RouteTables[0]

	// The Default Route Table for a VPC can change, so use a Resource UniqueID
	// for the ID here.
	d.SetId(resource.UniqueId())
	d.Set("default_route_table_id", rt.RouteTableId)
	d.Set("vpc_id", rt.VpcId)

	// revoke all default and pre-existing routes on the default route table.
	// In the UPDATE method, we'll apply only the rules in the configuration.
	log.Printf("[DEBUG] Revoking default routes for Default Route Table for %s", d.Id())
	if err := revokeAllRouteTableRules(*rt.RouteTableId, meta); err != nil {
		return err
	}

	return resourceAwsRouteTableUpdate(d, meta)
}

func resourceAwsDefaultRouteTableRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	filter1 := &ec2.Filter{
		Name:   aws.String("association.main"),
		Values: []*string{aws.String("true")},
	}
	filter2 := &ec2.Filter{
		Name:   aws.String("vpc-id"),
		Values: []*string{aws.String(d.Get("vpc_id").(string))},
	}

	findOpts := &ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{filter1, filter2},
	}

	resp, err := conn.DescribeRouteTables(findOpts)
	if err != nil {
		return err
	}

	if len(resp.RouteTables) < 1 || resp.RouteTables[0] == nil {
		return fmt.Errorf("Default Route table not found")
	}

	rt := resp.RouteTables[0]
	d.Set("default_route_table_id", rt.RouteTableId)

	propagatingVGWs := make([]string, 0, len(rt.PropagatingVgws))
	for _, vgw := range rt.PropagatingVgws {
		propagatingVGWs = append(propagatingVGWs, *vgw.GatewayId)
	}
	d.Set("propagating_vgws", propagatingVGWs)

	// Create an empty schema.Set to hold all routes
	route := &schema.Set{F: resourceAwsRouteTableHash}

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

		route.Add(m)
	}
	d.Set("route", route)

	// Tags
	d.Set("tags", tagsToMap(rt.Tags))

	return nil
}

func resourceAwsDefaultRouteTableDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[WARN] Cannot destroy Default Route Table. Terraform will remove this resource from the state file, however resources may remain.")
	d.SetId("")
	return nil
}

// revokeAllRouteTableRules revoke all ingress and egress rules that the Default
// Network ACL currently has
func revokeAllRouteTableRules(netaclId string, meta interface{}) error {
	// conn := meta.(*AWSClient).ec2conn
	log.Printf("\n***\nrevokeAllRouteTableRules\n***\n")

	// Disable the propagation as it no longer exists in the config
	// log.Printf(
	// 	"[INFO] Deleting VGW propagation from %s: %s",
	// 	d.Id(), id)
	// _, err := conn.DisableVgwRoutePropagation(&ec2.DisableVgwRoutePropagationInput{
	// 	RouteTableId: aws.String(d.Id()),
	// 	GatewayId:    aws.String(id),
	// })

	// // Delete the route as it no longer exists in the config
	// log.Printf(
	// 	"[INFO] Deleting route from %s: %s",
	// 	d.Id(), m["cidr_block"].(string))
	// _, err := conn.DeleteRoute(&ec2.DeleteRouteInput{
	// 	RouteTableId:         aws.String(d.Id()),
	// 	DestinationCidrBlock: aws.String(m["cidr_block"].(string)),
	// })
	// if err != nil {
	// 	return err
	// }

	return nil
}
