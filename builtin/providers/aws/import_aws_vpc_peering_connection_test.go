package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSVPCPeeringConnection_importBasic(t *testing.T) {
	resourceName := "aws_vpc_peering_connection.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSVpcPeeringConnectionDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVpcPeeringConfig,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"auto_accept"},
			},
		},
	})
}
