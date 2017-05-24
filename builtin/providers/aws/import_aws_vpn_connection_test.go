package aws

import (
	"testing"

	"github.com/r3labs/terraform/helper/acctest"
	"github.com/r3labs/terraform/helper/resource"
)

func TestAccAWSVpnConnection_importBasic(t *testing.T) {
	resourceName := "aws_vpn_connection.foo"
	rBgpAsn := acctest.RandIntRange(64512, 65534)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccAwsVpnConnectionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAwsVpnConnectionConfig(rBgpAsn),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
