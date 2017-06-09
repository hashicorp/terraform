package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsVpnConnectionRoute() *schema.Resource {
	return &schema.Resource{
		// You can't update a route. You can just delete one and make
		// a new one.
		Create: resourceAwsVpnConnectionRouteCreate,
		Read:   resourceAwsVpnConnectionRouteRead,
		Delete: resourceAwsVpnConnectionRouteDelete,

		Schema: map[string]*schema.Schema{
			"destination_cidr_block": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"vpn_connection_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsVpnConnectionRouteCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	createOpts := &ec2.CreateVpnConnectionRouteInput{
		DestinationCidrBlock: aws.String(d.Get("destination_cidr_block").(string)),
		VpnConnectionId:      aws.String(d.Get("vpn_connection_id").(string)),
	}

	// Create the route.
	log.Printf("[DEBUG] Creating VPN connection route")
	_, err := conn.CreateVpnConnectionRoute(createOpts)
	if err != nil {
		return fmt.Errorf("Error creating VPN connection route: %s", err)
	}

	// Store the ID by the only two data we have available to us.
	d.SetId(fmt.Sprintf("%s:%s", *createOpts.DestinationCidrBlock, *createOpts.VpnConnectionId))

	return resourceAwsVpnConnectionRouteRead(d, meta)
}

func resourceAwsVpnConnectionRouteRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	cidrBlock, vpnConnectionId := resourceAwsVpnConnectionRouteParseId(d.Id())

	routeFilters := []*ec2.Filter{
		&ec2.Filter{
			Name:   aws.String("route.destination-cidr-block"),
			Values: []*string{aws.String(cidrBlock)},
		},
		&ec2.Filter{
			Name:   aws.String("vpn-connection-id"),
			Values: []*string{aws.String(vpnConnectionId)},
		},
	}

	// Technically, we know everything there is to know about the route
	// from its ID, but we still want to catch cases where it changes
	// outside of terraform and results in a stale state file. Hence,
	// conduct a read.
	resp, err := conn.DescribeVpnConnections(&ec2.DescribeVpnConnectionsInput{
		Filters: routeFilters,
	})
	if err != nil {
		if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "InvalidVpnConnectionID.NotFound" {
			d.SetId("")
			return nil
		} else {
			log.Printf("[ERROR] Error finding VPN connection route: %s", err)
			return err
		}
	}
	if resp == nil || len(resp.VpnConnections) == 0 {
		// This is kind of a weird edge case. I'd rather return an error
		// instead of just blindly setting the ID to ""... since I don't know
		// what might cause this.
		return fmt.Errorf("No VPN connections returned")
	}

	vpnConnection := resp.VpnConnections[0]

	var found bool
	for _, r := range vpnConnection.Routes {
		if *r.DestinationCidrBlock == cidrBlock {
			d.Set("destination_cidr_block", *r.DestinationCidrBlock)
			d.Set("vpn_connection_id", *vpnConnection.VpnConnectionId)
			found = true
		}
	}
	if !found {
		// Something other than terraform eliminated the route.
		d.SetId("")
	}

	return nil
}

func resourceAwsVpnConnectionRouteDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	_, err := conn.DeleteVpnConnectionRoute(&ec2.DeleteVpnConnectionRouteInput{
		DestinationCidrBlock: aws.String(d.Get("destination_cidr_block").(string)),
		VpnConnectionId:      aws.String(d.Get("vpn_connection_id").(string)),
	})
	if err != nil {
		if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "InvalidVpnConnectionID.NotFound" {
			d.SetId("")
			return nil
		} else {
			log.Printf("[ERROR] Error deleting VPN connection route: %s", err)
			return err
		}
	}

	return nil
}

func resourceAwsVpnConnectionRouteParseId(id string) (string, string) {
	parts := strings.SplitN(id, ":", 2)
	return parts[0], parts[1]
}
