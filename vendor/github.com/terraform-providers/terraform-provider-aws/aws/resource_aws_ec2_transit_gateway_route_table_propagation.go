package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsEc2TransitGatewayRouteTablePropagation() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEc2TransitGatewayRouteTablePropagationCreate,
		Read:   resourceAwsEc2TransitGatewayRouteTablePropagationRead,
		Delete: resourceAwsEc2TransitGatewayRouteTablePropagationDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"resource_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"resource_type": {
				Type:     schema.TypeString,
				Computed: true,
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

func resourceAwsEc2TransitGatewayRouteTablePropagationCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	transitGatewayAttachmentID := d.Get("transit_gateway_attachment_id").(string)
	transitGatewayRouteTableID := d.Get("transit_gateway_route_table_id").(string)

	input := &ec2.EnableTransitGatewayRouteTablePropagationInput{
		TransitGatewayAttachmentId: aws.String(transitGatewayAttachmentID),
		TransitGatewayRouteTableId: aws.String(transitGatewayRouteTableID),
	}

	_, err := conn.EnableTransitGatewayRouteTablePropagation(input)
	if err != nil {
		return fmt.Errorf("error enabling EC2 Transit Gateway Route Table (%s) propagation (%s): %s", transitGatewayRouteTableID, transitGatewayAttachmentID, err)
	}

	d.SetId(fmt.Sprintf("%s_%s", transitGatewayRouteTableID, transitGatewayAttachmentID))

	return resourceAwsEc2TransitGatewayRouteTablePropagationRead(d, meta)
}

func resourceAwsEc2TransitGatewayRouteTablePropagationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	transitGatewayRouteTableID, transitGatewayAttachmentID, err := decodeEc2TransitGatewayRouteTablePropagationID(d.Id())
	if err != nil {
		return err
	}

	transitGatewayPropagation, err := ec2DescribeTransitGatewayRouteTablePropagation(conn, transitGatewayRouteTableID, transitGatewayAttachmentID)

	if isAWSErr(err, "InvalidRouteTableID.NotFound", "") {
		log.Printf("[WARN] EC2 Transit Gateway Route Table (%s) not found, removing from state", transitGatewayRouteTableID)
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error reading EC2 Transit Gateway Route Table (%s) Propagation (%s): %s", transitGatewayRouteTableID, transitGatewayAttachmentID, err)
	}

	if transitGatewayPropagation == nil {
		log.Printf("[WARN] EC2 Transit Gateway Route Table (%s) Propagation (%s) not found, removing from state", transitGatewayRouteTableID, transitGatewayAttachmentID)
		d.SetId("")
		return nil
	}

	d.Set("resource_id", aws.StringValue(transitGatewayPropagation.ResourceId))
	d.Set("resource_type", aws.StringValue(transitGatewayPropagation.ResourceType))
	d.Set("transit_gateway_attachment_id", aws.StringValue(transitGatewayPropagation.TransitGatewayAttachmentId))
	d.Set("transit_gateway_route_table_id", transitGatewayRouteTableID)

	return nil
}

func resourceAwsEc2TransitGatewayRouteTablePropagationDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	transitGatewayRouteTableID, transitGatewayAttachmentID, err := decodeEc2TransitGatewayRouteTablePropagationID(d.Id())
	if err != nil {
		return err
	}

	input := &ec2.DisableTransitGatewayRouteTablePropagationInput{
		TransitGatewayAttachmentId: aws.String(transitGatewayAttachmentID),
		TransitGatewayRouteTableId: aws.String(transitGatewayRouteTableID),
	}

	log.Printf("[DEBUG] Disabling EC2 Transit Gateway Route Table (%s) Propagation (%s): %s", transitGatewayRouteTableID, transitGatewayAttachmentID, input)
	_, err = conn.DisableTransitGatewayRouteTablePropagation(input)

	if isAWSErr(err, "InvalidRouteTableID.NotFound", "") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error disabling EC2 Transit Gateway Route Table (%s) Propagation (%s): %s", transitGatewayRouteTableID, transitGatewayAttachmentID, err)
	}

	return nil
}
