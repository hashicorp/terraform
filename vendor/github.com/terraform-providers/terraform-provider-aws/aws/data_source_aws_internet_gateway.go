package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsInternetGateway() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsInternetGatewayRead,
		Schema: map[string]*schema.Schema{
			"internet_gateway_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"filter": ec2CustomFiltersSchema(),
			"tags":   tagsSchemaComputed(),
			"attachments": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"state": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"vpc_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceAwsInternetGatewayRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	req := &ec2.DescribeInternetGatewaysInput{}
	internetGatewayId, internetGatewayIdOk := d.GetOk("internet_gateway_id")
	tags, tagsOk := d.GetOk("tags")
	filter, filterOk := d.GetOk("filter")

	if !internetGatewayIdOk && !filterOk && !tagsOk {
		return fmt.Errorf("One of internet_gateway_id or filter or tags must be assigned")
	}

	req.Filters = buildEC2AttributeFilterList(map[string]string{
		"internet-gateway-id": internetGatewayId.(string),
	})
	req.Filters = append(req.Filters, buildEC2TagFilterList(
		tagsFromMap(tags.(map[string]interface{})),
	)...)
	req.Filters = append(req.Filters, buildEC2CustomFilterList(
		filter.(*schema.Set),
	)...)

	log.Printf("[DEBUG] Reading Internet Gateway: %s", req)
	resp, err := conn.DescribeInternetGateways(req)

	if err != nil {
		return err
	}
	if resp == nil || len(resp.InternetGateways) == 0 {
		return fmt.Errorf("Your query returned no results. Please change your search criteria and try again.")
	}
	if len(resp.InternetGateways) > 1 {
		return fmt.Errorf("Multiple Internet Gateways matched; use additional constraints to reduce matches to a single Internet Gateway")
	}

	igw := resp.InternetGateways[0]
	d.SetId(aws.StringValue(igw.InternetGatewayId))
	d.Set("tags", tagsToMap(igw.Tags))
	d.Set("internet_gateway_id", igw.InternetGatewayId)
	if err := d.Set("attachments", dataSourceAttachmentsRead(igw.Attachments)); err != nil {
		return err
	}

	return nil
}

func dataSourceAttachmentsRead(igwAttachments []*ec2.InternetGatewayAttachment) []map[string]interface{} {
	attachments := make([]map[string]interface{}, 0, len(igwAttachments))
	for _, a := range igwAttachments {
		m := make(map[string]interface{})
		m["state"] = *a.State
		m["vpc_id"] = *a.VpcId
		attachments = append(attachments, m)
	}

	return attachments
}
