package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsCloudFormationExport() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsCloudFormationExportRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"value": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"exporting_stack_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsCloudFormationExportRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cfconn
	var value string
	name := d.Get("name").(string)
	region := meta.(*AWSClient).region
	d.SetId(fmt.Sprintf("cloudformation-exports-%s-%s", region, name))
	input := &cloudformation.ListExportsInput{}
	err := conn.ListExportsPages(input,
		func(page *cloudformation.ListExportsOutput, lastPage bool) bool {
			for _, e := range page.Exports {
				if name == aws.StringValue(e.Name) {
					value = aws.StringValue(e.Value)
					d.Set("value", e.Value)
					d.Set("exporting_stack_id", e.ExportingStackId)
					return false
				}
			}
			return !lastPage
		})
	if err != nil {
		return fmt.Errorf("Failed listing CloudFormation exports: %s", err)
	}
	if "" == value {
		return fmt.Errorf("%s was not found in CloudFormation Exports for region %s", name, region)
	}
	return nil
}
