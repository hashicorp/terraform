package azure

import (
	"fmt"
	"testing"

	"github.com/Azure/azure-sdk-for-go/management"
	"github.com/Azure/azure-sdk-for-go/management/networksecuritygroup"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureSecurityGroup_basic(t *testing.T) {
	var group networksecuritygroup.SecurityGroupResponse

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAzureSecurityGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureSecurityGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureSecurityGroupExists(
						"azure_security_group.foo", &group),
					resource.TestCheckResourceAttr(
						"azure_security_group.foo", "name", "terraform-security-group"),
					resource.TestCheckResourceAttr(
						"azure_security_group.foo", "location", "West US"),
					resource.TestCheckResourceAttr(
						"azure_security_group.foo", "label", "terraform testing security group"),
				),
			},
		},
	})
}

func testAccCheckAzureSecurityGroupExists(
	n string,
	group *networksecuritygroup.SecurityGroupResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Network Security Group ID is set")
		}

		secGroupClient := testAccProvider.Meta().(*Client).secGroupClient
		sg, err := secGroupClient.GetNetworkSecurityGroup(rs.Primary.ID)
		if err != nil {
			return err
		}

		if sg.Name != rs.Primary.ID {
			return fmt.Errorf("Security Group not found")
		}

		*group = sg

		return nil
	}
}

func testAccCheckAzureSecurityGroupDestroy(s *terraform.State) error {
	secGroupClient := testAccProvider.Meta().(*Client).secGroupClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azure_security_group" {
			continue
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Network Security Group ID is set")
		}

		_, err := secGroupClient.GetNetworkSecurityGroup(rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("Network Security Group %s still exists", rs.Primary.ID)
		}

		if !management.IsResourceNotFoundError(err) {
			return err
		}
	}

	return nil
}

const testAccAzureSecurityGroupConfigTemplate = `
resource "azure_security_group" "%s" {
    name = "%s"
    location = "West US"
    label = "terraform testing security group"
}`

var testAccAzureSecurityGroupConfig = fmt.Sprintf(
	testAccAzureSecurityGroupConfigTemplate,
	"foo", "terraform-security-group",
)
