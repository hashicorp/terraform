package vsphere

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"golang.org/x/net/context"
)

func TestAccVSphereVirtualMachine_basic(t *testing.T) {
	var vm virtualMachine
	var locationOpt string
	var datastoreOpt string

	if v := os.Getenv("VSPHERE_DATACENTER"); v != "" {
		locationOpt += fmt.Sprintf("    datacenter = \"%s\"\n", v)
	}
	if v := os.Getenv("VSPHERE_CLUSTER"); v != "" {
		locationOpt += fmt.Sprintf("    cluster = \"%s\"\n", v)
	}
	if v := os.Getenv("VSPHERE_RESOURCE_POOL"); v != "" {
		locationOpt += fmt.Sprintf("    resource_pool = \"%s\"\n", v)
	}
	if v := os.Getenv("VSPHERE_DATASTORE"); v != "" {
		datastoreOpt = fmt.Sprintf("        datastore = \"%s\"\n", v)
	}
	template := os.Getenv("VSPHERE_TEMPLATE")
	gateway := os.Getenv("VSPHERE_NETWORK_GATEWAY")
	label := os.Getenv("VSPHERE_NETWORK_LABEL")
	ip_address := os.Getenv("VSPHERE_NETWORK_IP_ADDRESS")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(
					testAccCheckVSphereVirtualMachineConfig_basic,
					locationOpt,
					gateway,
					label,
					ip_address,
					datastoreOpt,
					template,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVSphereVirtualMachineExists("vsphere_virtual_machine.foo", &vm),
					resource.TestCheckResourceAttr(
						"vsphere_virtual_machine.foo", "name", "terraform-test"),
					resource.TestCheckResourceAttr(
						"vsphere_virtual_machine.foo", "vcpu", "2"),
					resource.TestCheckResourceAttr(
						"vsphere_virtual_machine.foo", "memory", "4096"),
					resource.TestCheckResourceAttr(
						"vsphere_virtual_machine.foo", "disk.#", "2"),
					resource.TestCheckResourceAttr(
						"vsphere_virtual_machine.foo", "disk.0.template", template),
					resource.TestCheckResourceAttr(
						"vsphere_virtual_machine.foo", "network_interface.#", "1"),
					resource.TestCheckResourceAttr(
						"vsphere_virtual_machine.foo", "network_interface.0.label", label),
				),
			},
		},
	})
}

