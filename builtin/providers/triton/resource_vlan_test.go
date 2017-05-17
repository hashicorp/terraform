package triton

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/joyent/triton-go"
)

func TestAccTritonVLAN_basic(t *testing.T) {
	config := testAccTritonVLAN_basic(acctest.RandIntRange(3, 2048))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckTritonVLANDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonVLANExists("triton_vlan.test"),
				),
			},
		},
	})
}

func TestAccTritonVLAN_update(t *testing.T) {
	vlanNumber := acctest.RandIntRange(3, 2048)
	preConfig := testAccTritonVLAN_basic(vlanNumber)
	postConfig := testAccTritonVLAN_update(vlanNumber)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckTritonVLANDestroy,
		Steps: []resource.TestStep{
			{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonVLANExists("triton_vlan.test"),
					resource.TestCheckResourceAttr("triton_vlan.test", "vlan_id", strconv.Itoa(vlanNumber)),
					resource.TestCheckResourceAttr("triton_vlan.test", "name", "test-vlan"),
					resource.TestCheckResourceAttr("triton_vlan.test", "description", "test vlan"),
				),
			},

			{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonVLANExists("triton_vlan.test"),
					resource.TestCheckResourceAttr("triton_vlan.test", "vlan_id", strconv.Itoa(vlanNumber)),
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
		conn := testAccProvider.Meta().(*triton.Client)

		id, err := resourceVLANIDInt(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.Fabrics().GetFabricVLAN(context.Background(), &triton.GetFabricVLANInput{
			ID: id,
		})
		if err != nil && triton.IsResourceNotFound(err) {
			return fmt.Errorf("Bad: Check VLAN Exists: %s", err)
		} else if err != nil {
			return err
		}

		if resp == nil {
			return fmt.Errorf("Bad: VLAN %q does not exist", rs.Primary.ID)
		}

		return nil
	}
}

func testCheckTritonVLANDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*triton.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "triton_vlan" {
			continue
		}

		id, err := resourceVLANIDInt(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.Fabrics().GetFabricVLAN(context.Background(), &triton.GetFabricVLANInput{
			ID: id,
		})
		if triton.IsResourceNotFound(err) {
			return nil
		} else if err != nil {
			return err
		}

		if resp != nil {
			return fmt.Errorf("Bad: VLAN %q still exists", rs.Primary.ID)
		}
	}

	return nil
}

var testAccTritonVLAN_basic = func(vlanID int) string {
	return fmt.Sprintf(`resource "triton_vlan" "test" {
	  vlan_id = %d
	  name = "test-vlan"
	  description = "test vlan"
	}`, vlanID)
}

var testAccTritonVLAN_update = func(vlanID int) string {
	return fmt.Sprintf(`resource "triton_vlan" "test" {
	  vlan_id = %d
	  name = "test-vlan-2"
	  description = "test vlan 2"
	}`, vlanID)
}
