package cloudstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func TestAccCloudStackNetworkACL_basic(t *testing.T) {
	var acl cloudstack.NetworkACLList
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackNetworkACLDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudStackNetworkACL_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackNetworkACLExists(
						"cloudstack_network_acl.foo", &acl),
					testAccCheckCloudStackNetworkACLBasicAttributes(&acl),
					resource.TestCheckResourceAttr(
						"cloudstack_network_acl.foo", "vpc", "terraform-vpc"),
				),
			},
		},
	})
}

func testAccCheckCloudStackNetworkACLExists(
	n string, acl *cloudstack.NetworkACLList) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No network ACL ID is set")
		}

		cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)
		acllist, _, err := cs.NetworkACL.GetNetworkACLListByID(rs.Primary.ID)
		if err != nil {
			return err
		}

		if acllist.Id != rs.Primary.ID {
			return fmt.Errorf("Network ACL not found")
		}

		*acl = *acllist

		return nil
	}
}

func testAccCheckCloudStackNetworkACLBasicAttributes(
	acl *cloudstack.NetworkACLList) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if acl.Name != "terraform-acl" {
			return fmt.Errorf("Bad name: %s", acl.Name)
		}

		if acl.Description != "terraform-acl-text" {
			return fmt.Errorf("Bad description: %s", acl.Description)
		}

		return nil
	}
}

func testAccCheckCloudStackNetworkACLDestroy(s *terraform.State) error {
	cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "cloudstack_network_acl" {
			continue
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No network ACL ID is set")
		}

		_, _, err := cs.NetworkACL.GetNetworkACLListByID(rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("Network ACl list %s still exists", rs.Primary.ID)
		}
	}

	return nil
}

var testAccCloudStackNetworkACL_basic = fmt.Sprintf(`
resource "cloudstack_vpc" "foobar" {
  name = "terraform-vpc"
  cidr = "%s"
  vpc_offering = "%s"
  zone = "%s"
}

resource "cloudstack_network_acl" "foo" {
  name = "terraform-acl"
  description = "terraform-acl-text"
  vpc = "${cloudstack_vpc.foobar.name}"
}`,
	CLOUDSTACK_VPC_CIDR_1,
	CLOUDSTACK_VPC_OFFERING,
	CLOUDSTACK_ZONE)
