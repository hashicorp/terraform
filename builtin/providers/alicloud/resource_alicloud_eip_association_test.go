package alicloud

import (
	"fmt"
	"testing"

	"github.com/denverdino/aliyungo/ecs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"time"
)

func TestAccAlicloudEIPAssociation(t *testing.T) {
	var asso ecs.EipAddressSetType
	var inst ecs.InstanceAttributesType

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_eip_association.foo",

		Providers:    testAccProviders,
		CheckDestroy: testAccCheckEIPAssociationDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccEIPAssociationConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists(
						"alicloud_instance.instance", &inst),
					testAccCheckEIPExists(
						"alicloud_eip.eip", &asso),
					testAccCheckEIPAssociationExists(
						"alicloud_eip_association.foo", &inst, &asso),
				),
			},
		},
	})

}

func testAccCheckEIPAssociationExists(n string, instance *ecs.InstanceAttributesType, eip *ecs.EipAddressSetType) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No EIP Association ID is set")
		}

		client := testAccProvider.Meta().(*AliyunClient)
		return resource.Retry(3*time.Minute, func() *resource.RetryError {
			d, err := client.DescribeEipAddress(rs.Primary.Attributes["allocation_id"])

			if err != nil {
				return resource.NonRetryableError(err)
			}

			if d != nil {
				if d.Status != ecs.EipStatusInUse {
					return resource.RetryableError(fmt.Errorf("Eip is in associating - trying again while it associates"))
				} else if d.InstanceId == instance.InstanceId {
					*eip = *d
					return nil
				}
			}

			return resource.NonRetryableError(fmt.Errorf("EIP Association not found"))
		})
	}
}

func testAccCheckEIPAssociationDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*AliyunClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "alicloud_eip_association" {
			continue
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No EIP Association ID is set")
		}

		// Try to find the EIP
		eips, _, err := client.ecsconn.DescribeEipAddresses(&ecs.DescribeEipAddressesArgs{
			RegionId:     client.Region,
			AllocationId: rs.Primary.Attributes["allocation_id"],
		})

		for _, eip := range eips {
			if eip.Status != ecs.EipStatusAvailable {
				return fmt.Errorf("Error EIP Association still exist")
			}
		}

		// Verify the error is what we want
		if err != nil {
			return err
		}
	}

	return nil
}

const testAccEIPAssociationConfig = `
resource "alicloud_vpc" "main" {
  cidr_block = "10.1.0.0/21"
}

resource "alicloud_vswitch" "main" {
  vpc_id = "${alicloud_vpc.main.id}"
  cidr_block = "10.1.1.0/24"
  availability_zone = "cn-beijing-a"
  depends_on = [
    "alicloud_vpc.main"]
}

resource "alicloud_instance" "instance" {
  image_id = "ubuntu_140405_64_40G_cloudinit_20161115.vhd"
  instance_type = "ecs.s1.small"
  availability_zone = "cn-beijing-a"
  security_groups = ["${alicloud_security_group.group.id}"]
  vswitch_id = "${alicloud_vswitch.main.id}"
  instance_name = "hello"
  io_optimized = "none"

  tags {
    Name = "TerraformTest-instance"
  }
}

resource "alicloud_eip" "eip" {
}

resource "alicloud_eip_association" "foo" {
  allocation_id = "${alicloud_eip.eip.id}"
  instance_id = "${alicloud_instance.instance.id}"
}

resource "alicloud_security_group" "group" {
  name = "terraform-test-group"
  description = "New security group"
  vpc_id = "${alicloud_vpc.main.id}"
}
`
