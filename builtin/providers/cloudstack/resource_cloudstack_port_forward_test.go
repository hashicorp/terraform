package cloudstack

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func TestAccCloudStackPortForward_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackPortForwardDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudStackPortForward_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackPortForwardsExist("cloudstack_port_forward.foo"),
					resource.TestCheckResourceAttr(
						"cloudstack_port_forward.foo", "ip_address_id", CLOUDSTACK_PUBLIC_IPADDRESS),
					resource.TestCheckResourceAttr(
						"cloudstack_port_forward.foo", "forward.#", "1"),
				),
			},
		},
	})
}

func TestAccCloudStackPortForward_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackPortForwardDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudStackPortForward_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackPortForwardsExist("cloudstack_port_forward.foo"),
					resource.TestCheckResourceAttr(
						"cloudstack_port_forward.foo", "ip_address_id", CLOUDSTACK_PUBLIC_IPADDRESS),
					resource.TestCheckResourceAttr(
						"cloudstack_port_forward.foo", "forward.#", "1"),
				),
			},

			resource.TestStep{
				Config: testAccCloudStackPortForward_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackPortForwardsExist("cloudstack_port_forward.foo"),
					resource.TestCheckResourceAttr(
						"cloudstack_port_forward.foo", "ip_address_id", CLOUDSTACK_PUBLIC_IPADDRESS),
					resource.TestCheckResourceAttr(
						"cloudstack_port_forward.foo", "forward.#", "2"),
				),
			},
		},
	})
}

func testAccCheckCloudStackPortForwardsExist(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No port forward ID is set")
		}

		for k, id := range rs.Primary.Attributes {
			if !strings.Contains(k, "uuid") {
				continue
			}

			cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)
			_, count, err := cs.Firewall.GetPortForwardingRuleByID(id)

			if err != nil {
				return err
			}

			if count == 0 {
				return fmt.Errorf("Port forward for %s not found", k)
			}
		}

		return nil
	}
}

func testAccCheckCloudStackPortForwardDestroy(s *terraform.State) error {
	cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "cloudstack_port_forward" {
			continue
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No port forward ID is set")
		}

		for k, id := range rs.Primary.Attributes {
			if !strings.Contains(k, "uuid") {
				continue
			}

			_, _, err := cs.Firewall.GetPortForwardingRuleByID(id)
			if err == nil {
				return fmt.Errorf("Port forward %s still exists", rs.Primary.ID)
			}
		}
	}

	return nil
}

var testAccCloudStackPortForward_basic = fmt.Sprintf(`
resource "cloudstack_instance" "foobar" {
  name = "terraform-test"
  service_offering= "%s"
  network = "%s"
  template = "%s"
  zone = "%s"
  expunge = true
}

resource "cloudstack_port_forward" "foo" {
  ip_address_id = "%s"

  forward {
    protocol = "tcp"
    private_port = 443
    public_port = 8443
    virtual_machine_id = "${cloudstack_instance.foobar.id}"
  }
}`,
	CLOUDSTACK_SERVICE_OFFERING_1,
	CLOUDSTACK_NETWORK_1,
	CLOUDSTACK_TEMPLATE,
	CLOUDSTACK_ZONE,
	CLOUDSTACK_PUBLIC_IPADDRESS)

var testAccCloudStackPortForward_update = fmt.Sprintf(`
resource "cloudstack_instance" "foobar" {
  name = "terraform-test"
  service_offering= "%s"
  network = "%s"
  template = "%s"
  zone = "%s"
  expunge = true
}

resource "cloudstack_port_forward" "foo" {
  ip_address_id = "%s"

  forward {
    protocol = "tcp"
    private_port = 443
    public_port = 8443
    virtual_machine_id = "${cloudstack_instance.foobar.id}"
  }

  forward {
    protocol = "tcp"
    private_port = 80
    public_port = 8080
    virtual_machine_id = "${cloudstack_instance.foobar.id}"
  }
}`,
	CLOUDSTACK_SERVICE_OFFERING_1,
	CLOUDSTACK_NETWORK_1,
	CLOUDSTACK_TEMPLATE,
	CLOUDSTACK_ZONE,
	CLOUDSTACK_PUBLIC_IPADDRESS)
