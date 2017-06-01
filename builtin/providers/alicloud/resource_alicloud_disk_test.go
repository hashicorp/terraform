package alicloud

import (
	"fmt"
	"testing"

	"github.com/denverdino/aliyungo/ecs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"log"
)

func TestAccAlicloudDisk_basic(t *testing.T) {
	var v ecs.DiskItemType

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_disk.foo",

		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDiskDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDiskConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDiskExists(
						"alicloud_disk.foo", &v),
					resource.TestCheckResourceAttr(
						"alicloud_disk.foo",
						"category",
						"cloud_efficiency"),
					resource.TestCheckResourceAttr(
						"alicloud_disk.foo",
						"size",
						"30"),
				),
			},
		},
	})

}

func TestAccAlicloudDisk_withTags(t *testing.T) {
	var v ecs.DiskItemType

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		//module name
		IDRefreshName: "alicloud_disk.bar",

		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDiskDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDiskConfigWithTags,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDiskExists("alicloud_disk.bar", &v),
					resource.TestCheckResourceAttr(
						"alicloud_disk.bar",
						"tags.Name",
						"TerraformTest"),
				),
			},
		},
	})
}

func testAccCheckDiskExists(n string, disk *ecs.DiskItemType) resource.TestCheckFunc {
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
			DiskIds:  []string{rs.Primary.ID},
		}

		response, _, err := conn.DescribeDisks(request)
		log.Printf("[WARN] disk ids %#v", rs.Primary.ID)

		if err == nil {
			if response != nil && len(response) > 0 {
				*disk = response[0]
				return nil
			}
		}
		return fmt.Errorf("Error finding ECS Disk %#v", rs.Primary.ID)
	}
}

func testAccCheckDiskDestroy(s *terraform.State) error {

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "alicloud_disk" {
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

		if response != nil && len(response) > 0 {
			return fmt.Errorf("Error ECS Disk still exist")
		}

		if err != nil {
			// Verify the error is what we want
			return err
		}
	}

	return nil
}

const testAccDiskConfig = `
data "alicloud_zones" "default" {
	"available_disk_category"= "cloud_efficiency"
}

resource "alicloud_disk" "foo" {
	# cn-beijing
	availability_zone = "${data.alicloud_zones.default.zones.0.id}"
	name = "New-disk"
	description = "Hello ecs disk."
	category = "cloud_efficiency"
        size = "30"
}
`
const testAccDiskConfigWithTags = `
data "alicloud_zones" "default" {
	"available_disk_category"= "cloud_efficiency"
}

resource "alicloud_disk" "bar" {
	# cn-beijing
	availability_zone = "${data.alicloud_zones.default.zones.0.id}"
	category = "cloud_efficiency"
        size = "20"
        tags {
        	Name = "TerraformTest"
        }
}
`
