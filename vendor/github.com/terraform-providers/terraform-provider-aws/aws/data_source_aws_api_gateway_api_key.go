package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsApiGatewayApiKey() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsApiGatewayApiKeyRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"value": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
		},
	}
}

func dataSourceAwsApiGatewayApiKeyRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway
	apiKey, err := conn.GetApiKey(&apigateway.GetApiKeyInput{
		ApiKey:       aws.String(d.Get("id").(string)),
		IncludeValue: aws.Bool(true),
	})

	if err != nil {
		return err
	}

	d.SetId(aws.StringValue(apiKey.Id))
	d.Set("name", apiKey.Name)
	d.Set("value", apiKey.Value)
	return nil
}
