package triton

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/joyent/gosdc/cloudapi"
)

func TestAccTritonFabric_basic(t *testing.T) {
	fabricName := fmt.Sprintf("acctest-%d", acctest.RandInt())
	config := fmt.Sprintf(testAccTritonFabric_basic, fabricName)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckTritonFabricDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonFabricExists("triton_fabric.test"),
					func(*terraform.State) error {
						time.Sleep(10 * time.Second)
						return nil
					},
				),
			},
		},
	})
}

func testCheckTritonFabricExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		conn := testAccProvider.Meta().(*cloudapi.Client)

		id, err := strconv.ParseInt(rs.Primary.Attributes["vlan_id"], 10, 16)
		if err != nil {
			return err
		}

		fabric, err := conn.GetFabricNetwork(int16(id), rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Bad: Check Fabric Exists: %s", err)
		}

		if fabric == nil {
			return fmt.Errorf("Bad: Fabric %q does not exist", rs.Primary.ID)
		}

		return nil
	}
}

func testCheckTritonFabricDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*cloudapi.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "triton_fabric" {
			continue
		}

		id, err := strconv.ParseInt(rs.Primary.Attributes["vlan_id"], 10, 16)
		if err != nil {
			return err
		}

		fabric, err := conn.GetFabricNetwork(int16(id), rs.Primary.ID)
		if err != nil {
			return nil
		}

		if fabric != nil {
			return fmt.Errorf("Bad: Fabric %q still exists", rs.Primary.ID)
		}
	}

	return nil
}

var testAccTritonFabric_basic = `
resource "triton_fabric" "test" {
  name = "%s"
  description = "test network"
  vlan_id = 2 # every DC seems to have a vlan 2 available

  subnet = "10.0.0.0/22"
  gateway = "10.0.0.1"
  provision_start_ip = "10.0.0.5"
  provision_end_ip = "10.0.3.250"

  resolvers = ["8.8.8.8", "8.8.4.4"]
}
`
