package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsEc2TransitGatewayRouteTableAssociation() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEc2TransitGatewayRouteTableAssociationCreate,
		Read:   resourceAwsEc2TransitGatewayRouteTableAssociationRead,
		Delete: resourceAwsEc2TransitGatewayRouteTableAssociationDelete,
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

func resourceAwsEc2TransitGatewayRouteTableAssociationCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	transitGatewayAttachmentID := d.Get("transit_gateway_attachment_id").(string)
	transitGatewayRouteTableID := d.Get("transit_gateway_route_table_id").(string)

	input := &ec2.AssociateTransitGatewayRouteTableInput{
		TransitGatewayAttachmentId: aws.String(transitGatewayAttachmentID),
		TransitGatewayRouteTableId: aws.String(transitGatewayRouteTableID),
	}

	_, err := conn.AssociateTransitGatewayRouteTable(input)
	if err != nil {
		return fmt.Errorf("error associating EC2 Transit Gateway Route Table (%s) association (%s): %s", transitGatewayRouteTableID, transitGatewayAttachmentID, err)
	}

	d.SetId(fmt.Sprintf("%s_%s", transitGatewayRouteTableID, transitGatewayAttachmentID))

	if err := waitForEc2TransitGatewayRouteTableAssociationCreation(conn, transitGatewayRouteTableID, transitGatewayAttachmentID); err != nil {
		return fmt.Errorf("error waiting for EC2 Transit Gateway Route Table (%s) association (%s): %s", transitGatewayRouteTableID, transitGatewayAttachmentID, err)
	}

	return resourceAwsEc2TransitGatewayRouteTableAssociationRead(d, meta)
}

func resourceAwsEc2TransitGatewayRouteTableAssociationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	transitGatewayRouteTableID, transitGatewayAttachmentID, err := decodeEc2TransitGatewayRouteTableAssociationID(d.Id())
	if err != nil {
		return err
	}

	transitGatewayAssociation, err := ec2DescribeTransitGatewayRouteTableAssociation(conn, transitGatewayRouteTableID, transitGatewayAttachmentID)

	if isAWSErr(err, "InvalidRouteTableID.NotFound", "") {
		log.Printf("[WARN] EC2 Transit Gateway Route Table (%s) not found, removing from state", transitGatewayRouteTableID)
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error reading EC2 Transit Gateway Route Table (%s) Association (%s): %s", transitGatewayRouteTableID, transitGatewayAttachmentID, err)
	}

	if transitGatewayAssociation == nil {
		log.Printf("[WARN] EC2 Transit Gateway Route Table (%s) Association (%s) not found, removing from state", transitGatewayRouteTableID, transitGatewayAttachmentID)
		d.SetId("")
		return nil
	}

	if aws.StringValue(transitGatewayAssociation.State) == ec2.TransitGatewayAssociationStateDisassociating {
		log.Printf("[WARN] EC2 Transit Gateway Route Table (%s) Association (%s) in deleted state (%s), removing from state", transitGatewayRouteTableID, transitGatewayAttachmentID, aws.StringValue(transitGatewayAssociation.State))
		d.SetId("")
		return nil
	}

	d.Set("resource_id", aws.StringValue(transitGatewayAssociation.ResourceId))
	d.Set("resource_type", aws.StringValue(transitGatewayAssociation.ResourceType))
	d.Set("transit_gateway_attachment_id", aws.StringValue(transitGatewayAssociation.TransitGatewayAttachmentId))
	d.Set("transit_gateway_route_table_id", transitGatewayRouteTableID)

	return nil
}

func resourceAwsEc2TransitGatewayRouteTableAssociationDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	transitGatewayRouteTableID, transitGatewayAttachmentID, err := decodeEc2TransitGatewayRouteTableAssociationID(d.Id())
	if err != nil {
		return err
	}

	input := &ec2.DisassociateTransitGatewayRouteTableInput{
		TransitGatewayAttachmentId: aws.String(transitGatewayAttachmentID),
		TransitGatewayRouteTableId: aws.String(transitGatewayRouteTableID),
	}

	log.Printf("[DEBUG] Disassociating EC2 Transit Gateway Route Table (%s) Association (%s): %s", transitGatewayRouteTableID, transitGatewayAttachmentID, input)
	_, err = conn.DisassociateTransitGatewayRouteTable(input)

	if isAWSErr(err, "InvalidRouteTableID.NotFound", "") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error disassociating EC2 Transit Gateway Route Table (%s) Association (%s): %s", transitGatewayRouteTableID, transitGatewayAttachmentID, err)
	}

	if err := waitForEc2TransitGatewayRouteTableAssociationDeletion(conn, transitGatewayRouteTableID, transitGatewayAttachmentID); err != nil {
		return fmt.Errorf("error waiting for EC2 Transit Gateway Route Table (%s) Association (%s) disassociation: %s", transitGatewayRouteTableID, transitGatewayAttachmentID, err)
	}

	return nil
}
