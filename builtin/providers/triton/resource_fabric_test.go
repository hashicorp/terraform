package triton

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/joyent/triton-go"
)

func TestAccTritonFabric_basic(t *testing.T) {
	fabricName := fmt.Sprintf("acctest-%d", acctest.RandInt())
	config := fmt.Sprintf(testAccTritonFabric_basic, acctest.RandIntRange(3, 2049), fabricName, fabricName)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckTritonFabricDestroy,
		Steps: []resource.TestStep{
			{
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
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		conn := testAccProvider.Meta().(*triton.Client)

		vlanID, err := strconv.Atoi(rs.Primary.Attributes["vlan_id"])
		if err != nil {
			return err
		}

		exists, err := resourceExists(conn.Fabrics().GetFabricNetwork(context.Background(), &triton.GetFabricNetworkInput{
			FabricVLANID: vlanID,
			NetworkID:    rs.Primary.ID,
		}))
		if err != nil {
			return fmt.Errorf("Error: Check Fabric Exists: %s", err)
		}

		if exists {
			return nil
		}

		return fmt.Errorf("Error: Fabric %q (VLAN %d) Does Not Exist", rs.Primary.ID, vlanID)
	}
}

func testCheckTritonFabricDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*triton.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "triton_fabric" {
			continue
		}

		vlanID, err := strconv.Atoi(rs.Primary.Attributes["vlan_id"])
		if err != nil {
			return err
		}

		exists, err := resourceExists(conn.Fabrics().GetFabricNetwork(context.Background(), &triton.GetFabricNetworkInput{
			FabricVLANID: vlanID,
			NetworkID:    rs.Primary.ID,
		}))
		if err != nil {
			return nil
		}

		if exists {
			return fmt.Errorf("Error: Fabric %q (VLAN %d) Still Exists", rs.Primary.ID, vlanID)
		}

		return nil
	}

	return nil
}

var testAccTritonFabric_basic = `
resource "triton_vlan" "test" {
  vlan_id = "%d"
  name = "%s"
  description = "testAccTritonFabric_basic"
}

resource "triton_fabric" "test" {
  name = "%s"
  description = "test network"
  vlan_id = "${triton_vlan.test.id}"

  subnet = "10.0.0.0/22"
  gateway = "10.0.0.1"
  provision_start_ip = "10.0.0.5"
  provision_end_ip = "10.0.3.250"

  resolvers = ["8.8.8.8", "8.8.4.4"]
}
`
