package aws

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

// How long to sleep if a limit-exceeded event happens
var routeTargetValidationError = errors.New("Error: more than 1 target specified. Only 1 of gateway_id, " +
	"egress_only_gateway_id, nat_gateway_id, instance_id, network_interface_id or " +
	"vpc_peering_connection_id is allowed.")

// AWS Route resource Schema declaration
func resourceAwsRoute() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsRouteCreate,
		Read:   resourceAwsRouteRead,
		Update: resourceAwsRouteUpdate,
		Delete: resourceAwsRouteDelete,
		Exists: resourceAwsRouteExists,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(2 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"destination_cidr_block": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"destination_ipv6_cidr_block": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"destination_prefix_list_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"gateway_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"egress_only_gateway_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"nat_gateway_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"instance_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"instance_owner_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"network_interface_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"origin": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"state": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"route_table_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"vpc_peering_connection_id": {
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
		"egress_only_gateway_id",
		"gateway_id",
		"nat_gateway_id",
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
		return routeTargetValidationError
	}

	createOpts := &ec2.CreateRouteInput{}
	// Formulate CreateRouteInput based on the target type
	switch setTarget {
	case "gateway_id":
		createOpts = &ec2.CreateRouteInput{
			RouteTableId: aws.String(d.Get("route_table_id").(string)),
			GatewayId:    aws.String(d.Get("gateway_id").(string)),
		}

		if v, ok := d.GetOk("destination_cidr_block"); ok {
			createOpts.DestinationCidrBlock = aws.String(v.(string))
		}

		if v, ok := d.GetOk("destination_ipv6_cidr_block"); ok {
			createOpts.DestinationIpv6CidrBlock = aws.String(v.(string))
		}

	case "egress_only_gateway_id":
		createOpts = &ec2.CreateRouteInput{
			RouteTableId:                aws.String(d.Get("route_table_id").(string)),
			DestinationIpv6CidrBlock:    aws.String(d.Get("destination_ipv6_cidr_block").(string)),
			EgressOnlyInternetGatewayId: aws.String(d.Get("egress_only_gateway_id").(string)),
		}
	case "nat_gateway_id":
		createOpts = &ec2.CreateRouteInput{
			RouteTableId:         aws.String(d.Get("route_table_id").(string)),
			DestinationCidrBlock: aws.String(d.Get("destination_cidr_block").(string)),
			NatGatewayId:         aws.String(d.Get("nat_gateway_id").(string)),
		}
	case "instance_id":
		createOpts = &ec2.CreateRouteInput{
			RouteTableId: aws.String(d.Get("route_table_id").(string)),
			InstanceId:   aws.String(d.Get("instance_id").(string)),
		}

		if v, ok := d.GetOk("destination_cidr_block"); ok {
			createOpts.DestinationCidrBlock = aws.String(v.(string))
		}

		if v, ok := d.GetOk("destination_ipv6_cidr_block"); ok {
			createOpts.DestinationIpv6CidrBlock = aws.String(v.(string))
		}

	case "network_interface_id":
		createOpts = &ec2.CreateRouteInput{
			RouteTableId:       aws.String(d.Get("route_table_id").(string)),
			NetworkInterfaceId: aws.String(d.Get("network_interface_id").(string)),
		}

		if v, ok := d.GetOk("destination_cidr_block"); ok {
			createOpts.DestinationCidrBlock = aws.String(v.(string))
		}

		if v, ok := d.GetOk("destination_ipv6_cidr_block"); ok {
			createOpts.DestinationIpv6CidrBlock = aws.String(v.(string))
		}

	case "vpc_peering_connection_id":
		createOpts = &ec2.CreateRouteInput{
			RouteTableId:           aws.String(d.Get("route_table_id").(string)),
			VpcPeeringConnectionId: aws.String(d.Get("vpc_peering_connection_id").(string)),
		}

		if v, ok := d.GetOk("destination_cidr_block"); ok {
			createOpts.DestinationCidrBlock = aws.String(v.(string))
		}

		if v, ok := d.GetOk("destination_ipv6_cidr_block"); ok {
			createOpts.DestinationIpv6CidrBlock = aws.String(v.(string))
		}

	default:
		return fmt.Errorf("A valid target type is missing. Specify one of the following attributes: %s", strings.Join(allowedTargets, ", "))
	}
	log.Printf("[DEBUG] Route create config: %s", createOpts)

	// Create the route
	var err error

	err = resource.Retry(d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		_, err = conn.CreateRoute(createOpts)

		if err != nil {
			ec2err, ok := err.(awserr.Error)
			if !ok {
				return resource.NonRetryableError(err)
			}
			if ec2err.Code() == "InvalidParameterException" {
				log.Printf("[DEBUG] Trying to create route again: %q", ec2err.Message())
				return resource.RetryableError(err)
			}

			return resource.NonRetryableError(err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("Error creating route: %s", err)
	}

	var route *ec2.Route

	if v, ok := d.GetOk("destination_cidr_block"); ok {
		err = resource.Retry(d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
			route, err = findResourceRoute(conn, d.Get("route_table_id").(string), v.(string), "")
			return resource.RetryableError(err)
		})
		if err != nil {
			return fmt.Errorf("Error finding route after creating it: %s", err)
		}
	}

	if v, ok := d.GetOk("destination_ipv6_cidr_block"); ok {
		err = resource.Retry(d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
			route, err = findResourceRoute(conn, d.Get("route_table_id").(string), "", v.(string))
			return resource.RetryableError(err)
		})
		if err != nil {
			return fmt.Errorf("Error finding route after creating it: %s", err)
		}
	}

	d.SetId(routeIDHash(d, route))
	resourceAwsRouteSetResourceData(d, route)
	return nil
}

func resourceAwsRouteRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	routeTableId := d.Get("route_table_id").(string)

	destinationCidrBlock := d.Get("destination_cidr_block").(string)
	destinationIpv6CidrBlock := d.Get("destination_ipv6_cidr_block").(string)

	route, err := findResourceRoute(conn, routeTableId, destinationCidrBlock, destinationIpv6CidrBlock)
	if err != nil {
		if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "InvalidRouteTableID.NotFound" {
			log.Printf("[WARN] Route Table %q could not be found. Removing Route from state.",
				routeTableId)
			d.SetId("")
			return nil
		}
		return err
	}
	resourceAwsRouteSetResourceData(d, route)
	return nil
}

func resourceAwsRouteSetResourceData(d *schema.ResourceData, route *ec2.Route) {
	d.Set("destination_prefix_list_id", route.DestinationPrefixListId)
	d.Set("gateway_id", route.GatewayId)
	d.Set("egress_only_gateway_id", route.EgressOnlyInternetGatewayId)
	d.Set("nat_gateway_id", route.NatGatewayId)
	d.Set("instance_id", route.InstanceId)
	d.Set("instance_owner_id", route.InstanceOwnerId)
	d.Set("network_interface_id", route.NetworkInterfaceId)
	d.Set("origin", route.Origin)
	d.Set("state", route.State)
	d.Set("vpc_peering_connection_id", route.VpcPeeringConnectionId)
}

func resourceAwsRouteUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	var numTargets int
	var setTarget string

	allowedTargets := []string{
		"egress_only_gateway_id",
		"gateway_id",
		"nat_gateway_id",
		"network_interface_id",
		"instance_id",
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

	switch setTarget {
	//instance_id is a special case due to the fact that AWS will "discover" the network_interace_id
	//when it creates the route and return that data.  In the case of an update, we should ignore the
	//existing network_interface_id
	case "instance_id":
		if numTargets > 2 || (numTargets == 2 && len(d.Get("network_interface_id").(string)) == 0) {
			return routeTargetValidationError
		}
	default:
		if numTargets > 1 {
			return routeTargetValidationError
		}
	}

	// Formulate ReplaceRouteInput based on the target type
	switch setTarget {
	case "gateway_id":
		replaceOpts = &ec2.ReplaceRouteInput{
			RouteTableId:         aws.String(d.Get("route_table_id").(string)),
			DestinationCidrBlock: aws.String(d.Get("destination_cidr_block").(string)),
			GatewayId:            aws.String(d.Get("gateway_id").(string)),
		}
	case "egress_only_gateway_id":
		replaceOpts = &ec2.ReplaceRouteInput{
			RouteTableId:                aws.String(d.Get("route_table_id").(string)),
			DestinationIpv6CidrBlock:    aws.String(d.Get("destination_ipv6_cidr_block").(string)),
			EgressOnlyInternetGatewayId: aws.String(d.Get("egress_only_gateway_id").(string)),
		}
	case "nat_gateway_id":
		replaceOpts = &ec2.ReplaceRouteInput{
			RouteTableId:         aws.String(d.Get("route_table_id").(string)),
			DestinationCidrBlock: aws.String(d.Get("destination_cidr_block").(string)),
			NatGatewayId:         aws.String(d.Get("nat_gateway_id").(string)),
		}
	case "instance_id":
		replaceOpts = &ec2.ReplaceRouteInput{
			RouteTableId:         aws.String(d.Get("route_table_id").(string)),
			DestinationCidrBlock: aws.String(d.Get("destination_cidr_block").(string)),
			InstanceId:           aws.String(d.Get("instance_id").(string)),
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
		return fmt.Errorf("An invalid target type specified: %s", setTarget)
	}
	log.Printf("[DEBUG] Route replace config: %s", replaceOpts)

	// Replace the route
	_, err := conn.ReplaceRoute(replaceOpts)
	if err != nil {
		return err
	}

	return nil
}

func resourceAwsRouteDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	deleteOpts := &ec2.DeleteRouteInput{
		RouteTableId: aws.String(d.Get("route_table_id").(string)),
	}
	if v, ok := d.GetOk("destination_cidr_block"); ok {
		deleteOpts.DestinationCidrBlock = aws.String(v.(string))
	}
	if v, ok := d.GetOk("destination_ipv6_cidr_block"); ok {
		deleteOpts.DestinationIpv6CidrBlock = aws.String(v.(string))
	}
	log.Printf("[DEBUG] Route delete opts: %s", deleteOpts)

	var err error
	err = resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		log.Printf("[DEBUG] Trying to delete route with opts %s", deleteOpts)
		resp, err := conn.DeleteRoute(deleteOpts)
		log.Printf("[DEBUG] Route delete result: %s", resp)

		if err == nil {
			return nil
		}

		ec2err, ok := err.(awserr.Error)
		if !ok {
			return resource.NonRetryableError(err)
		}
		if ec2err.Code() == "InvalidParameterException" {
			log.Printf("[DEBUG] Trying to delete route again: %q",
				ec2err.Message())
			return resource.RetryableError(err)
		}

		return resource.NonRetryableError(err)
	})

	if err != nil {
		return err
	}

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
		if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "InvalidRouteTableID.NotFound" {
			log.Printf("[WARN] Route Table %q could not be found.", routeTableId)
			return false, nil
		}
		return false, fmt.Errorf("Error while checking if route exists: %s", err)
	}

	if len(res.RouteTables) < 1 || res.RouteTables[0] == nil {
		log.Printf("[WARN] Route Table %q is gone, or route does not exist.",
			routeTableId)
		return false, nil
	}

	if v, ok := d.GetOk("destination_cidr_block"); ok {
		for _, route := range (*res.RouteTables[0]).Routes {
			if route.DestinationCidrBlock != nil && *route.DestinationCidrBlock == v.(string) {
				return true, nil
			}
		}
	}

	if v, ok := d.GetOk("destination_ipv6_cidr_block"); ok {
		for _, route := range (*res.RouteTables[0]).Routes {
			if route.DestinationIpv6CidrBlock != nil && *route.DestinationIpv6CidrBlock == v.(string) {
				return true, nil
			}
		}
	}

	return false, nil
}

// Create an ID for a route
func routeIDHash(d *schema.ResourceData, r *ec2.Route) string {

	if r.DestinationIpv6CidrBlock != nil && *r.DestinationIpv6CidrBlock != "" {
		return fmt.Sprintf("r-%s%d", d.Get("route_table_id").(string), hashcode.String(*r.DestinationIpv6CidrBlock))
	}

	return fmt.Sprintf("r-%s%d", d.Get("route_table_id").(string), hashcode.String(*r.DestinationCidrBlock))
}

// Helper: retrieve a route
func findResourceRoute(conn *ec2.EC2, rtbid string, cidr string, ipv6cidr string) (*ec2.Route, error) {
	routeTableID := rtbid

	findOpts := &ec2.DescribeRouteTablesInput{
		RouteTableIds: []*string{&routeTableID},
	}

	resp, err := conn.DescribeRouteTables(findOpts)
	if err != nil {
		return nil, err
	}

	if len(resp.RouteTables) < 1 || resp.RouteTables[0] == nil {
		return nil, fmt.Errorf("Route Table %q is gone, or route does not exist.",
			routeTableID)
	}

	if cidr != "" {
		for _, route := range (*resp.RouteTables[0]).Routes {
			if route.DestinationCidrBlock != nil && *route.DestinationCidrBlock == cidr {
				return route, nil
			}
		}

		return nil, fmt.Errorf("Unable to find matching route for Route Table (%s) "+
			"and destination CIDR block (%s).", rtbid, cidr)
	}

	if ipv6cidr != "" {
		for _, route := range (*resp.RouteTables[0]).Routes {
			if route.DestinationIpv6CidrBlock != nil && *route.DestinationIpv6CidrBlock == ipv6cidr {
				return route, nil
			}
		}

		return nil, fmt.Errorf("Unable to find matching route for Route Table (%s) "+
			"and destination IPv6 CIDR block (%s).", rtbid, ipv6cidr)
	}

	return nil, fmt.Errorf("When trying to find a matching route for Route Table %q "+
		"you need to specify a CIDR block of IPv6 CIDR Block", rtbid)

}
