package ovh

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccPublicCloudRegionDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccPublicCloudRegionDatasourceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccPublicCloudRegionDatasource("data.ovh_publiccloud_region.region_attr.0"),
					testAccPublicCloudRegionDatasource("data.ovh_publiccloud_region.region_attr.1"),
					testAccPublicCloudRegionDatasource("data.ovh_publiccloud_region.region_attr.2"),
				),
			},
		},
	})
}

func testAccPublicCloudRegionDatasource(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Can't find regions data source: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Cannot find region attributes for project %s and region %s", rs.Primary.Attributes["project_id"], rs.Primary.Attributes["region"])
		}

		return nil
	}
}

var testAccPublicCloudRegionDatasourceConfig = fmt.Sprintf(`
data "ovh_publiccloud_regions" "regions" {
  project_id = "%s"
}

data "ovh_publiccloud_region" "region_attr" {
  count = 3
  project_id = "${data.ovh_publiccloud_regions.regions.project_id}"
  name = "${element(data.ovh_publiccloud_regions.regions.names, count.index)}"
}
`, os.Getenv("OVH_PUBLIC_CLOUD"))
