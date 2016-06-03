package triton

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccTritonTags_basic(t *testing.T) {
	machineName := fmt.Sprintf("acctest-%d", acctest.RandInt())
	step_0 := fmt.Sprintf(testAccTritonTags_basic_0, machineName)
	step_10 := fmt.Sprintf(testAccTritonTags_basic_10, machineName)

	machine := "triton_machine.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckTritonMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: step_0,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonMachineExists(machine),
					resource.TestCheckResourceAttr(machine,
						"triton_cns_status",
						"up",
					),
					resource.TestCheckResourceAttr(machine,
						"tags.triton_cns_services",
						"bastion",
					),
					resource.TestCheckResourceAttr(machine,
						"tags.triton_cns_reverse__ptr",
						"www.joyent.com",
					),
				),
			},
			resource.TestStep{
				Config: step_10,
				Check: resource.ComposeTestCheckFunc(
					func(*terraform.State) error {
						time.Sleep(10 * time.Second)
						return nil
					},
					resource.TestCheckResourceAttr(machine,
						"triton_cns_status",
						"down",
					),
				),
			},
		},
	})
}

var testAccTritonTags_basic_0 = `
resource "triton_machine" "test" {
  name = "%s"
  package = "t4-standard-128M"
  image = "eb9fc1ea-e19a-11e5-bb27-8b954d8c125c"

  tags = {
    triton_cns_services = "bastion"
    triton_cns_reverse__ptr = "www.joyent.com"
  }
}
`

var testAccTritonTags_basic_10 = `
resource "triton_machine" "test" {
  name = "%s"
  package = "t4-standard-128M"
  image = "eb9fc1ea-e19a-11e5-bb27-8b954d8c125c"

	triton_cns_status = "down"

  tags = {
    triton_cns_services = "bastion"
    triton_cns_reverse__ptr = "www.joyent.com"
  }
}
`
