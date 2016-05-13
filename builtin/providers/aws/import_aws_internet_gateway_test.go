package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSInternetGateway_importBasic(t *testing.T) {
	resourceName := "aws_internet_gateway.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInternetGatewayDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInternetGatewayConfig,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
