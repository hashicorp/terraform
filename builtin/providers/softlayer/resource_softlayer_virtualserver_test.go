package softlayer

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	datatypes "github.com/maximilien/softlayer-go/data_types"
)

func TestAccSoftLayerVirtualserver_Basic(t *testing.T) {
	var server datatypes.SoftLayer_Virtual_Guest

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: testAccCheckSoftLayerVirtualserverDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckSoftLayerVirtualserverConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSoftLayerVirtualserverExists("softlayer_virtualserver.terraform-acceptance-test-1", &server),
					resource.TestCheckResourceAttr(
						"softlayer_virtualserver.terraform-acceptance-test-1", "name", "terraform-test"),
					resource.TestCheckResourceAttr(
						"softlayer_virtualserver.terraform-acceptance-test-1", "domain", "bar.example.com"),
					resource.TestCheckResourceAttr(
						"softlayer_virtualserver.terraform-acceptance-test-1", "region", "ams01"),
					resource.TestCheckResourceAttr(
						"softlayer_virtualserver.terraform-acceptance-test-1", "public_network_speed", "10"),
					resource.TestCheckResourceAttr(
						"softlayer_virtualserver.terraform-acceptance-test-1", "cpu", "1"),
					resource.TestCheckResourceAttr(
						"softlayer_virtualserver.terraform-acceptance-test-1", "ram", "1024"),
					resource.TestCheckResourceAttr(
						"softlayer_virtualserver.terraform-acceptance-test-1", "user_data", "{\"fox\":[45]}"),
				),
			},
		},
	})
}

func testAccCheckSoftLayerVirtualserverDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Client).virtualGuestService

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "softlayer_virtualserver" {
			continue
		}

		serverId, _ := strconv.Atoi(rs.Primary.ID)

		// Try to find the server
		_, err := client.GetObject(serverId)

		// Wait

		if err != nil && !strings.Contains(err.Error(), "404") {
			return fmt.Errorf(
				"Error waiting for server (%s) to be destroyed: %s",
				rs.Primary.ID, err)
		}
	}

	return nil
}

func testAccCheckSoftLayerVirtualserverExists(n string, server *datatypes.SoftLayer_Virtual_Guest) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No virtual server ID is set")
		}

		id, err := strconv.Atoi(rs.Primary.ID)

		if err != nil {
			return err
		}

		client := testAccProvider.Meta().(*Client).virtualGuestService
		retrieveServer, err := client.GetObject(id)

		if err != nil {
			return err
		}

		fmt.Printf("The ID is %d", id)

		if retrieveServer.Id != id {
			return fmt.Errorf("Virtual server not found")
		}

		*server = retrieveServer

		return nil
	}
}

const testAccCheckSoftLayerVirtualserverConfig_basic = `
resource "softlayer_virtualserver" "terraform-acceptance-test-1" {
    name = "terraform-test"
    domain = "bar.example.com"
    image = "DEBIAN_7_64"
    region = "ams01"
    public_network_speed = 10
    cpu = 1
    ram = 1024
    user_data = "{\"fox\":[45]}"
}
`
