package azure

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/svanharmelen/azure-sdk-for-go/management"
	"github.com/svanharmelen/azure-sdk-for-go/management/networksecuritygroup"
)

func TestAccAzureSecurityGroup_basic(t *testing.T) {
	var group networksecuritygroup.SecurityGroupResponse

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAzureSecurityGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureSecurityGroup_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureSecurityGroupExists(
						"azure_security_group.foo", &group),
					testAccCheckAzureSecurityGroupBasicAttributes(&group),
					resource.TestCheckResourceAttr(
						"azure_security_group.foo", "name", "terraform-security-group"),
					resource.TestCheckResourceAttr(
						"azure_security_group.foo", "location", "West US"),
					resource.TestCheckResourceAttr(
						"azure_security_group.foo", "rule.936204579.name", "RDP"),
					resource.TestCheckResourceAttr(
						"azure_security_group.foo", "rule.936204579.source_port", "*"),
					resource.TestCheckResourceAttr(
						"azure_security_group.foo", "rule.936204579.destination_port", "3389"),
				),
			},
		},
	})
}

func TestAccAzureSecurityGroup_update(t *testing.T) {
	var group networksecuritygroup.SecurityGroupResponse

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAzureSecurityGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureSecurityGroup_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureSecurityGroupExists(
						"azure_security_group.foo", &group),
					testAccCheckAzureSecurityGroupBasicAttributes(&group),
					resource.TestCheckResourceAttr(
						"azure_security_group.foo", "name", "terraform-security-group"),
					resource.TestCheckResourceAttr(
						"azure_security_group.foo", "location", "West US"),
					resource.TestCheckResourceAttr(
						"azure_security_group.foo", "rule.936204579.name", "RDP"),
					resource.TestCheckResourceAttr(
						"azure_security_group.foo", "rule.936204579.source_cidr", "*"),
					resource.TestCheckResourceAttr(
						"azure_security_group.foo", "rule.936204579.destination_port", "3389"),
				),
			},

			resource.TestStep{
				Config: testAccAzureSecurityGroup_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureSecurityGroupExists(
						"azure_security_group.foo", &group),
					testAccCheckAzureSecurityGroupUpdatedAttributes(&group),
					resource.TestCheckResourceAttr(
						"azure_security_group.foo", "rule.3322523298.name", "RDP"),
					resource.TestCheckResourceAttr(
						"azure_security_group.foo", "rule.3322523298.source_cidr", "192.168.0.0/24"),
					resource.TestCheckResourceAttr(
						"azure_security_group.foo", "rule.3322523298.destination_port", "3389"),
					resource.TestCheckResourceAttr(
						"azure_security_group.foo", "rule.3929353075.name", "WINRM"),
					resource.TestCheckResourceAttr(
						"azure_security_group.foo", "rule.3929353075.source_cidr", "192.168.0.0/24"),
					resource.TestCheckResourceAttr(
						"azure_security_group.foo", "rule.3929353075.destination_port", "5985"),
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

		mc := testAccProvider.Meta().(*Client).mgmtClient
		sg, err := networksecuritygroup.NewClient(mc).GetNetworkSecurityGroup(rs.Primary.ID)
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

func testAccCheckAzureSecurityGroupBasicAttributes(
	group *networksecuritygroup.SecurityGroupResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if group.Name != "terraform-security-group" {
			return fmt.Errorf("Bad name: %s", group.Name)
		}

		for _, r := range group.Rules {
			if !r.IsDefault {
				if r.Name != "RDP" {
					return fmt.Errorf("Bad rule name: %s", r.Name)
				}
				if r.Priority != 101 {
					return fmt.Errorf("Bad rule priority: %d", r.Priority)
				}
				if r.SourceAddressPrefix != "*" {
					return fmt.Errorf("Bad source CIDR: %s", r.SourceAddressPrefix)
				}
				if r.DestinationAddressPrefix != "*" {
					return fmt.Errorf("Bad destination CIDR: %s", r.DestinationAddressPrefix)
				}
				if r.DestinationPortRange != "3389" {
					return fmt.Errorf("Bad destination port: %s", r.DestinationPortRange)
				}
			}
		}

		return nil
	}
}

func testAccCheckAzureSecurityGroupUpdatedAttributes(
	group *networksecuritygroup.SecurityGroupResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if group.Name != "terraform-security-group" {
			return fmt.Errorf("Bad name: %s", group.Name)
		}

		foundRDP := false
		foundWINRM := false
		for _, r := range group.Rules {
			if !r.IsDefault {
				if r.Name == "RDP" {
					if r.SourceAddressPrefix != "192.168.0.0/24" {
						return fmt.Errorf("Bad source CIDR: %s", r.SourceAddressPrefix)
					}

					foundRDP = true
				}

				if r.Name == "WINRM" {
					if r.Priority != 102 {
						return fmt.Errorf("Bad rule priority: %d", r.Priority)
					}
					if r.SourceAddressPrefix != "192.168.0.0/24" {
						return fmt.Errorf("Bad source CIDR: %s", r.SourceAddressPrefix)
					}
					if r.DestinationAddressPrefix != "*" {
						return fmt.Errorf("Bad destination CIDR: %s", r.DestinationAddressPrefix)
					}
					if r.DestinationPortRange != "5985" {
						return fmt.Errorf("Bad destination port: %s", r.DestinationPortRange)
					}

					foundWINRM = true
				}
			}
		}

		if !foundRDP {
			return fmt.Errorf("RDP rule not found")
		}

		if !foundWINRM {
			return fmt.Errorf("WINRM rule not found")
		}

		return nil
	}
}

func testAccCheckAzureSecurityGroupDestroy(s *terraform.State) error {
	mc := testAccProvider.Meta().(*Client).mgmtClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azure_security_group" {
			continue
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Network Security Group ID is set")
		}

		_, err := networksecuritygroup.NewClient(mc).GetNetworkSecurityGroup(rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("Resource %s still exists", rs.Primary.ID)
		}

		if !management.IsResourceNotFoundError(err) {
			return err
		}
	}

	return nil
}

const testAccAzureSecurityGroup_basic = `
resource "azure_security_group" "foo" {
    name = "terraform-security-group"
    location = "West US"

    rule {
        name = "RDP"
        priority = 101
        source_cidr = "*"
        source_port = "*"
        destination_cidr = "*"
        destination_port = "3389"
        protocol = "TCP"
    }
}`

const testAccAzureSecurityGroup_update = `
resource "azure_security_group" "foo" {
    name = "terraform-security-group"
    location = "West US"

    rule {
        name = "RDP"
        priority = 101
        source_cidr = "192.168.0.0/24"
        source_port = "*"
        destination_cidr = "*"
        destination_port = "3389"
        protocol = "TCP"
    }

    rule {
        name = "WINRM"
        priority = 102
        source_cidr = "192.168.0.0/24"
        source_port = "*"
        destination_cidr = "*"
        destination_port = "5985"
        protocol = "TCP"
    }
}`
