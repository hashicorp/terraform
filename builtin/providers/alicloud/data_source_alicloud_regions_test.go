package alicloud

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAlicloudRegionsDataSource_regions(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAlicloudRegionsDataSourceRegionsConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAlicloudDataSourceID("data.alicloud_regions.region"),

					resource.TestCheckResourceAttr("data.alicloud_regions.region", "name", "cn-beijing"),
					resource.TestCheckResourceAttr("data.alicloud_regions.region", "current", "true"),

					resource.TestCheckResourceAttr("data.alicloud_regions.region", "regions.#", "1"),

					resource.TestCheckResourceAttr("data.alicloud_regions.region", "regions.0.id", "cn-beijing"),
					resource.TestCheckResourceAttr("data.alicloud_regions.region", "regions.0.region_id", "cn-beijing"),
					resource.TestCheckResourceAttr("data.alicloud_regions.region", "regions.0.local_name", "华北 2"),
				),
			},
		},
	})
}

func TestAccAlicloudRegionsDataSource_name(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAlicloudRegionsDataSourceNameConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAlicloudDataSourceID("data.alicloud_regions.name_filtered_region"),
					resource.TestCheckResourceAttr("data.alicloud_regions.name_filtered_region", "name", "cn-hangzhou")),
			},
		},
	})
}

func TestAccAlicloudRegionsDataSource_current(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAlicloudRegionsDataSourceCurrentConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAlicloudDataSourceID("data.alicloud_regions.current_filtered_region"),
					resource.TestCheckResourceAttr("data.alicloud_regions.current_filtered_region", "current", "true"),
				),
			},
		},
	})
}

func TestAccAlicloudRegionsDataSource_empty(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAlicloudRegionsDataSourceEmptyConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAlicloudDataSourceID("data.alicloud_regions.empty_params_region"),

					resource.TestCheckResourceAttr("data.alicloud_regions.empty_params_region", "regions.0.id", "cn-shenzhen"),
					resource.TestCheckResourceAttr("data.alicloud_regions.empty_params_region", "regions.0.region_id", "cn-shenzhen"),
					resource.TestCheckResourceAttr("data.alicloud_regions.empty_params_region", "regions.0.local_name", "华南 1"),
				),
			},
		},
	})
}

// Instance store test - using centos regions
const testAccCheckAlicloudRegionsDataSourceRegionsConfig = `
data "alicloud_regions" "region" {
	name = "cn-beijing"
	current = true
}
`

// Testing name parameter
const testAccCheckAlicloudRegionsDataSourceNameConfig = `
data "alicloud_regions" "name_filtered_region" {
	name = "cn-hangzhou"
}
`

// Testing current parameter
const testAccCheckAlicloudRegionsDataSourceCurrentConfig = `
data "alicloud_regions" "current_filtered_region" {
	current = true
}
`

// Testing empty parmas
const testAccCheckAlicloudRegionsDataSourceEmptyConfig = `
data "alicloud_regions" "empty_params_region" {
}
`
