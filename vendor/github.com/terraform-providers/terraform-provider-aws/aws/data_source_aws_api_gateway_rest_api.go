package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsApiGatewayRestApi() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsApiGatewayRestApiRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"root_resource_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsApiGatewayRestApiRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway
	params := &apigateway.GetRestApisInput{}

	target := d.Get("name")
	var matchedApis []*apigateway.RestApi
	log.Printf("[DEBUG] Reading API Gateway REST APIs: %s", params)
	err := conn.GetRestApisPages(params, func(page *apigateway.GetRestApisOutput, lastPage bool) bool {
		for _, api := range page.Items {
			if aws.StringValue(api.Name) == target {
				matchedApis = append(matchedApis, api)
			}
		}
		return !lastPage
	})
	if err != nil {
		return fmt.Errorf("error describing API Gateway REST APIs: %s", err)
	}

	if len(matchedApis) == 0 {
		return fmt.Errorf("no REST APIs with name %q found in this region", target)
	}
	if len(matchedApis) > 1 {
		return fmt.Errorf("multiple REST APIs with name %q found in this region", target)
	}

	match := matchedApis[0]

	d.SetId(*match.Id)

	resp, err := conn.GetResources(&apigateway.GetResourcesInput{
		RestApiId: aws.String(d.Id()),
	})
	if err != nil {
		return err
	}

	for _, item := range resp.Items {
		if *item.Path == "/" {
			d.Set("root_resource_id", item.Id)
			break
		}
	}

	return nil
}
