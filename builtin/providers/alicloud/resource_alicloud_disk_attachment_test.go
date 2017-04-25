package alicloud

import (
	"fmt"
	"testing"

	"github.com/denverdino/aliyungo/ecs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"time"
)

func TestAccAlicloudDiskAttachment(t *testing.T) {
	var i ecs.InstanceAttributesType
	var v ecs.DiskItemType

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_disk_attachment.disk-att",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckDiskAttachmentDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDiskAttachmentConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists(
						"alicloud_instance.instance", &i),
					testAccCheckDiskExists(
						"alicloud_disk.disk", &v),
					testAccCheckDiskAttachmentExists(
						"alicloud_disk_attachment.disk-att", &i, &v),
					resource.TestCheckResourceAttr(
						"alicloud_disk_attachment.disk-att",
						"device_name",
						"/dev/xvdb"),
				),
			},
		},
	})

}

func testAccCheckDiskAttachmentExists(n string, instance *ecs.InstanceAttributesType, disk *ecs.DiskItemType) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Disk ID is set")
		}

		client := testAccProvider.Meta().(*AliyunClient)
		conn := client.ecsconn

		request := &ecs.DescribeDisksArgs{
			RegionId: client.Region,
			DiskIds:  []string{rs.Primary.Attributes["disk_id"]},
		}

		return resource.Retry(3*time.Minute, func() *resource.RetryError {
			response, _, err := conn.DescribeDisks(request)
			if response != nil {
				for _, d := range response {
					if d.Status != ecs.DiskStatusInUse {
						return resource.RetryableError(fmt.Errorf("Disk is in attaching - trying again while it attaches"))
					} else if d.InstanceId == instance.InstanceId {
						// pass
						*disk = d
						return nil
					}
				}
			}
			if err != nil {
				return resource.NonRetryableError(err)
			}

			return resource.NonRetryableError(fmt.Errorf("Error finding instance/disk"))
		})
	}
}

func testAccCheckDiskAttachmentDestroy(s *terraform.State) error {

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "alicloud_disk_attachment" {
			continue
		}
		// Try to find the Disk
		client := testAccProvider.Meta().(*AliyunClient)
		conn := client.ecsconn

		request := &ecs.DescribeDisksArgs{
			RegionId: client.Region,
			DiskIds:  []string{rs.Primary.ID},
		}

		response, _, err := conn.DescribeDisks(request)

		for _, disk := range response {
			if disk.Status != ecs.DiskStatusAvailable {
				return fmt.Errorf("Error ECS Disk Attachment still exist")
			}
		}

		if err != nil {
			// Verify the error is what we want
			return err
		}
	}

	return nil
}

const testAccDiskAttachmentConfig = `
resource "alicloud_disk" "disk" {
  availability_zone = "cn-beijing-a"
  size = "50"

  tags {
    Name = "TerraformTest-disk"
  }
}

resource "alicloud_instance" "instance" {
  image_id = "ubuntu_140405_64_40G_cloudinit_20161115.vhd"
  instance_type = "ecs.s1.small"
  availability_zone = "cn-beijing-a"
  security_groups = ["${alicloud_security_group.group.id}"]
  instance_name = "hello"
  internet_charge_type = "PayByBandwidth"
  io_optimized = "none"

  tags {
    Name = "TerraformTest-instance"
  }
}

resource "alicloud_disk_attachment" "disk-att" {
  disk_id = "${alicloud_disk.disk.id}"
  instance_id = "${alicloud_instance.instance.id}"
  device_name = "/dev/xvdb"
}

resource "alicloud_security_group" "group" {
  name = "terraform-test-group"
  description = "New security group"
}

`
