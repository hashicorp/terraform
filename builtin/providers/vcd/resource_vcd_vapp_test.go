package vcd

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	govcd "github.com/ukcloud/govcloudair"
)

func TestAccVcdVApp_PowerOff(t *testing.T) {
	var vapp govcd.VApp

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVcdVAppDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckVcdVApp_basic, os.Getenv("VCD_EDGE_GATWEWAY")),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVcdVAppExists("vcd_vapp.foobar", &vapp),
					testAccCheckVcdVAppAttributes(&vapp),
					resource.TestCheckResourceAttr(
						"vcd_vapp.foobar", "name", "foobar"),
					resource.TestCheckResourceAttr(
						"vcd_vapp.foobar", "ip", "10.10.102.160"),
					resource.TestCheckResourceAttr(
						"vcd_vapp.foobar", "power_on", "true"),
				),
			},
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckVcdVApp_powerOff, os.Getenv("VCD_EDGE_GATWEWAY")),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVcdVAppExists("vcd_vapp.foobar", &vapp),
					testAccCheckVcdVAppAttributes_off(&vapp),
					resource.TestCheckResourceAttr(
						"vcd_vapp.foobar", "name", "foobar"),
					resource.TestCheckResourceAttr(
						"vcd_vapp.foobar", "ip", "10.10.102.160"),
					resource.TestCheckResourceAttr(
						"vcd_vapp.foobar", "power_on", "false"),
				),
			},
		},
	})
}

func testAccCheckVcdVAppExists(n string, vapp *govcd.VApp) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No VAPP ID is set")
		}

		conn := testAccProvider.Meta().(*VCDClient)

		resp, err := conn.OrgVdc.FindVAppByName(rs.Primary.ID)
		if err != nil {
			return err
		}

		*vapp = resp

		return nil
	}
}

func testAccCheckVcdVAppDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*VCDClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "vcd_vapp" {
			continue
		}

		_, err := conn.OrgVdc.FindVAppByName(rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("VPCs still exist")
		}

		return nil
	}

	return nil
}

func testAccCheckVcdVAppAttributes(vapp *govcd.VApp) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if vapp.VApp.Name != "foobar" {
			return fmt.Errorf("Bad name: %s", vapp.VApp.Name)
		}

		if vapp.VApp.Name != vapp.VApp.Children.VM[0].Name {
			return fmt.Errorf("VApp and VM names do not match. %s != %s",
				vapp.VApp.Name, vapp.VApp.Children.VM[0].Name)
		}

		status, _ := vapp.GetStatus()
		if status != "POWERED_ON" {
			return fmt.Errorf("VApp is not powered on")
		}

		return nil
	}
}

func testAccCheckVcdVAppAttributes_off(vapp *govcd.VApp) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if vapp.VApp.Name != "foobar" {
			return fmt.Errorf("Bad name: %s", vapp.VApp.Name)
		}

		if vapp.VApp.Name != vapp.VApp.Children.VM[0].Name {
			return fmt.Errorf("VApp and VM names do not match. %s != %s",
				vapp.VApp.Name, vapp.VApp.Children.VM[0].Name)
		}

		status, _ := vapp.GetStatus()
		if status != "POWERED_OFF" {
			return fmt.Errorf("VApp is still powered on")
		}

		return nil
	}
}

const testAccCheckVcdVApp_basic = `
resource "vcd_network" "foonet" {
	name = "foonet"
	edge_gateway = "%s"
	gateway = "10.10.102.1"
	static_ip_pool {
		start_address = "10.10.102.2"
		end_address = "10.10.102.254"
	}
}

resource "vcd_vapp" "foobar" {
  name = "foobar"
  template_name = "base-centos-7.0-x86_64_v-0.1_b-74"
  catalog_name = "NubesLab"
  network_name = "${vcd_network.foonet.name}"
  memory = 1024
	cpus = 1
	ip = "10.10.102.160"
}
`

const testAccCheckVcdVApp_powerOff = `
resource "vcd_network" "foonet" {
	name = "foonet"
	edge_gateway = "%s"
	gateway = "10.10.102.1"
	static_ip_pool {
		start_address = "10.10.102.2"
		end_address = "10.10.102.254"
	}
}

resource "vcd_vapp" "foobar" {
  name = "foobar"
  template_name = "base-centos-7.0-x86_64_v-0.1_b-74"
  catalog_name = "NubesLab"
  network_name = "${vcd_network.foonet.name}"
  memory = 1024
	cpus = 1
	ip = "10.10.102.160"
	power_on = false
}
`
