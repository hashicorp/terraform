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

func TestAccOneandoneServer_Basic(t *testing.T) {
	var server oneandone.Server

	name := "test_server"
	name_updated := "test_server_renamed"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDOneandoneServerDestroyCheck,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckOneandoneServer_basic, name, name),
				Check: resource.ComposeTestCheckFunc(
					func(*terraform.State) error {
						time.Sleep(10 * time.Second)
						return nil
					},
					testAccCheckOneandoneServerExists("oneandone_server.server", &server),
					testAccCheckOneandoneServerAttributes("oneandone_server.server", name),
					resource.TestCheckResourceAttr("oneandone_server.server", "name", name),
				),
			},
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckOneandoneServer_basic, name_updated, name_updated),
				Check: resource.ComposeTestCheckFunc(
					func(*terraform.State) error {
						time.Sleep(10 * time.Second)
						return nil
					},
					testAccCheckOneandoneServerExists("oneandone_server.server", &server),
					testAccCheckOneandoneServerAttributes("oneandone_server.server", name_updated),
					resource.TestCheckResourceAttr("oneandone_server.server", "name", name_updated),
				),
			},
		},
	})
}

func testAccCheckDOneandoneServerDestroyCheck(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "oneandone_server" {
			continue
		}

		api := oneandone.New(os.Getenv("ONEANDONE_TOKEN"), oneandone.BaseUrl)

		_, err := api.GetServer(rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("Server still exists %s %s", rs.Primary.ID, err.Error())
		}
	}

	return nil
}
func testAccCheckOneandoneServerAttributes(n string, reverse_dns string) resource.TestCheckFunc {
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

func testAccCheckOneandoneServerExists(n string, server *oneandone.Server) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		api := oneandone.New(os.Getenv("ONEANDONE_TOKEN"), oneandone.BaseUrl)

		found_server, err := api.GetServer(rs.Primary.ID)

		if err != nil {
			return fmt.Errorf("Error occured while fetching Server: %s", rs.Primary.ID)
		}
		if found_server.Id != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}
		server = found_server

		return nil
	}
}

const testAccCheckOneandoneServer_basic = `
resource "oneandone_server" "server" {
  name = "%s"
  description = "%s"
  image = "ubuntu"
  datacenter = "GB"
  vcores = 1
  cores_per_processor = 1
  ram = 2
  password = "Kv40kd8PQb"
  hdds = [
    {
      disk_size = 20
      is_main = true
    }
  ]
}`
