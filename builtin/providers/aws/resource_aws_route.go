package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

// AWS Route resource Schema declaration
func resourceAwsRoute() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsRouteCreate,
		Read:   resourceAwsRouteRead,
		Update: resourceAwsRouteUpdate,
		Delete: resourceAwsRouteDelete,
		Exists: resourceAwsRouteExists,

		Schema: map[string]*schema.Schema{
			"destination_cidr_block": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"destination_prefix_list_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"gateway_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"instance_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"instance_owner_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"network_interface_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"origin": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"state": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"route_table_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"vpc_peering_connection_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceAwsRouteCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	var numTargets int
	var setTarget string
	allowedTargets := []string{
		"gateway_id",
		"instance_id",
		"network_interface_id",
		"vpc_peering_connection_id",
	}

	// Check if more than 1 target is specified
	for _, target := range allowedTargets {
		if len(d.Get(target).(string)) > 0 {
			numTargets++
			setTarget = target
		}
	}

	if numTargets > 1 {
		fmt.Errorf("Error: more than 1 target specified. Only 1 of gateway_id" +
			"instance_id, network_interface_id, route_table_id or" +
			"vpc_peering_connection_id is allowed.")
	}

	createOpts := &ec2.CreateRouteInput{}
	// Formulate CreateRouteInput based on the target type
	switch setTarget {
	case "gateway_id":
		createOpts = &ec2.CreateRouteInput{
			RouteTableId:         aws.String(d.Get("route_table_id").(string)),
			DestinationCidrBlock: aws.String(d.Get("destination_cidr_block").(string)),
			GatewayId:            aws.String(d.Get("gateway_id").(string)),
		}
	case "instance_id":
		createOpts = &ec2.CreateRouteInput{
			RouteTableId:         aws.String(d.Get("route_table_id").(string)),
			DestinationCidrBlock: aws.String(d.Get("destination_cidr_block").(string)),
			InstanceId:           aws.String(d.Get("instance_id").(string)),
		}
	case "network_interface_id":
		createOpts = &ec2.CreateRouteInput{
			RouteTableId:         aws.String(d.Get("route_table_id").(string)),
			DestinationCidrBlock: aws.String(d.Get("destination_cidr_block").(string)),
			NetworkInterfaceId:   aws.String(d.Get("network_interface_id").(string)),
		}
	case "vpc_peering_connection_id":
		createOpts = &ec2.CreateRouteInput{
			RouteTableId:           aws.String(d.Get("route_table_id").(string)),
			DestinationCidrBlock:   aws.String(d.Get("destination_cidr_block").(string)),
			VpcPeeringConnectionId: aws.String(d.Get("vpc_peering_connection_id").(string)),
		}
	default:
		fmt.Errorf("Error: invalid target type specified.")
	}
	log.Printf("[DEBUG] Route create config: %s", awsutil.Prettify(createOpts))

	// Create the route
	_, err := conn.CreateRoute(createOpts)
	if err != nil {
		return fmt.Errorf("Error creating route: %s", err)
	}

	route, err := findResourceRoute(conn, d.Get("route_table_id").(string), d.Get("destination_cidr_block").(string))
	if err != nil {
		fmt.Errorf("Error: %s", awsutil.Prettify(err))
	}

	d.SetId(routeIDHash(d, route))

	return resourceAwsRouteRead(d, meta)
}

func resourceAwsRouteRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	route, err := findResourceRoute(conn, d.Get("route_table_id").(string), d.Get("destination_cidr_block").(string))
	if err != nil {
		return err
	}

	d.Set("destination_prefix_list_id", route.DestinationPrefixListId)
	d.Set("gateway_id", route.GatewayId)
	d.Set("instance_id", route.InstanceId)
	d.Set("instance_owner_id", route.InstanceOwnerId)
	d.Set("network_interface_id", route.NetworkInterfaceId)
	d.Set("origin", route.Origin)
	d.Set("state", route.State)
	d.Set("vpc_peering_connection_id", route.VpcPeeringConnectionId)

	return nil
}

