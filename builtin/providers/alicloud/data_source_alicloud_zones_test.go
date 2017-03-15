package alicloud

import (
	"github.com/hashicorp/terraform/helper/resource"
	"testing"
)

func TestAccAlicloudZonesDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAlicloudZonesDataSourceBasicConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAlicloudDataSourceID("data.alicloud_zones.foo"),
				),
			},
		},
	})
}

func TestAccAlicloudZonesDataSource_filter(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAlicloudZonesDataSourceFilter,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAlicloudDataSourceID("data.alicloud_zones.foo"),
					resource.TestCheckResourceAttr("data.alicloud_zones.foo", "zones.#", "2"),
				),
			},

			resource.TestStep{
				Config: testAccCheckAlicloudZonesDataSourceFilterIoOptimized,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAlicloudDataSourceID("data.alicloud_zones.foo"),
					resource.TestCheckResourceAttr("data.alicloud_zones.foo", "zones.#", "1"),
				),
			},
		},
	})
}

const testAccCheckAlicloudZonesDataSourceBasicConfig = `
data "alicloud_zones" "foo" {
}
`

const testAccCheckAlicloudZonesDataSourceFilter = `
data "alicloud_zones" "foo" {
	"available_instance_type"= "ecs.c2.xlarge"
	"available_resource_creation"= "VSwitch"
	"available_disk_category"= "cloud_efficiency"
}
`

const testAccCheckAlicloudZonesDataSourceFilterIoOptimized = `
data "alicloud_zones" "foo" {
	"available_instance_type"= "ecs.c2.xlarge"
	"available_resource_creation"= "IoOptimized"
	"available_disk_category"= "cloud"
}
`
