package ovh

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccPublicCloudRegionsDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccPublicCloudRegionsDatasourceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccPublicCloudRegionsDatasource("data.ovh_publiccloud_regions.regions"),
				),
			},
		},
	})
}

func testAccPublicCloudRegionsDatasource(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Can't find regions data source: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Cannot find regions for project %s", rs.Primary.Attributes["project_id"])
		}

		return nil
	}
}

var testAccPublicCloudRegionsDatasourceConfig = fmt.Sprintf(`
data "ovh_publiccloud_regions" "regions" {
  project_id = "%s"
}
`, os.Getenv("OVH_PUBLIC_CLOUD"))