func resourceAwsRouteUpdate(d *schema.ResourceData, meta interface{}) error {
	if d.HasChange("destination_cidr_block") {
		return resourceAwsRouteRecreate(d, meta)
	}

	conn := meta.(*AWSClient).ec2conn
	var numTargets int
	var setTarget string
	allowedTargets := []string{
		"gateway_id",
		"instance_id",
		"network_interface_id",
		"vpc_peering_connection_id",
	}
	replaceOpts := &ec2.ReplaceRouteInput{}

	// Check if more than 1 target is specified
	for _, target := range allowedTargets {
		if len(d.Get(target).(string)) > 0 {
			numTargets++
			setTarget = target
		}
	}

	if numTargets > 1 {
		fmt.Errorf("Error: more than 1 target specified. Only 1 of gateway_id" +
			"instance_id, network_interface_id, route_table_id or" +
			"vpc_peering_connection_id is allowed.")
	}

	// Formulate ReplaceRouteInput based on the target type
	switch setTarget {
	case "gateway_id":
		replaceOpts = &ec2.ReplaceRouteInput{
			RouteTableId:         aws.String(d.Get("route_table_id").(string)),
			DestinationCidrBlock: aws.String(d.Get("destination_cidr_block").(string)),
			GatewayId:            aws.String(d.Get("gateway_id").(string)),
		}
	case "instance_id":
		replaceOpts = &ec2.ReplaceRouteInput{
			RouteTableId:         aws.String(d.Get("route_table_id").(string)),
			DestinationCidrBlock: aws.String(d.Get("destination_cidr_block").(string)),
			InstanceId:           aws.String(d.Get("instance_id").(string)),
			//NOOP: Ensure we don't blow away network interface id that is set after instance is launched
			NetworkInterfaceId:	  aws.String(d.Get("network_interface_id").(string)),
		}
	case "network_interface_id":
		replaceOpts = &ec2.ReplaceRouteInput{
			RouteTableId:         aws.String(d.Get("route_table_id").(string)),
			DestinationCidrBlock: aws.String(d.Get("destination_cidr_block").(string)),
			NetworkInterfaceId:   aws.String(d.Get("network_interface_id").(string)),
		}
	case "vpc_peering_connection_id":
		replaceOpts = &ec2.ReplaceRouteInput{
			RouteTableId:           aws.String(d.Get("route_table_id").(string)),
			DestinationCidrBlock:   aws.String(d.Get("destination_cidr_block").(string)),
			VpcPeeringConnectionId: aws.String(d.Get("vpc_peering_connection_id").(string)),
		}
	default:
		fmt.Errorf("Error: invalid target type specified.")
	}
	log.Printf("[DEBUG] Route replace config: %s", awsutil.Prettify(replaceOpts))

	// Replace the route
	_, err := conn.ReplaceRoute(replaceOpts)
	if err != nil {
		return err
	}

	return nil
}

func resourceAwsRouteRecreate(d *schema.ResourceData, meta interface{}) error {
	//Destination Cidr is used for identification
	// if changed, we should delete the old route, recreate the new route
	conn := meta.(*AWSClient).ec2conn

	oc, _ := d.GetChange("destination_cidr_block")

	var oldRtId interface{}
	if d.HasChange("route_table_id") {
		oldRtId, _ = d.GetChange("route_table_id")
	} else {
		oldRtId = d.Get("route_table_id")
	}

	if err := deleteAwsRoute(conn, oldRtId.(string), oc.(string)); err != nil {
		return err
	}
	d.SetId("")

	return resourceAwsRouteCreate(d, meta)
}

func resourceAwsRouteDelete(d *schema.ResourceData, meta interface{}) error {
	err := deleteAwsRoute(meta.(*AWSClient).ec2conn,
		d.Get("route_table_id").(string), d.Get("destination_cidr_block").(string))
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}

func resourceAwsRouteExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(*AWSClient).ec2conn
	routeTableId := d.Get("route_table_id").(string)

	findOpts := &ec2.DescribeRouteTablesInput{
		RouteTableIds: []*string{&routeTableId},
	}

	res, err := conn.DescribeRouteTables(findOpts)
	if err != nil {
		return false, err
	}

	cidr := d.Get("destination_cidr_block").(string)
	for _, route := range (*res.RouteTables[0]).Routes {
		if *route.DestinationCidrBlock == cidr {
			return true, nil
		}
	}

	return false, nil
}

// Create an ID for a route
func routeIDHash(d *schema.ResourceData, r *ec2.Route) string {
	return fmt.Sprintf("r-%s%d", d.Get("route_table_id").(string), hashcode.String(*r.DestinationCidrBlock))
}

// Helper: retrieve a route
func findResourceRoute(conn *ec2.EC2, rtbid string, cidr string) (*ec2.Route, error) {
	routeTableID := rtbid

	findOpts := &ec2.DescribeRouteTablesInput{
		RouteTableIds: []*string{&routeTableID},
	}

	resp, err := conn.DescribeRouteTables(findOpts)
	if err != nil {
		return nil, err
	}

	for _, route := range (*resp.RouteTables[0]).Routes {
		if *route.DestinationCidrBlock == cidr {
			return route, nil
		}
	}

	return nil, nil
}

func deleteAwsRoute(conn *ec2.EC2, routeTableId string, cidr string) error {
	deleteOpts := &ec2.DeleteRouteInput{
		RouteTableId:         aws.String(routeTableId),
		DestinationCidrBlock: aws.String(cidr),
	}
	log.Printf("[DEBUG] Route delete opts: %s", awsutil.Prettify(deleteOpts))

	resp, err := conn.DeleteRoute(deleteOpts)
	log.Printf("[DEBUG] Route delete result: %s", awsutil.Prettify(resp))
	if err != nil {
		return err
	}
	return nil
}
