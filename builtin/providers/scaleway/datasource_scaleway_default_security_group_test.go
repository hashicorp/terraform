package scaleway

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccScalewayDefaultSecurityGroup_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckScalewayDefaultSecurityGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSecurityGroupMeta("data.scaleway_default_security_group.default", "name", "Default security group"),
					testAccCheckSecurityGroupMeta("data.scaleway_default_security_group.default", "description", "Auto generated security group."),
				),
			},
		},
	})
}

func testAccCheckSecurityGroupMeta(n, key, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Can't find resource: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Resource ID not set")
		}

		if rs.Primary.Attributes[key] != value {
			return fmt.Errorf("Expected %q, got %q\n", value, rs.Primary.Attributes[key])
		}

		return nil
	}
}

var testAccCheckScalewayDefaultSecurityGroupConfig = `
data "scaleway_default_security_group" "default" {}
`
