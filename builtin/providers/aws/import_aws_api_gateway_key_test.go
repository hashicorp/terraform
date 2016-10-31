package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSAPIGatewayApiKey_importBasic(t *testing.T) {
	resourceName := "aws_api_gateway_api_key.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAPIGatewayApiKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSAPIGatewayApiKeyConfig,
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
