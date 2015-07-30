package cloudstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func TestAccCloudStackNIC_basic(t *testing.T) {
	var nic cloudstack.Nic

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackNICDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudStackNIC_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackNICExists(
						"cloudstack_instance.foobar", "cloudstack_nic.foo", &nic),
					testAccCheckCloudStackNICAttributes(&nic),
				),
			},
		},
	})
}

func TestAccCloudStackNIC_update(t *testing.T) {
	var nic cloudstack.Nic

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackNICDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudStackNIC_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackNICExists(
						"cloudstack_instance.foobar", "cloudstack_nic.foo", &nic),
					testAccCheckCloudStackNICAttributes(&nic),
				),
			},

			resource.TestStep{
				Config: testAccCloudStackNIC_ipaddress,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackNICExists(
						"cloudstack_instance.foobar", "cloudstack_nic.foo", &nic),
					testAccCheckCloudStackNICIPAddress(&nic),
					resource.TestCheckResourceAttr(
						"cloudstack_nic.foo", "ipaddress", CLOUDSTACK_2ND_NIC_IPADDRESS),
				),
			},
		},
	})
}

func testAccCheckCloudStackNICExists(
	v, n string, nic *cloudstack.Nic) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rsv, ok := s.RootModule().Resources[v]
		if !ok {
			return fmt.Errorf("Not found: %s", v)
		}

		if rsv.Primary.ID == "" {
			return fmt.Errorf("No instance ID is set")
		}

		rsn, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rsn.Primary.ID == "" {
			return fmt.Errorf("No NIC ID is set")
		}

		cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)
		vm, _, err := cs.VirtualMachine.GetVirtualMachineByID(rsv.Primary.ID)

		if err != nil {
			return err
		}

		for _, n := range vm.Nic {
			if n.Id == rsn.Primary.ID {
				*nic = n
				return nil
			}
		}

		return fmt.Errorf("NIC not found")
	}
}

func testAccCheckCloudStackNICAttributes(
	nic *cloudstack.Nic) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if nic.Networkname != CLOUDSTACK_2ND_NIC_NETWORK {
			return fmt.Errorf("Bad network: %s", nic.Networkname)
		}

		return nil
	}
}

func testAccCheckCloudStackNICIPAddress(
	nic *cloudstack.Nic) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if nic.Networkname != CLOUDSTACK_2ND_NIC_NETWORK {
			return fmt.Errorf("Bad network: %s", nic.Networkname)
		}

		if nic.Ipaddress != CLOUDSTACK_2ND_NIC_IPADDRESS {
			return fmt.Errorf("Bad IP address: %s", nic.Ipaddress)
		}

		return nil
	}
}

func testAccCheckCloudStackNICDestroy(s *terraform.State) error {
	cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)

	// Deleting the instance automatically deletes any additional NICs
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "cloudstack_instance" {
			continue
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No instance ID is set")
		}

		_, _, err := cs.VirtualMachine.GetVirtualMachineByID(rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("Virtual Machine %s still exists", rs.Primary.ID)
		}
	}

	return nil
}

var testAccCloudStackNIC_basic = fmt.Sprintf(`
resource "cloudstack_instance" "foobar" {
  name = "terraform-test"
  display_name = "terraform"
  service_offering= "%s"
  network = "%s"
  template = "%s"
  zone = "%s"
  expunge = true
}

resource "cloudstack_nic" "foo" {
  network = "%s"
  virtual_machine = "${cloudstack_instance.foobar.name}"
}`,
	CLOUDSTACK_SERVICE_OFFERING_1,
	CLOUDSTACK_NETWORK_1,
	CLOUDSTACK_TEMPLATE,
	CLOUDSTACK_ZONE,
	CLOUDSTACK_2ND_NIC_NETWORK)

var testAccCloudStackNIC_ipaddress = fmt.Sprintf(`
resource "cloudstack_instance" "foobar" {
  name = "terraform-test"
  display_name = "terraform"
  service_offering= "%s"
  network = "%s"
  template = "%s"
  zone = "%s"
  expunge = true
}

resource "cloudstack_nic" "foo" {
  network = "%s"
  ipaddress = "%s"
  virtual_machine = "${cloudstack_instance.foobar.name}"
}`,
	CLOUDSTACK_SERVICE_OFFERING_1,
	CLOUDSTACK_NETWORK_1,
	CLOUDSTACK_TEMPLATE,
	CLOUDSTACK_ZONE,
	CLOUDSTACK_2ND_NIC_NETWORK,
	CLOUDSTACK_2ND_NIC_IPADDRESS)
