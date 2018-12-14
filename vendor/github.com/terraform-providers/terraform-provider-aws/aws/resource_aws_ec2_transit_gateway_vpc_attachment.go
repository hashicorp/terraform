package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsEc2TransitGatewayVpcAttachment() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEc2TransitGatewayVpcAttachmentCreate,
		Read:   resourceAwsEc2TransitGatewayVpcAttachmentRead,
		Update: resourceAwsEc2TransitGatewayVpcAttachmentUpdate,
		Delete: resourceAwsEc2TransitGatewayVpcAttachmentDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"dns_support": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  ec2.DnsSupportValueEnable,
				ValidateFunc: validation.StringInSlice([]string{
					ec2.DnsSupportValueDisable,
					ec2.DnsSupportValueEnable,
				}, false),
			},
			"ipv6_support": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  ec2.Ipv6SupportValueDisable,
				ValidateFunc: validation.StringInSlice([]string{
					ec2.Ipv6SupportValueDisable,
					ec2.Ipv6SupportValueEnable,
				}, false),
			},
			"subnet_ids": {
				Type:     schema.TypeSet,
				Required: true,
				MinItems: 1,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"tags": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"transit_gateway_default_route_table_association": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"transit_gateway_default_route_table_propagation": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"transit_gateway_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.NoZeroValues,
			},
			"vpc_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.NoZeroValues,
			},
			"vpc_owner_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsEc2TransitGatewayVpcAttachmentCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	transitGatewayID := d.Get("transit_gateway_id").(string)

	input := &ec2.CreateTransitGatewayVpcAttachmentInput{
		Options: &ec2.CreateTransitGatewayVpcAttachmentRequestOptions{
			DnsSupport:  aws.String(d.Get("dns_support").(string)),
			Ipv6Support: aws.String(d.Get("ipv6_support").(string)),
		},
		SubnetIds:         expandStringSet(d.Get("subnet_ids").(*schema.Set)),
		TransitGatewayId:  aws.String(transitGatewayID),
		TagSpecifications: expandEc2TransitGatewayAttachmentTagSpecifications(d.Get("tags").(map[string]interface{})),
		VpcId:             aws.String(d.Get("vpc_id").(string)),
	}

	log.Printf("[DEBUG] Creating EC2 Transit Gateway VPC Attachment: %s", input)
	output, err := conn.CreateTransitGatewayVpcAttachment(input)
	if err != nil {
		return fmt.Errorf("error creating EC2 Transit Gateway VPC Attachment: %s", err)
	}

	d.SetId(aws.StringValue(output.TransitGatewayVpcAttachment.TransitGatewayAttachmentId))

	if err := waitForEc2TransitGatewayRouteTableAttachmentCreation(conn, d.Id()); err != nil {
		return fmt.Errorf("error waiting for EC2 Transit Gateway VPC Attachment (%s) availability: %s", d.Id(), err)
	}

	transitGateway, err := ec2DescribeTransitGateway(conn, transitGatewayID)
	if err != nil {
		return fmt.Errorf("error describing EC2 Transit Gateway (%s): %s", transitGatewayID, err)
	}

	if transitGateway.Options == nil {
		return fmt.Errorf("error describing EC2 Transit Gateway (%s): missing options", transitGatewayID)
	}

	if err := ec2TransitGatewayRouteTableAssociationUpdate(conn, aws.StringValue(transitGateway.Options.AssociationDefaultRouteTableId), d.Id(), d.Get("transit_gateway_default_route_table_association").(bool)); err != nil {
		return fmt.Errorf("error updating EC2 Transit Gateway Attachment (%s) Route Table (%s) association: %s", d.Id(), aws.StringValue(transitGateway.Options.AssociationDefaultRouteTableId), err)
	}

	if err := ec2TransitGatewayRouteTablePropagationUpdate(conn, aws.StringValue(transitGateway.Options.PropagationDefaultRouteTableId), d.Id(), d.Get("transit_gateway_default_route_table_propagation").(bool)); err != nil {
		return fmt.Errorf("error updating EC2 Transit Gateway Attachment (%s) Route Table (%s) propagation: %s", d.Id(), aws.StringValue(transitGateway.Options.PropagationDefaultRouteTableId), err)
	}

	return resourceAwsEc2TransitGatewayVpcAttachmentRead(d, meta)
}

