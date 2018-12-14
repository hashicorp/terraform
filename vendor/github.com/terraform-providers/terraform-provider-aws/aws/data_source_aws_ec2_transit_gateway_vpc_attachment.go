package aws

import (
	"errors"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsEc2TransitGatewayVpcAttachment() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsEc2TransitGatewayVpcAttachmentRead,

		Schema: map[string]*schema.Schema{
			"dns_support": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"filter": dataSourceFiltersSchema(),
			"id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"ipv6_support": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"subnet_ids": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"transit_gateway_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags": tagsSchemaComputed(),
			"vpc_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"vpc_owner_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsEc2TransitGatewayVpcAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	input := &ec2.DescribeTransitGatewayVpcAttachmentsInput{}

	if v, ok := d.GetOk("filter"); ok {
		input.Filters = buildAwsDataSourceFilters(v.(*schema.Set))
	}

	if v, ok := d.GetOk("id"); ok {
		input.TransitGatewayAttachmentIds = []*string{aws.String(v.(string))}
	}

	log.Printf("[DEBUG] Reading EC2 Transit Gateways: %s", input)
	output, err := conn.DescribeTransitGatewayVpcAttachments(input)

	if err != nil {
		return fmt.Errorf("error reading EC2 Transit Gateway Route Table: %s", err)
	}

	if output == nil || len(output.TransitGatewayVpcAttachments) == 0 {
		return errors.New("error reading EC2 Transit Gateway Route Table: no results found")
	}

	if len(output.TransitGatewayVpcAttachments) > 1 {
		return errors.New("error reading EC2 Transit Gateway Route Table: multiple results found, try adjusting search criteria")
	}

	transitGatewayVpcAttachment := output.TransitGatewayVpcAttachments[0]

	if transitGatewayVpcAttachment == nil {
		return errors.New("error reading EC2 Transit Gateway Route Table: empty result")
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

	d.Set("transit_gateway_id", aws.StringValue(transitGatewayVpcAttachment.TransitGatewayId))
	d.Set("vpc_id", aws.StringValue(transitGatewayVpcAttachment.VpcId))
	d.Set("vpc_owner_id", aws.StringValue(transitGatewayVpcAttachment.VpcOwnerId))

	d.SetId(aws.StringValue(transitGatewayVpcAttachment.TransitGatewayAttachmentId))

	return nil
}
