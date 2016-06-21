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

func TestAccTritonMachine_nic(t *testing.T) {
	machineName := fmt.Sprintf("acctest-%d", acctest.RandInt())
	config := fmt.Sprintf(testAccTritonMachine_withnic, machineName, machineName)

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
					testCheckTritonMachineHasFabric("triton_machine.test", "triton_fabric.test"),
				),
			},
		},
	})
}

func TestAccTritonMachine_addnic(t *testing.T) {
	machineName := fmt.Sprintf("acctest-%d", acctest.RandInt())
	without := fmt.Sprintf(testAccTritonMachine_withoutnic, machineName, machineName)
	with := fmt.Sprintf(testAccTritonMachine_withnic, machineName, machineName)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckTritonMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: without,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonMachineExists("triton_machine.test"),
					func(*terraform.State) error {
						time.Sleep(10 * time.Second)
						return nil
					},
					testCheckTritonMachineHasNoFabric("triton_machine.test", "triton_fabric.test"),
				),
			},
			resource.TestStep{
				Config: with,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonMachineExists("triton_machine.test"),
					testCheckTritonMachineHasFabric("triton_machine.test", "triton_fabric.test"),
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

func testCheckTritonMachineHasFabric(name, fabricName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		machine, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		network, ok := s.RootModule().Resources[fabricName]
		if !ok {
			return fmt.Errorf("Not found: %s", fabricName)
		}
		conn := testAccProvider.Meta().(*cloudapi.Client)

		nics, err := conn.ListNICs(machine.Primary.ID)
		if err != nil {
			return fmt.Errorf("Bad: Check NICs Exist: %s", err)
		}

		for _, nic := range nics {
			if nic.Network == network.Primary.ID {
				return nil
			}
		}

		return fmt.Errorf("Bad: Machine %q does not have Fabric %q", machine.Primary.ID, network.Primary.ID)
	}
}

func testCheckTritonMachineHasNoFabric(name, fabricName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		machine, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		network, ok := s.RootModule().Resources[fabricName]
		if !ok {
			return fmt.Errorf("Not found: %s", fabricName)
		}
		conn := testAccProvider.Meta().(*cloudapi.Client)

		nics, err := conn.ListNICs(machine.Primary.ID)
		if err != nil {
			return fmt.Errorf("Bad: Check NICs Exist: %s", err)
		}

		for _, nic := range nics {
			if nic.Network == network.Primary.ID {
				return fmt.Errorf("Bad: Machine %q has Fabric %q", machine.Primary.ID, network.Primary.ID)
			}
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

func TestAccTritonMachine_firewall(t *testing.T) {
	machineName := fmt.Sprintf("acctest-%d", acctest.RandInt())
	disabled_config := fmt.Sprintf(testAccTritonMachine_firewall_0, machineName)
	enabled_config := fmt.Sprintf(testAccTritonMachine_firewall_1, machineName)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckTritonMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: enabled_config,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonMachineExists("triton_machine.test"),
					resource.TestCheckResourceAttr(
						"triton_machine.test", "firewall_enabled", "true"),
				),
			},
			resource.TestStep{
				Config: disabled_config,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonMachineExists("triton_machine.test"),
					resource.TestCheckResourceAttr(
						"triton_machine.test", "firewall_enabled", "false"),
				),
			},
			resource.TestStep{
				Config: enabled_config,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonMachineExists("triton_machine.test"),
					resource.TestCheckResourceAttr(
						"triton_machine.test", "firewall_enabled", "true"),
				),
			},
		},
	})
}

func TestAccTritonMachine_metadata(t *testing.T) {
	machineName := fmt.Sprintf("acctest-%d", acctest.RandInt())
	basic := fmt.Sprintf(testAccTritonMachine_metadata_1, machineName)
	add_metadata := fmt.Sprintf(testAccTritonMachine_metadata_1, machineName)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckTritonMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: basic,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonMachineExists("triton_machine.test"),
				),
			},
			resource.TestStep{
				Config: add_metadata,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonMachineExists("triton_machine.test"),
					resource.TestCheckResourceAttr(
						"triton_machine.test", "user_data", "hello"),
				),
			},
		},
	})
}

var testAccTritonMachine_basic = `
provider "triton" {
  url = "https://us-west-1.api.joyentcloud.com"
}

resource "triton_machine" "test" {
  name = "%s"
  package = "g3-standard-0.25-smartos"
  image = "c20b4b7c-e1a6-11e5-9a4d-ef590901732e"

  tags = {
	test = "hello!"
  }
}
`

var testAccTritonMachine_firewall_0 = `
provider "triton" {
  url = "https://us-west-1.api.joyentcloud.com"
}

resource "triton_machine" "test" {
  name = "%s"
  package = "g3-standard-0.25-smartos"
  image = "c20b4b7c-e1a6-11e5-9a4d-ef590901732e"

	firewall_enabled = 0
}
`
var testAccTritonMachine_firewall_1 = `
provider "triton" {
  url = "https://us-west-1.api.joyentcloud.com"
}

resource "triton_machine" "test" {
  name = "%s"
  package = "g3-standard-0.25-smartos"
  image = "c20b4b7c-e1a6-11e5-9a4d-ef590901732e"

	firewall_enabled = 1
}
`

var testAccTritonMachine_metadata_1 = `
provider "triton" {
  url = "https://us-west-1.api.joyentcloud.com"
}

resource "triton_machine" "test" {
  name = "%s"
  package = "g3-standard-0.25-smartos"
  image = "c20b4b7c-e1a6-11e5-9a4d-ef590901732e"

  user_data = "hello"

  tags = {
    test = "hello!"
	}
}
`

var testAccTritonMachine_withnic = `
resource "triton_fabric" "test" {
  name = "%s-network"
  description = "test network"
  vlan_id = 2 # every DC seems to have a vlan 2 available

  subnet = "10.0.0.0/22"
  gateway = "10.0.0.1"
  provision_start_ip = "10.0.0.5"
  provision_end_ip = "10.0.3.250"

  resolvers = ["8.8.8.8", "8.8.4.4"]
}

resource "triton_machine" "test" {
  name = "%s"
  package = "g3-standard-0.25-smartos"
  image = "842e6fa6-6e9b-11e5-8402-1b490459e334"

  tags = {
    test = "hello!"
	}

  nic { network = "${triton_fabric.test.id}" }
}
`

var testAccTritonMachine_withoutnic = `
resource "triton_fabric" "test" {
  name = "%s-network"
  description = "test network"
  vlan_id = 2 # every DC seems to have a vlan 2 available

  subnet = "10.0.0.0/22"
  gateway = "10.0.0.1"
  provision_start_ip = "10.0.0.5"
  provision_end_ip = "10.0.3.250"

  resolvers = ["8.8.8.8", "8.8.4.4"]
}

resource "triton_machine" "test" {
  name = "%s"
  package = "g3-standard-0.25-smartos"
  image = "842e6fa6-6e9b-11e5-8402-1b490459e334"

  tags = {
    test = "hello!"
	}
}
`
