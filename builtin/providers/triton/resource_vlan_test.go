package triton

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/joyent/gosdc/cloudapi"
)

func TestAccTritonVLAN_basic(t *testing.T) {
	config := testAccTritonVLAN_basic

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckTritonVLANDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonVLANExists("triton_vlan.test"),
				),
			},
		},
	})
}

func TestAccTritonVLAN_update(t *testing.T) {
	preConfig := testAccTritonVLAN_basic
	postConfig := testAccTritonVLAN_update

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckTritonVLANDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonVLANExists("triton_vlan.test"),
					resource.TestCheckResourceAttr("triton_vlan.test", "name", "test-vlan"),
					resource.TestCheckResourceAttr("triton_vlan.test", "description", "test vlan"),
				),
			},

			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonVLANExists("triton_vlan.test"),
					resource.TestCheckResourceAttr("triton_vlan.test", "name", "test-vlan-2"),
					resource.TestCheckResourceAttr("triton_vlan.test", "description", "test vlan 2"),
				),
			},
		},
	})
}

func testCheckTritonVLANExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		conn := testAccProvider.Meta().(*cloudapi.Client)

		id, err := resourceVLANIDInt16(rs.Primary.ID)
		if err != nil {
			return err
		}

		rule, err := conn.GetFabricVLAN(id)
		if err != nil {
			return fmt.Errorf("Bad: Check VLAN Exists: %s", err)
		}

		if rule == nil {
			return fmt.Errorf("Bad: VLAN %q does not exist", rs.Primary.ID)
		}

		return nil
	}
}

func testCheckTritonVLANDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*cloudapi.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "triton_vlan" {
			continue
		}

		id, err := resourceVLANIDInt16(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.GetFabricVLAN(id)
		if err != nil {
			return nil
		}

		if resp != nil {
			return fmt.Errorf("Bad: VLAN %q still exists", rs.Primary.ID)
		}
	}

	return nil
}

var testAccTritonVLAN_basic = `
resource "triton_vlan" "test" {
  vlan_id = 1024
  name = "test-vlan"
  description = "test vlan"
}
`

var testAccTritonVLAN_update = `
resource "triton_vlan" "test" {
  vlan_id = 1024
  name = "test-vlan-2"
  description = "test vlan 2"
}
`
