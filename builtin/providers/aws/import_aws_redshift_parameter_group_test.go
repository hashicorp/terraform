package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSRedshiftParameterGroup_importBasic(t *testing.T) {
	resourceName := "aws_redshift_parameter_group.bar"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRedshiftParameterGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSRedshiftParameterGroupConfig,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
