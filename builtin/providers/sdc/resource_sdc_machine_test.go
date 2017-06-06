package sdc

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/kiasaki/go-sdc"
)

const ImageBaseSmartOS = "c02a2044-c1bd-11e4-bd8c-dfc1db8b0182" // slug: base-64-lts
const PackageStandard05Smart = "g3-standard-0.5-smartos"        // 06a0251c-038f-4eda-8af2-653c46b3aee8" // slug: g3-standard-0.5-smartos

func TestAccSDCMachine_Basic(t *testing.T) {
	var machine sdc.Machine

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSDCMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckSDCMachineConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSDCMachineExists("sdc_machine.foobar", &machine),
					testAccCheckSDCMachineAttributes(&machine),
					resource.TestCheckResourceAttr(
						"sdc_machine.foobar", "name", "foo"),
					resource.TestCheckResourceAttr(
						"sdc_machine.foobar", "image", ImageBaseSmartOS),
					resource.TestCheckResourceAttr(
						"sdc_machine.foobar", "package", PackageStandard05Smart),
				),
			},
		},
	})
}

func testAccCheckSDCMachineDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*sdc.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "sdc_machine" {
			continue
		}

		// Try to find the Machine
		_, err := client.GetMachine(rs.Primary.ID)

		if sdcErr, ok := err.(sdc.SDCError); err != nil && (!ok || sdcErr.Code != "ResourceNotFound") {
			return fmt.Errorf(
				"Error waiting for machine (%s) to be destroyed: %s",
				rs.Primary.ID, err)
		}
	}

	return nil
}

func testAccCheckSDCMachineAttributes(machine *sdc.Machine) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if machine.Image != ImageBaseSmartOS {
			return fmt.Errorf("Bad image: %s", machine.Image)
		}

		if machine.Package != PackageStandard05Smart {
			return fmt.Errorf("Bad package: %s", machine.Package)
		}

		if machine.Name != "foo" {
			return fmt.Errorf("Bad name: %s", machine.Name)
		}

		return nil
	}
}

func testAccCheckSDCMachineExists(n string, machine *sdc.Machine) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Machine ID is set")
		}

		client := testAccProvider.Meta().(*sdc.Client)

		retrivedMachine, err := client.GetMachine(rs.Primary.ID)

		if err != nil {
			return err
		}

		if retrivedMachine.Id != rs.Primary.ID {
			return fmt.Errorf("Machine not found")
		}

		*machine = *retrivedMachine

		return nil
	}
}

const testAccCheckSDCMachineConfig_basic = `
resource "sdc_machine" "foobar" {
    name = "foo"
    image = "c02a2044-c1bd-11e4-bd8c-dfc1db8b0182"
    package = "g3-standard-0.5-smartos"

	metadata {
	  bar = "baz"
	}
	tags {
	  size = "5"
	}
}
`
