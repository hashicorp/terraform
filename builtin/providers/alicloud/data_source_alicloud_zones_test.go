package alicloud

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"strconv"
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
					testCheckZoneLength("data.alicloud_zones.foo"),
				),
			},

			resource.TestStep{
				Config: testAccCheckAlicloudZonesDataSourceFilterIoOptimized,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAlicloudDataSourceID("data.alicloud_zones.foo"),
					testCheckZoneLength("data.alicloud_zones.foo"),
				),
			},
		},
	})
}

func TestAccAlicloudZonesDataSource_unitRegion(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAlicloudZonesDataSource_unitRegion,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAlicloudDataSourceID("data.alicloud_zones.foo"),
				),
			},
		},
	})
}

// the zone length changed occasionally
// check by range to avoid test case failure
func testCheckZoneLength(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ms := s.RootModule()
		rs, ok := ms.Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		is := rs.Primary
		if is == nil {
			return fmt.Errorf("No primary instance: %s", name)
		}

		i, err := strconv.Atoi(is.Attributes["zones.#"])

		if err != nil {
			return fmt.Errorf("convert zone length err: %#v", err)
		}

		if i <= 0 {
			return fmt.Errorf("zone length expected greater than 0 got err: %d", i)
		}

		return nil
	}
}

const testAccCheckAlicloudZonesDataSourceBasicConfig = `
data "alicloud_zones" "foo" {
}
`

const testAccCheckAlicloudZonesDataSourceFilter = `
data "alicloud_zones" "foo" {
	available_instance_type= "ecs.c2.xlarge"
	available_resource_creation= "VSwitch"
	available_disk_category= "cloud_efficiency"
}
`

const testAccCheckAlicloudZonesDataSourceFilterIoOptimized = `
data "alicloud_zones" "foo" {
	available_instance_type= "ecs.c2.xlarge"
	available_resource_creation= "IoOptimized"
	available_disk_category= "cloud"
}
`

const testAccCheckAlicloudZonesDataSource_unitRegion = `
provider "alicloud" {
	alias = "northeast"
	region = "ap-northeast-1"
}

data "alicloud_zones" "foo" {
	provider = "alicloud.northeast"
	available_resource_creation= "VSwitch"
}
`
