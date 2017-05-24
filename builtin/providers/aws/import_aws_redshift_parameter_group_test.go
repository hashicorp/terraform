package aws

import (
	"testing"

	"github.com/r3labs/terraform/helper/acctest"
	"github.com/r3labs/terraform/helper/resource"
)

func TestAccAWSRedshiftParameterGroup_importBasic(t *testing.T) {
	resourceName := "aws_redshift_parameter_group.bar"
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRedshiftParameterGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSRedshiftParameterGroupConfig(rInt),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
