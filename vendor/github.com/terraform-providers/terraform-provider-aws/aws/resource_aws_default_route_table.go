package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsDefaultRouteTable() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDefaultRouteTableCreate,
		Read:   resourceAwsDefaultRouteTableRead,
		Update: resourceAwsRouteTableUpdate,
		Delete: resourceAwsDefaultRouteTableDelete,

		Schema: map[string]*schema.Schema{
			"default_route_table_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"vpc_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"propagating_vgws": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"route": {
				Type:     schema.TypeSet,
				Computed: true,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cidr_block": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"ipv6_cidr_block": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"egress_only_gateway_id": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"gateway_id": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"instance_id": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"nat_gateway_id": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"transit_gateway_id": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"vpc_peering_connection_id": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"network_interface_id": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
				Set: resourceAwsRouteTableHash,
			},

			"tags": tagsSchema(),

			"owner_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsDefaultRouteTableCreate(d *schema.ResourceData, meta interface{}) error {
	d.SetId(d.Get("default_route_table_id").(string))

	conn := meta.(*AWSClient).ec2conn
	rtRaw, _, err := resourceAwsRouteTableStateRefreshFunc(conn, d.Id())()
	if err != nil {
		return err
	}
	if rtRaw == nil {
		log.Printf("[WARN] Default Route Table not found")
		d.SetId("")
		return nil
	}

	rt := rtRaw.(*ec2.RouteTable)

	d.Set("vpc_id", rt.VpcId)

	// revoke all default and pre-existing routes on the default route table.
	// In the UPDATE method, we'll apply only the rules in the configuration.
	log.Printf("[DEBUG] Revoking default routes for Default Route Table for %s", d.Id())
	if err := revokeAllRouteTableRules(d.Id(), meta); err != nil {
		return err
	}

	return resourceAwsRouteTableUpdate(d, meta)
}

func resourceAwsDefaultRouteTableRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	// look up default route table for VPC
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
	d.SetId(*rt.RouteTableId)

	// re-use regular AWS Route Table READ. This is an extra API call but saves us
	// from trying to manually keep parity
	return resourceAwsRouteTableRead(d, meta)
}

func resourceAwsDefaultRouteTableDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[WARN] Cannot destroy Default Route Table. Terraform will remove this resource from the state file, however resources may remain.")
	return nil
}

// revokeAllRouteTableRules revoke all routes on the Default Route Table
// This should only be ran once at creation time of this resource
func revokeAllRouteTableRules(defaultRouteTableId string, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	log.Printf("\n***\nrevokeAllRouteTableRules\n***\n")

	resp, err := conn.DescribeRouteTables(&ec2.DescribeRouteTablesInput{
		RouteTableIds: []*string{aws.String(defaultRouteTableId)},
	})
	if err != nil {
		return err
	}

	if len(resp.RouteTables) < 1 || resp.RouteTables[0] == nil {
		return fmt.Errorf("Default Route table not found")
	}

	rt := resp.RouteTables[0]

	// Remove all Gateway association
	for _, r := range rt.PropagatingVgws {
		log.Printf(
			"[INFO] Deleting VGW propagation from %s: %s",
			defaultRouteTableId, *r.GatewayId)
		_, err := conn.DisableVgwRoutePropagation(&ec2.DisableVgwRoutePropagationInput{
			RouteTableId: aws.String(defaultRouteTableId),
			GatewayId:    r.GatewayId,
		})
		if err != nil {
			return err
		}
	}

	// Delete all routes
	for _, r := range rt.Routes {
		// you cannot delete the local route
		if r.GatewayId != nil && *r.GatewayId == "local" {
			continue
		}
		if r.DestinationPrefixListId != nil {
			// Skipping because VPC endpoint routes are handled separately
			// See aws_vpc_endpoint
			continue
		}

		if r.DestinationCidrBlock != nil {
			log.Printf(
				"[INFO] Deleting route from %s: %s",
				defaultRouteTableId, *r.DestinationCidrBlock)
			_, err := conn.DeleteRoute(&ec2.DeleteRouteInput{
				RouteTableId:         aws.String(defaultRouteTableId),
				DestinationCidrBlock: r.DestinationCidrBlock,
			})
			if err != nil {
				return err
			}
		}

		if r.DestinationIpv6CidrBlock != nil {
			log.Printf(
				"[INFO] Deleting route from %s: %s",
				defaultRouteTableId, *r.DestinationIpv6CidrBlock)
			_, err := conn.DeleteRoute(&ec2.DeleteRouteInput{
				RouteTableId:             aws.String(defaultRouteTableId),
				DestinationIpv6CidrBlock: r.DestinationIpv6CidrBlock,
			})
			if err != nil {
				return err
			}
		}

	}

	return nil
}
