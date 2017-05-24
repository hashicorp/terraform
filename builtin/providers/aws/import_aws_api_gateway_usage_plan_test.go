package aws

import (
	"testing"

	"github.com/r3labs/terraform/helper/acctest"
	"github.com/r3labs/terraform/helper/resource"
)

func TestAccAWSAPIGatewayUsagePlan_importBasic(t *testing.T) {
	resourceName := "aws_api_gateway_usage_plan.main"
	rName := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAPIGatewayUsagePlanDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSApiGatewayUsagePlanBasicConfig(rName),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
