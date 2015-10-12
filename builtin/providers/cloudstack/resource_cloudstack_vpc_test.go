package cloudstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func TestAccCloudStackVPC_basic(t *testing.T) {
	var vpc cloudstack.VPC

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackVPCDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudStackVPC_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackVPCExists(
						"cloudstack_vpc.foo", &vpc),
					testAccCheckCloudStackVPCAttributes(&vpc),
					resource.TestCheckResourceAttr(
						"cloudstack_vpc.foo", "vpc_offering", CLOUDSTACK_VPC_OFFERING),
				),
			},
		},
	})
}

func testAccCheckCloudStackVPCExists(
	n string, vpc *cloudstack.VPC) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No VPC ID is set")
		}

		cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)
		v, _, err := cs.VPC.GetVPCByID(rs.Primary.ID)

		if err != nil {
			return err
		}

		if v.Id != rs.Primary.ID {
			return fmt.Errorf("VPC not found")
		}

		*vpc = *v

		return nil
	}
}

func testAccCheckCloudStackVPCAttributes(
	vpc *cloudstack.VPC) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if vpc.Name != "terraform-vpc" {
			return fmt.Errorf("Bad name: %s", vpc.Name)
		}

		if vpc.Displaytext != "terraform-vpc-text" {
			return fmt.Errorf("Bad display text: %s", vpc.Displaytext)
		}

		if vpc.Cidr != CLOUDSTACK_VPC_CIDR_1 {
			return fmt.Errorf("Bad VPC CIDR: %s", vpc.Cidr)
		}

		if vpc.Networkdomain != "terraform-domain" {
			return fmt.Errorf("Bad network domain: %s", vpc.Networkdomain)
		}

		return nil
	}
}

func testAccCheckCloudStackVPCDestroy(s *terraform.State) error {
	cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "cloudstack_vpc" {
			continue
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No VPC ID is set")
		}

		_, _, err := cs.VPC.GetVPCByID(rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("VPC %s still exists", rs.Primary.ID)
		}
	}

	return nil
}

var testAccCloudStackVPC_basic = fmt.Sprintf(`
resource "cloudstack_vpc" "foo" {
  name = "terraform-vpc"
  display_text = "terraform-vpc-text"
  cidr = "%s"
  vpc_offering = "%s"
  network_domain = "terraform-domain"
  zone = "%s"
}`,
	CLOUDSTACK_VPC_CIDR_1,
	CLOUDSTACK_VPC_OFFERING,
	CLOUDSTACK_ZONE)
