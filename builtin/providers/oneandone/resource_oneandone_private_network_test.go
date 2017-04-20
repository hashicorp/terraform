package oneandone

import (
	"fmt"
	"testing"

	"github.com/1and1/oneandone-cloudserver-sdk-go"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"os"
	"time"
)

func TestAccOneandonePrivateNetwork_Basic(t *testing.T) {
	var net oneandone.PrivateNetwork

	name := "test"
	name_updated := "test1"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckOneandonePrivateNetworkDestroyCheck,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckOneandonePrivateNetwork_basic, name),
				Check: resource.ComposeTestCheckFunc(
					func(*terraform.State) error {
						time.Sleep(10 * time.Second)
						return nil
					},
					testAccCheckOneandonePrivateNetworkExists("oneandone_private_network.pn", &net),
					testAccCheckOneandonePrivateNetworkAttributes("oneandone_private_network.pn", name),
					resource.TestCheckResourceAttr("oneandone_private_network.pn", "name", name),
				),
			},
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckOneandonePrivateNetwork_basic, name_updated),
				Check: resource.ComposeTestCheckFunc(
					func(*terraform.State) error {
						time.Sleep(10 * time.Second)
						return nil
					},
					testAccCheckOneandonePrivateNetworkExists("oneandone_private_network.pn", &net),
					testAccCheckOneandonePrivateNetworkAttributes("oneandone_private_network.pn", name_updated),
					resource.TestCheckResourceAttr("oneandone_private_network.pn", "name", name_updated),
				),
			},
		},
	})
}

func testAccCheckOneandonePrivateNetworkDestroyCheck(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "oneandone_private_network" {
			continue
		}

		api := oneandone.New(os.Getenv("ONEANDONE_TOKEN"), oneandone.BaseUrl)

		_, err := api.GetPrivateNetwork(rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("PrivateNetwork still exists %s %s", rs.Primary.ID, err.Error())
		}
	}

	return nil
}
func testAccCheckOneandonePrivateNetworkAttributes(n string, reverse_dns string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		if rs.Primary.Attributes["name"] != reverse_dns {
			return fmt.Errorf("Bad name: expected %s : found %s ", reverse_dns, rs.Primary.Attributes["name"])
		}

		return nil
	}
}

func testAccCheckOneandonePrivateNetworkExists(n string, server *oneandone.PrivateNetwork) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		api := oneandone.New(os.Getenv("ONEANDONE_TOKEN"), oneandone.BaseUrl)

		found_server, err := api.GetPrivateNetwork(rs.Primary.ID)

		if err != nil {
			return fmt.Errorf("Error occured while fetching PrivateNetwork: %s", rs.Primary.ID)
		}
		if found_server.Id != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}
		server = found_server

		return nil
	}
}

const testAccCheckOneandonePrivateNetwork_basic = `
resource "oneandone_server" "server1" {
  name = "server_private_net_01"
  description = "ttt"
  image = "CoreOS_Stable_64std"
  datacenter = "US"
  vcores = 1
  cores_per_processor = 1
  ram = 2
  password = "Kv40kd8PQb"
  hdds = [
    {
      disk_size = 60
      is_main = true
    }
  ]
}

resource "oneandone_server" "server2" {
  name = "server_private_net_02"
  description = "ttt"
  image = "CoreOS_Stable_64std"
  datacenter = "US"
  vcores = 1
  cores_per_processor = 1
  ram = 2
  password = "${oneandone_server.server1.password}"
  hdds = [
    {
      disk_size = 60
      is_main = true
    }
  ]
}

resource "oneandone_private_network" "pn" {
  name = "%s",
  description = "new private net"
  datacenter = "US"
  network_address = "192.168.7.0"
  subnet_mask = "255.255.255.0"
    server_ids = [
      "${oneandone_server.server1.id}",
      "${oneandone_server.server2.id}"
    ]
}
`
