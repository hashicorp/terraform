package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSNetworkAclAssociation(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_network_acl.bar",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSNetworkAclDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSNetworkAclAssoc,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAwsRMNetworkAclAssocExists("aws_network_acl_association.test"),
				),
			},
		},
	})
}

func testCheckAwsRMNetworkAclAssocExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		_, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		return nil
	}
}

const testAccAWSNetworkAclAssoc = `
resource "aws_vpc" "testespvpc" {
  cidr_block = "10.1.0.0/16"
	tags {
		Name = "testAccAWSNetworkAclEsp"
	}
}

 resource "aws_network_acl" "acl_a" {
   vpc_id = "${aws_vpc.testespvpc.id}"

   tags {
     Name = "terraform test"
   }
 }

 resource "aws_subnet" "sunet_a" {
   vpc_id = "${aws_vpc.testespvpc.id}"
   cidr_block = "10.0.33.0/24"
   tags {
     Name = "terraform test"
   }
 }

 resource "aws_network_acl_association" "test" {
   network_acl_id = "${aws_network_acl.acl_a.id}"
   subnet_id = "${aws_subnet.subnet_a.id}"
 }
}
`
