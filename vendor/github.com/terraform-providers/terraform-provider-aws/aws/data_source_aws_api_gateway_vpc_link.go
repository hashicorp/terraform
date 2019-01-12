package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsApiGatewayVpcLink() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsApiGatewayVpcLinkRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func dataSourceAwsApiGatewayVpcLinkRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway
	params := &apigateway.GetVpcLinksInput{}

	target := d.Get("name")
	var matchedVpcLinks []*apigateway.UpdateVpcLinkOutput
	log.Printf("[DEBUG] Reading API Gateway VPC links: %s", params)
	err := conn.GetVpcLinksPages(params, func(page *apigateway.GetVpcLinksOutput, lastPage bool) bool {
		for _, api := range page.Items {
			if aws.StringValue(api.Name) == target {
				matchedVpcLinks = append(matchedVpcLinks, api)
			}
		}
		return !lastPage
	})
	if err != nil {
		return fmt.Errorf("error describing API Gateway VPC links: %s", err)
	}

	if len(matchedVpcLinks) == 0 {
		return fmt.Errorf("no API Gateway VPC link with name %q found in this region", target)
	}
	if len(matchedVpcLinks) > 1 {
		return fmt.Errorf("multiple API Gateway VPC links with name %q found in this region", target)
	}

	match := matchedVpcLinks[0]

	d.SetId(*match.Id)
	d.Set("name", match.Name)

	return nil
}
