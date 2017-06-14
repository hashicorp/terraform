package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsVpnGatewayRoutePropagation() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsVpnGatewayRoutePropagationEnable,
		Read:   resourceAwsVpnGatewayRoutePropagationRead,
		Delete: resourceAwsVpnGatewayRoutePropagationDisable,

		Schema: map[string]*schema.Schema{
			"vpn_gateway_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"route_table_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsVpnGatewayRoutePropagationEnable(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	gwID := d.Get("vpn_gateway_id").(string)
	rtID := d.Get("route_table_id").(string)

	log.Printf("[INFO] Enabling VGW propagation from %s to %s", gwID, rtID)
	_, err := conn.EnableVgwRoutePropagation(&ec2.EnableVgwRoutePropagationInput{
		GatewayId:    aws.String(gwID),
		RouteTableId: aws.String(rtID),
	})
	if err != nil {
		return fmt.Errorf("error enabling VGW propagation: %s", err)
	}

	d.SetId(fmt.Sprintf("%s_%s", gwID, rtID))
	return nil
}

func resourceAwsVpnGatewayRoutePropagationDisable(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	gwID := d.Get("vpn_gateway_id").(string)
	rtID := d.Get("route_table_id").(string)

	log.Printf("[INFO] Disabling VGW propagation from %s to %s", gwID, rtID)
	_, err := conn.DisableVgwRoutePropagation(&ec2.DisableVgwRoutePropagationInput{
		GatewayId:    aws.String(gwID),
		RouteTableId: aws.String(rtID),
	})
	if err != nil {
		return fmt.Errorf("error disabling VGW propagation: %s", err)
	}

	d.SetId("")
	return nil
}

func resourceAwsVpnGatewayRoutePropagationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	gwID := d.Get("vpn_gateway_id").(string)
	rtID := d.Get("route_table_id").(string)

	log.Printf("[INFO] Reading route table %s to check for VPN gateway %s", rtID, gwID)
	rtRaw, _, err := resourceAwsRouteTableStateRefreshFunc(conn, rtID)()
	if err != nil {
		return err
	}
	if rtRaw == nil {
		log.Printf("[INFO] Route table %q doesn't exist, so dropping %q route propagation from state", rtID, gwID)
		d.SetId("")
		return nil
	}

	rt := rtRaw.(*ec2.RouteTable)
	exists := false
	for _, vgw := range rt.PropagatingVgws {
		if *vgw.GatewayId == gwID {
			exists = true
		}
	}
	if !exists {
		log.Printf("[INFO] %s is no longer propagating to %s, so dropping route propagation from state", rtID, gwID)
		d.SetId("")
		return nil
	}

	return nil
}
