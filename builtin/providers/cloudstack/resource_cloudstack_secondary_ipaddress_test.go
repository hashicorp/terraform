package cloudstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func TestAccCloudStackSecondaryIPAddress_basic(t *testing.T) {
	var ip cloudstack.AddIpToNicResponse

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackSecondaryIPAddressDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudStackSecondaryIPAddress_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackSecondaryIPAddressExists(
						"cloudstack_secondary_ipaddress.foo", &ip),
				),
			},
		},
	})
}

func TestAccCloudStackSecondaryIPAddress_fixedIP(t *testing.T) {
	var ip cloudstack.AddIpToNicResponse

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackSecondaryIPAddressDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudStackSecondaryIPAddress_fixedIP,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackSecondaryIPAddressExists(
						"cloudstack_secondary_ipaddress.foo", &ip),
					testAccCheckCloudStackSecondaryIPAddressAttributes(&ip),
					resource.TestCheckResourceAttr(
						"cloudstack_secondary_ipaddress.foo", "ipaddress", CLOUDSTACK_NETWORK_1_IPADDRESS1),
				),
			},
		},
	})
}

func testAccCheckCloudStackSecondaryIPAddressExists(
	n string, ip *cloudstack.AddIpToNicResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No IP address ID is set")
		}

		cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)

		// Retrieve the virtual_machine ID
		virtualmachineid, e := retrieveID(
			cs, "virtual_machine", rs.Primary.Attributes["virtual_machine"])
		if e != nil {
			return e.Error()
		}

		// Get the virtual machine details
		vm, count, err := cs.VirtualMachine.GetVirtualMachineByID(virtualmachineid)
		if err != nil {
			if count == 0 {
				return fmt.Errorf("Instance not found")
			}
			return err
		}

		nicid := rs.Primary.Attributes["nicid"]
		if nicid == "" {
			nicid = vm.Nic[0].Id
		}

		p := cs.Nic.NewListNicsParams(virtualmachineid)
		p.SetNicid(nicid)

		l, err := cs.Nic.ListNics(p)
		if err != nil {
			return err
		}

		if l.Count == 0 {
			return fmt.Errorf("NIC not found")
		}

		if l.Count > 1 {
			return fmt.Errorf("Found more then one possible result: %v", l.Nics)
		}

		for _, sip := range l.Nics[0].Secondaryip {
			if sip.Id == rs.Primary.ID {
				ip.Ipaddress = sip.Ipaddress
				ip.Nicid = l.Nics[0].Id
				return nil
			}
		}

		return fmt.Errorf("IP address not found")
	}
}

func testAccCheckCloudStackSecondaryIPAddressAttributes(
	ip *cloudstack.AddIpToNicResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if ip.Ipaddress != CLOUDSTACK_NETWORK_1_IPADDRESS1 {
			return fmt.Errorf("Bad IP address: %s", ip.Ipaddress)
		}
		return nil
	}
}

func testAccCheckCloudStackSecondaryIPAddressDestroy(s *terraform.State) error {
	cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "cloudstack_secondary_ipaddress" {
			continue
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No IP address ID is set")
		}

		// Retrieve the virtual_machine ID
		virtualmachineid, e := retrieveID(
			cs, "virtual_machine", rs.Primary.Attributes["virtual_machine"])
		if e != nil {
			return e.Error()
		}

		// Get the virtual machine details
		vm, count, err := cs.VirtualMachine.GetVirtualMachineByID(virtualmachineid)
		if err != nil {
			if count == 0 {
				return fmt.Errorf("Instance not found")
			}
			return err
		}

		nicid := rs.Primary.Attributes["nicid"]
		if nicid == "" {
			nicid = vm.Nic[0].Id
		}

		p := cs.Nic.NewListNicsParams(virtualmachineid)
		p.SetNicid(nicid)

		l, err := cs.Nic.ListNics(p)
		if err != nil {
			return err
		}

		if l.Count == 0 {
			return fmt.Errorf("NIC not found")
		}

		if l.Count > 1 {
			return fmt.Errorf("Found more then one possible result: %v", l.Nics)
		}

		for _, sip := range l.Nics[0].Secondaryip {
			if sip.Id == rs.Primary.ID {
				return fmt.Errorf("IP address %s still exists", rs.Primary.ID)
			}
		}

		return nil
	}

	return nil
}

var testAccCloudStackSecondaryIPAddress_basic = fmt.Sprintf(`
resource "cloudstack_instance" "foobar" {
  name = "terraform-test"
  service_offering= "%s"
  network = "%s"
  template = "%s"
  zone = "%s"
  expunge = true
}

resource "cloudstack_secondary_ipaddress" "foo" {
	virtual_machine = "${cloudstack_instance.foobar.id}"
}
`,
	CLOUDSTACK_SERVICE_OFFERING_1,
	CLOUDSTACK_NETWORK_1,
	CLOUDSTACK_TEMPLATE,
	CLOUDSTACK_ZONE)

var testAccCloudStackSecondaryIPAddress_fixedIP = fmt.Sprintf(`
resource "cloudstack_instance" "foobar" {
  name = "terraform-test"
  service_offering= "%s"
  network = "%s"
  template = "%s"
  zone = "%s"
  expunge = true
}

resource "cloudstack_secondary_ipaddress" "foo" {
	ipaddress = "%s"
	virtual_machine = "${cloudstack_instance.foobar.id}"
}`,
	CLOUDSTACK_SERVICE_OFFERING_1,
	CLOUDSTACK_NETWORK_1,
	CLOUDSTACK_TEMPLATE,
	CLOUDSTACK_ZONE,
	CLOUDSTACK_NETWORK_1_IPADDRESS1)
