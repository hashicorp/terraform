package grafana

import (
	"fmt"
	"regexp"
	"testing"

	gapi "github.com/apparentlymart/go-grafana-api"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDashboard_basic(t *testing.T) {
	var dashboard gapi.Dashboard

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccDashboardCheckDestroy(&dashboard),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDashboardConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccDashboardCheckExists("grafana_dashboard.test", &dashboard),
					resource.TestMatchResourceAttr(
						"grafana_dashboard.test", "id", regexp.MustCompile(`terraform-acceptance-test.*`),
					),
				),
			},
		},
	})
}

func testAccDashboardCheckExists(rn string, dashboard *gapi.Dashboard) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testAccProvider.Meta().(*gapi.Client)
		gotDashboard, err := client.Dashboard(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error getting dashboard: %s", err)
		}

		*dashboard = *gotDashboard

		return nil
	}
}

func testAccDashboardCheckDestroy(dashboard *gapi.Dashboard) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*gapi.Client)
		_, err := client.Dashboard(dashboard.Meta.Slug)
		if err == nil {
			return fmt.Errorf("dashboard still exists")
		}
		return nil
	}
}

// The "id" and "version" properties in the config below are there to test
// that we correctly normalize them away. They are not actually used by this
// resource, since it uses slugs for identification and never modifies an
// existing dashboard.
const testAccDashboardConfig_basic = `
resource "grafana_dashboard" "test" {
    config_json = <<EOT
{
    "title": "Terraform Acceptance Test",
    "id": 12,
    "version": "43"
}
EOT
}
`
