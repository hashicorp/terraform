package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSAPIGatewayAccount_importBasic(t *testing.T) {
	resourceName := "aws_api_gateway_account.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAPIGatewayAccountDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAPIGatewayAccountConfig_empty,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