func TestAccVSphereVirtualMachine_dhcp(t *testing.T) {
	var vm virtualMachine
	var locationOpt string
	var datastoreOpt string

	if v := os.Getenv("VSPHERE_DATACENTER"); v != "" {
		locationOpt += fmt.Sprintf("    datacenter = \"%s\"\n", v)
	}
	if v := os.Getenv("VSPHERE_CLUSTER"); v != "" {
		locationOpt += fmt.Sprintf("    cluster = \"%s\"\n", v)
	}
	if v := os.Getenv("VSPHERE_RESOURCE_POOL"); v != "" {
		locationOpt += fmt.Sprintf("    resource_pool = \"%s\"\n", v)
	}
	if v := os.Getenv("VSPHERE_DATASTORE"); v != "" {
		datastoreOpt = fmt.Sprintf("        datastore = \"%s\"\n", v)
	}
	template := os.Getenv("VSPHERE_TEMPLATE")
	label := os.Getenv("VSPHERE_NETWORK_LABEL_DHCP")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(
					testAccCheckVSphereVirtualMachineConfig_dhcp,
					locationOpt,
					label,
					datastoreOpt,
					template,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVSphereVirtualMachineExists("vsphere_virtual_machine.bar", &vm),
					resource.TestCheckResourceAttr(
						"vsphere_virtual_machine.bar", "name", "terraform-test"),
					resource.TestCheckResourceAttr(
						"vsphere_virtual_machine.bar", "vcpu", "2"),
					resource.TestCheckResourceAttr(
						"vsphere_virtual_machine.bar", "memory", "4096"),
					resource.TestCheckResourceAttr(
						"vsphere_virtual_machine.bar", "disk.#", "1"),
					resource.TestCheckResourceAttr(
						"vsphere_virtual_machine.bar", "disk.0.template", template),
					resource.TestCheckResourceAttr(
						"vsphere_virtual_machine.bar", "network_interface.#", "1"),
					resource.TestCheckResourceAttr(
						"vsphere_virtual_machine.bar", "network_interface.0.label", label),
				),
			},
		},
	})
}

					testAccCheckVSphereVirtualMachineExists("vsphere_virtual_machine.bar", &vm),
					resource.TestCheckResourceAttr(
						"vsphere_virtual_machine.car", "name", "terraform-test"),
					resource.TestCheckResourceAttr(
						"vsphere_virtual_machine.car", "vcpu", "2"),
					resource.TestCheckResourceAttr(
						"vsphere_virtual_machine.car", "memory", "4096"),
					resource.TestCheckResourceAttr(
						"vsphere_virtual_machine.car", "disk.#", "1"),
					resource.TestCheckResourceAttr(
						"vsphere_virtual_machine.car", "disk.0.template", template),
					resource.TestCheckResourceAttr(
						"vsphere_virtual_machine.car", "network_interface.#", "1"),
					resource.TestCheckResourceAttr(
						"vsphere_virtual_machine.car", "custom_configuration_parameters.foo", "bar"),
					resource.TestCheckResourceAttr(
						"vsphere_virtual_machine.car", "custom_configuration_parameters.car", "ferrai"),
					resource.TestCheckResourceAttr(
						"vsphere_virtual_machine.car", "custom_configuration_parameters.num", "42"),
					resource.TestCheckResourceAttr(
						"vsphere_virtual_machine.bar", "network_interface.0.label", label),
				),
			},
		},
	})
}

func testAccCheckVSphereVirtualMachineDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*govmomi.Client)
	finder := find.NewFinder(client.Client, true)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "vsphere_virtual_machine" {
			continue
		}

		dc, err := finder.Datacenter(context.TODO(), rs.Primary.Attributes["datacenter"])
		if err != nil {
			return fmt.Errorf("error %s", err)
		}

		dcFolders, err := dc.Folders(context.TODO())
		if err != nil {
			return fmt.Errorf("error %s", err)
		}

		_, err = object.NewSearchIndex(client.Client).FindChild(context.TODO(), dcFolders.VmFolder, rs.Primary.Attributes["name"])
		if err == nil {
			return fmt.Errorf("Record still exists")
		}
	}

	return nil
}

func testAccCheckVSphereVirtualMachineExists(n string, vm *virtualMachine) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client := testAccProvider.Meta().(*govmomi.Client)
		finder := find.NewFinder(client.Client, true)

		dc, err := finder.Datacenter(context.TODO(), rs.Primary.Attributes["datacenter"])
		if err != nil {
			return fmt.Errorf("error %s", err)
		}

		dcFolders, err := dc.Folders(context.TODO())
		if err != nil {
			return fmt.Errorf("error %s", err)
		}

		_, err = object.NewSearchIndex(client.Client).FindChild(context.TODO(), dcFolders.VmFolder, rs.Primary.Attributes["name"])

		*vm = virtualMachine{
			name: rs.Primary.ID,
		}

		return nil
	}
}

const testAccCheckVSphereVirtualMachineConfig_basic = `
resource "vsphere_virtual_machine" "foo" {
    name = "terraform-test"
%s
    vcpu = 2
    memory = 4096
    gateway = "%s"
    network_interface {
        label = "%s"
        ip_address = "%s"
        subnet_mask = "255.255.255.0"
    }
    disk {
%s
        template = "%s"
        iops = 500
    }
    disk {
        size = 1
        iops = 500
    }
}
`
const testAccCheckVSphereVirtualMachineConfig_dhcp = `
resource "vsphere_virtual_machine" "bar" {
    name = "terraform-test"
%s
    vcpu = 2
    memory = 4096
    network_interface {
        label = "%s"
    }
    disk {
%s
        template = "%s"
    }
}
`
        foo = "bar",
	car = "ferrai",
	num = 42
