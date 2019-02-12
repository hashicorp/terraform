package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsEc2TransitGatewayRoute() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEc2TransitGatewayRouteCreate,
		Read:   resourceAwsEc2TransitGatewayRouteRead,
		Delete: resourceAwsEc2TransitGatewayRouteDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"destination_cidr_block": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"transit_gateway_attachment_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.NoZeroValues,
			},
			"transit_gateway_route_table_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.NoZeroValues,
			},
		},
	}
}

func resourceAwsEc2TransitGatewayRouteCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	destination := d.Get("destination_cidr_block").(string)
	transitGatewayRouteTableID := d.Get("transit_gateway_route_table_id").(string)

	input := &ec2.CreateTransitGatewayRouteInput{
		DestinationCidrBlock:       aws.String(destination),
		TransitGatewayAttachmentId: aws.String(d.Get("transit_gateway_attachment_id").(string)),
		TransitGatewayRouteTableId: aws.String(transitGatewayRouteTableID),
	}

	log.Printf("[DEBUG] Creating EC2 Transit Gateway Route: %s", input)
	_, err := conn.CreateTransitGatewayRoute(input)
	if err != nil {
		return fmt.Errorf("error creating EC2 Transit Gateway Route: %s", err)
	}

	d.SetId(fmt.Sprintf("%s_%s", transitGatewayRouteTableID, destination))

	return resourceAwsEc2TransitGatewayRouteRead(d, meta)
}

func resourceAwsEc2TransitGatewayRouteRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	transitGatewayRouteTableID, destination, err := decodeEc2TransitGatewayRouteID(d.Id())
	if err != nil {
		return err
	}

	// Handle EC2 eventual consistency
	var transitGatewayRoute *ec2.TransitGatewayRoute
	err = resource.Retry(1*time.Minute, func() *resource.RetryError {
		var err error
		transitGatewayRoute, err = ec2DescribeTransitGatewayRoute(conn, transitGatewayRouteTableID, destination)

		if err != nil {
			return resource.NonRetryableError(err)
		}

		if d.IsNewResource() && transitGatewayRoute == nil {
			return resource.RetryableError(&resource.NotFoundError{})
		}

		return nil
	})

	if isAWSErr(err, "InvalidRouteTableID.NotFound", "") {
		log.Printf("[WARN] EC2 Transit Gateway Route Table (%s) not found, removing from state", transitGatewayRouteTableID)
		d.SetId("")
		return nil
	}

	if isResourceNotFoundError(err) {
		log.Printf("[WARN] EC2 Transit Gateway Route (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error reading EC2 Transit Gateway Route: %s", err)
	}

	if transitGatewayRoute == nil {
		log.Printf("[WARN] EC2 Transit Gateway Route (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	state := aws.StringValue(transitGatewayRoute.State)
	if state == ec2.TransitGatewayRouteStateDeleted || state == ec2.TransitGatewayRouteStateDeleting {
		log.Printf("[WARN] EC2 Transit Gateway Route (%s) deleted, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("destination_cidr_block", transitGatewayRoute.DestinationCidrBlock)

	d.Set("transit_gateway_attachment_id", "")
	if len(transitGatewayRoute.TransitGatewayAttachments) > 0 && transitGatewayRoute.TransitGatewayAttachments[0] != nil {
		d.Set("transit_gateway_attachment_id", transitGatewayRoute.TransitGatewayAttachments[0].TransitGatewayAttachmentId)
	}

	d.Set("transit_gateway_route_table_id", transitGatewayRouteTableID)

	return nil
}

func resourceAwsEc2TransitGatewayRouteDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	transitGatewayRouteTableID, destination, err := decodeEc2TransitGatewayRouteID(d.Id())
	if err != nil {
		return err
	}

	input := &ec2.DeleteTransitGatewayRouteInput{
		DestinationCidrBlock:       aws.String(destination),
		TransitGatewayRouteTableId: aws.String(transitGatewayRouteTableID),
	}

	log.Printf("[DEBUG] Deleting EC2 Transit Gateway Route (%s): %s", d.Id(), input)
	_, err = conn.DeleteTransitGatewayRoute(input)

	if isAWSErr(err, "InvalidRoute.NotFound", "") || isAWSErr(err, "InvalidRouteTableID.NotFound", "") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error deleting EC2 Transit Gateway Route: %s", err)
	}

	return nil
}
