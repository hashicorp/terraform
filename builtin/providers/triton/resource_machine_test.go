package triton

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/joyent/gosdc/cloudapi"
)

func TestAccTritonMachine_basic(t *testing.T) {
	machineName := fmt.Sprintf("acctest-%d", acctest.RandInt())
	config := fmt.Sprintf(testAccTritonMachine_basic, machineName)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckTritonMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonMachineExists("triton_machine.test"),
					func(*terraform.State) error {
						time.Sleep(10 * time.Second)
						return nil
					},
				),
			},
		},
	})
}

func testCheckTritonMachineExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		conn := testAccProvider.Meta().(*cloudapi.Client)

		rule, err := conn.GetMachine(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Bad: Check Machine Exists: %s", err)
		}

		if rule == nil {
			return fmt.Errorf("Bad: Machine %q does not exist", rs.Primary.ID)
		}

		return nil
	}
}

func testCheckTritonMachineDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*cloudapi.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "triton_machine" {
			continue
		}

		resp, err := conn.GetMachine(rs.Primary.ID)
		if err != nil {
			return nil
		}

		if resp != nil {
			return fmt.Errorf("Bad: Machine %q still exists", rs.Primary.ID)
		}
	}

	return nil
}

var testAccTritonMachine_basic = `
resource "triton_machine" "test" {
  name = "%s"
  package = "g3-standard-0.25-smartos"
  image = "842e6fa6-6e9b-11e5-8402-1b490459e334"

  tags = {
	test = "hello!"
  }
}
`
