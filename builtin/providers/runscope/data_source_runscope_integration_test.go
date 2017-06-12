package runscope

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"os"
)

func TestAccDataSourceRunscopeIntegration_Basic(t *testing.T) {

	teamId := os.Getenv("RUNSCOPE_TEAM_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceRunscopeIntegrationConfig, teamId),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceRunscopeIntegration("data.runscope_integration.by_type"),
				),
			},
		},
	})
}

func testAccDataSourceRunscopeIntegration(dataSource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		r := s.RootModule().Resources[dataSource]
		a := r.Primary.Attributes

		if a["id"] == "" {
			return fmt.Errorf("Expected to get an integration ID from runscope data resource")
		}

		if a["type"] != "pagerduty" {
			return fmt.Errorf("Expected to get an integration type pagerduty from runscope data resource")
		}

		return nil
	}
}

const testAccDataSourceRunscopeIntegrationConfig = `
data "runscope_integration" "by_type" {
	team_uuid = "%s"
	type      = "pagerduty"
}
`
