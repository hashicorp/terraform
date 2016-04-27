package azure

import (
	"fmt"
	"testing"

	"github.com/Azure/azure-sdk-for-go/management"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureDnsServerBasic(t *testing.T) {
	name := "azure_dns_server.foo"

	random := acctest.RandInt()
	config := testAccAzureDnsServerBasic(random)
	serverName := fmt.Sprintf("tf-dns-server-%d", random)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAzureDnsServerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureDnsServerExists(name),
					resource.TestCheckResourceAttr(name, "name", serverName),
					resource.TestCheckResourceAttr(name, "dns_address", "8.8.8.8"),
				),
			},
		},
	})
}

func TestAccAzureDnsServerUpdate(t *testing.T) {
	name := "azure_dns_server.foo"

	random := acctest.RandInt()
	basicConfig := testAccAzureDnsServerBasic(random)
	updateConfig := testAccAzureDnsServerUpdate(random)
	serverName := fmt.Sprintf("tf-dns-server-%d", random)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAzureDnsServerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: basicConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureDnsServerExists(name),
					resource.TestCheckResourceAttr(name, "name", serverName),
					resource.TestCheckResourceAttr(name, "dns_address", "8.8.8.8"),
				),
			},

			resource.TestStep{
				Config: updateConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureDnsServerExists(name),
					resource.TestCheckResourceAttr(name, "name", serverName),
					resource.TestCheckResourceAttr(name, "dns_address", "8.8.4.4"),
				),
			},
		},
	})
}

func testAccCheckAzureDnsServerExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resource, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Resource not found: %s", name)
		}

		if resource.Primary.ID == "" {
			return fmt.Errorf("No DNS Server ID set.")
		}

		vnetClient := testAccProvider.Meta().(*Client).vnetClient
		netConf, err := vnetClient.GetVirtualNetworkConfiguration()
		if err != nil {
			return fmt.Errorf("Failed fetching networking configuration: %s", err)
		}

		for _, dns := range netConf.Configuration.DNS.DNSServers {
			if dns.Name == resource.Primary.ID {
				return nil
			}
		}

		return fmt.Errorf("Azure DNS Server not found.")
	}
}

func testAccCheckAzureDnsServerDestroy(s *terraform.State) error {
	vnetClient := testAccProvider.Meta().(*Client).vnetClient

	for _, resource := range s.RootModule().Resources {
		if resource.Type != "azure_dns_server" {
			continue
		}

		if resource.Primary.ID == "" {
			return fmt.Errorf("No DNS Server ID is set.")
		}

		netConf, err := vnetClient.GetVirtualNetworkConfiguration()
		if err != nil {
			// This is desirable - if there is no network config there can't be any DNS Servers
			if management.IsResourceNotFoundError(err) {
				continue
			}
			return fmt.Errorf("Error retrieving networking configuration from Azure: %s", err)
		}

		for _, dns := range netConf.Configuration.DNS.DNSServers {
			if dns.Name == resource.Primary.ID {
				return fmt.Errorf("Azure DNS Server still exists.")
			}
		}
	}

	return nil
}

func testAccAzureDnsServerBasic(random int) string {
	return fmt.Sprintf(`
resource "azure_dns_server" "foo" {
    name = "tf-dns-server-%d"
    dns_address = "8.8.8.8"
}
`, random)
}

func testAccAzureDnsServerUpdate(random int) string {
	return fmt.Sprintf(`
resource "azure_dns_server" "foo" {
    name = "tf-dns-server-%d"
    dns_address = "8.8.4.4"
}
`, random)
}
