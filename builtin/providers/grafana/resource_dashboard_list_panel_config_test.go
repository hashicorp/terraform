package grafana

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDashboardListPanelConfig(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDashboardListPanelConfigConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"grafana_dashboard_list_panel_config.test", "json", testAccDashboardListPanelConfigExpected,
					),
				),
			},
		},
	})
}

const testAccDashboardListPanelConfigConfig = `
resource "grafana_dashboard_list_panel_config" "test" {
    title = "Terraform-related Dashboards"
    mode = "search"
    tags = ["terraform"]
    query = "Terraform"
    limit = 100
}
`

const testAccDashboardListPanelConfigExpected = "{\"limit\":100,\"mode\":\"search\",\"query\":\"Terraform\",\"tags\":[\"terraform\"],\"title\":\"Terraform-related Dashboards\",\"type\":\"dashlist\"}"