func resourceAwsEc2TransitGatewayVpcAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	transitGatewayVpcAttachment, err := ec2DescribeTransitGatewayVpcAttachment(conn, d.Id())

	if isAWSErr(err, "InvalidTransitGatewayAttachmentID.NotFound", "") {
		log.Printf("[WARN] EC2 Transit Gateway VPC Attachment (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error reading EC2 Transit Gateway VPC Attachment: %s", err)
	}

	if transitGatewayVpcAttachment == nil {
		log.Printf("[WARN] EC2 Transit Gateway VPC Attachment (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if aws.StringValue(transitGatewayVpcAttachment.State) == ec2.TransitGatewayAttachmentStateDeleting || aws.StringValue(transitGatewayVpcAttachment.State) == ec2.TransitGatewayAttachmentStateDeleted {
		log.Printf("[WARN] EC2 Transit Gateway VPC Attachment (%s) in deleted state (%s), removing from state", d.Id(), aws.StringValue(transitGatewayVpcAttachment.State))
		d.SetId("")
		return nil
	}

	transitGatewayID := aws.StringValue(transitGatewayVpcAttachment.TransitGatewayId)
	transitGateway, err := ec2DescribeTransitGateway(conn, transitGatewayID)
	if err != nil {
		return fmt.Errorf("error describing EC2 Transit Gateway (%s): %s", transitGatewayID, err)
	}

	if transitGateway.Options == nil {
		return fmt.Errorf("error describing EC2 Transit Gateway (%s): missing options", transitGatewayID)
	}

	transitGatewayAssociationDefaultRouteTableID := aws.StringValue(transitGateway.Options.AssociationDefaultRouteTableId)
	transitGatewayDefaultRouteTableAssociation, err := ec2DescribeTransitGatewayRouteTableAssociation(conn, transitGatewayAssociationDefaultRouteTableID, d.Id())
	if err != nil {
		return fmt.Errorf("error determining EC2 Transit Gateway Attachment (%s) association to Route Table (%s): %s", d.Id(), transitGatewayAssociationDefaultRouteTableID, err)
	}

	transitGatewayPropagationDefaultRouteTableID := aws.StringValue(transitGateway.Options.PropagationDefaultRouteTableId)
	transitGatewayDefaultRouteTablePropagation, err := ec2DescribeTransitGatewayRouteTablePropagation(conn, transitGatewayPropagationDefaultRouteTableID, d.Id())
	if err != nil {
		return fmt.Errorf("error determining EC2 Transit Gateway Attachment (%s) propagation to Route Table (%s): %s", d.Id(), transitGatewayPropagationDefaultRouteTableID, err)
	}

	if transitGatewayVpcAttachment.Options == nil {
		return fmt.Errorf("error reading EC2 Transit Gateway VPC Attachment (%s): missing options", d.Id())
	}

	d.Set("dns_support", transitGatewayVpcAttachment.Options.DnsSupport)
	d.Set("ipv6_support", transitGatewayVpcAttachment.Options.Ipv6Support)

	if err := d.Set("subnet_ids", aws.StringValueSlice(transitGatewayVpcAttachment.SubnetIds)); err != nil {
		return fmt.Errorf("error setting subnet_ids: %s", err)
	}

	if err := d.Set("tags", tagsToMap(transitGatewayVpcAttachment.Tags)); err != nil {
		return fmt.Errorf("error setting tags: %s", err)
	}

	d.Set("transit_gateway_default_route_table_association", (transitGatewayDefaultRouteTableAssociation != nil))
	d.Set("transit_gateway_default_route_table_propagation", (transitGatewayDefaultRouteTablePropagation != nil))
	d.Set("transit_gateway_id", aws.StringValue(transitGatewayVpcAttachment.TransitGatewayId))
	d.Set("vpc_id", aws.StringValue(transitGatewayVpcAttachment.VpcId))
	d.Set("vpc_owner_id", aws.StringValue(transitGatewayVpcAttachment.VpcOwnerId))

	return nil
}

