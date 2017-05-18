package alicloud

import (
	"github.com/hashicorp/terraform/helper/resource"
	"testing"
)

func TestAccAlicloudInstanceTypesDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAlicloudInstanceTypesDataSourceBasicConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAlicloudDataSourceID("data.alicloud_instance_types.4c8g"),

					resource.TestCheckResourceAttr("data.alicloud_instance_types.4c8g", "instance_types.0.cpu_core_count", "4"),
					resource.TestCheckResourceAttr("data.alicloud_instance_types.4c8g", "instance_types.0.memory_size", "8"),
					resource.TestCheckResourceAttr("data.alicloud_instance_types.4c8g", "instance_types.0.id", "ecs.s3.large"),
				),
			},

			resource.TestStep{
				Config: testAccCheckAlicloudInstanceTypesDataSourceBasicConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAlicloudDataSourceID("data.alicloud_instance_types.4c8g"),

					resource.TestCheckResourceAttr("data.alicloud_instance_types.4c8g", "instance_types.#", "1"),

					resource.TestCheckResourceAttr("data.alicloud_instance_types.4c8g", "instance_types.0.cpu_core_count", "4"),
					resource.TestCheckResourceAttr("data.alicloud_instance_types.4c8g", "instance_types.0.memory_size", "8"),
				),
			},
		},
	})
}

const testAccCheckAlicloudInstanceTypesDataSourceBasicConfig = `
data "alicloud_instance_types" "4c8g" {
	cpu_core_count = 4
	memory_size = 8
}
`

const testAccCheckAlicloudInstanceTypesDataSourceBasicConfigUpdate = `
data "alicloud_instance_types" "4c8g" {
	instance_type_family= "ecs.s3"
	cpu_core_count = 4
	memory_size = 8
}
`
