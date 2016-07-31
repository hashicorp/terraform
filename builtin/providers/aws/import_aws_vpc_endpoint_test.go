package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSVpcEndpoint_importBasic(t *testing.T) {
	resourceName := "aws_vpc_endpoint.second-private-s3"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVpcEndpointDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVpcEndpointWithRouteTableAndPolicyConfig,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