func resourceAwsEc2TransitGatewayVpcAttachmentUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	if d.HasChange("dns_support") || d.HasChange("ipv6_support") || d.HasChange("subnet_ids") {
		input := &ec2.ModifyTransitGatewayVpcAttachmentInput{
			Options: &ec2.ModifyTransitGatewayVpcAttachmentRequestOptions{
				DnsSupport:  aws.String(d.Get("dns_support").(string)),
				Ipv6Support: aws.String(d.Get("ipv6_support").(string)),
			},
			TransitGatewayAttachmentId: aws.String(d.Id()),
		}

		oldRaw, newRaw := d.GetChange("subnet_ids")
		oldSet := oldRaw.(*schema.Set)
		newSet := newRaw.(*schema.Set)

		if added := newSet.Difference(oldSet); added.Len() > 0 {
			input.AddSubnetIds = expandStringSet(added)
		}

		if removed := oldSet.Difference(newSet); removed.Len() > 0 {
			input.RemoveSubnetIds = expandStringSet(removed)
		}

		if _, err := conn.ModifyTransitGatewayVpcAttachment(input); err != nil {
			return fmt.Errorf("error modifying EC2 Transit Gateway VPC Attachment (%s): %s", d.Id(), err)
		}

		if err := waitForEc2TransitGatewayRouteTableAttachmentUpdate(conn, d.Id()); err != nil {
			return fmt.Errorf("error waiting for EC2 Transit Gateway VPC Attachment (%s) update: %s", d.Id(), err)
		}
	}

	if d.HasChange("transit_gateway_default_route_table_association") || d.HasChange("transit_gateway_default_route_table_propagation") {
		transitGatewayID := d.Get("transit_gateway_id").(string)

		transitGateway, err := ec2DescribeTransitGateway(conn, transitGatewayID)
		if err != nil {
			return fmt.Errorf("error describing EC2 Transit Gateway (%s): %s", transitGatewayID, err)
		}

		if transitGateway.Options == nil {
			return fmt.Errorf("error describing EC2 Transit Gateway (%s): missing options", transitGatewayID)
		}

		if d.HasChange("transit_gateway_default_route_table_association") {
			if err := ec2TransitGatewayRouteTableAssociationUpdate(conn, aws.StringValue(transitGateway.Options.AssociationDefaultRouteTableId), d.Id(), d.Get("transit_gateway_default_route_table_association").(bool)); err != nil {
				return fmt.Errorf("error updating EC2 Transit Gateway Attachment (%s) Route Table (%s) association: %s", d.Id(), aws.StringValue(transitGateway.Options.AssociationDefaultRouteTableId), err)
			}
		}

		if d.HasChange("transit_gateway_default_route_table_propagation") {
			if err := ec2TransitGatewayRouteTablePropagationUpdate(conn, aws.StringValue(transitGateway.Options.PropagationDefaultRouteTableId), d.Id(), d.Get("transit_gateway_default_route_table_propagation").(bool)); err != nil {
				return fmt.Errorf("error updating EC2 Transit Gateway Attachment (%s) Route Table (%s) propagation: %s", d.Id(), aws.StringValue(transitGateway.Options.PropagationDefaultRouteTableId), err)
			}
		}
	}

	if d.HasChange("tags") {
		if err := setTags(conn, d); err != nil {
			return fmt.Errorf("error updating EC2 Transit Gateway VPC Attachment (%s) tags: %s", d.Id(), err)
		}
	}

	return nil
}

func resourceAwsEc2TransitGatewayVpcAttachmentDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	input := &ec2.DeleteTransitGatewayVpcAttachmentInput{
		TransitGatewayAttachmentId: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Deleting EC2 Transit Gateway VPC Attachment (%s): %s", d.Id(), input)
	_, err := conn.DeleteTransitGatewayVpcAttachment(input)

	if isAWSErr(err, "InvalidTransitGatewayAttachmentID.NotFound", "") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error deleting EC2 Transit Gateway VPC Attachment: %s", err)
	}

	if err := waitForEc2TransitGatewayRouteTableAttachmentDeletion(conn, d.Id()); err != nil {
		return fmt.Errorf("error waiting for EC2 Transit Gateway VPC Attachment (%s) deletion: %s", d.Id(), err)
	}

	return nil
}
