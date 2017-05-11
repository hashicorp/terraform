package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsCloudFormationExports() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsCloudFormationExportsRead,

		Schema: map[string]*schema.Schema{
			"values": {
				Type:     schema.TypeMap,
				Computed: true,
			},
			"stack_ids": {
				Type:     schema.TypeMap,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsCloudFormationExportsRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cfconn
	d.SetId(fmt.Sprintf("cloudformation-exports-%s", meta.(*AWSClient).region))
	input := &cloudformation.ListExportsInput{}
	values := make(map[string]string)
	stack_ids := make(map[string]string)
	err := conn.ListExportsPages(input,
		func(page *cloudformation.ListExportsOutput, lastPage bool) bool {
			flattenCloudformationExports(values, stack_ids, page.Exports)
			if page.NextToken != nil {
				return true
			} else {
				return false
			}
		})
	if err != nil {
		return fmt.Errorf("Failed listing CloudFormation exports: %s", err)
	}
	d.Set("values", values)
	d.Set("stack_ids", stack_ids)
	return nil
}
