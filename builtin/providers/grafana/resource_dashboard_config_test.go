package grafana

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDashboardConfig(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDashboardConfigConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"grafana_dashboard_config.test", "json", testAccDashboardConfigExpected,
					),
				),
			},
		},
	})
}

const testAccDashboardConfigConfig = `
resource "grafana_dashboard_config" "test" {
    title = "Terraform Acceptance Tests"
    tags = ["terraform", "acctest"]

    row {
        title = "First Row"

        panel {
            id = 1
            span = 6
            config_json = "{}"
        }

        panel {
            id = 2
            span = 6
            config_json = "{}"
        }
    }

    annotation {
        data_source_name = "terraform-acc-test"
        query = "SELECT foo FROM baz"
        name = "Test Annotation"
        index_name = "stuff"
    }

    link {
        type = "link"
        title = "example.com"
        url = "http://example.com/"
        open_new_window = true
        tooltip = "Example Link"
        tags = ["terraform"]
        as_dropdown_menu = true
        keep_time_range = true
        keep_variable_values = true
    }

    template_variable {
        type = "query"
        data_source_name = "terraform-test"
        refresh_on_load = true
        name = "another"
        include_all_option = true
        query = "SELECT foo FROM baz"
        hide_label = true
        label = "Another Thing"
    }
}
`

const testAccDashboardConfigExpected = "{\"annotations\":{\"list\":[{\"datasource\":\"terraform-acc-test\",\"enable\":true,\"iconColor\":\"#C0C6BE\",\"iconSize\":13,\"index\":\"stuff\",\"lineColor\":\"rgba(255, 96, 96, 0.592157)\",\"name\":\"Test Annotation\",\"query\":\"SELECT foo FROM baz\",\"showLine\":true,\"tagsColumn\":\"\",\"tagsField\":\"\",\"textColumn\":\"\",\"textField\":\"\",\"timeField\":\"\",\"titleColumn\":\"\",\"titleField\":\"\"}]},\"editable\":false,\"hideControls\":true,\"links\":[{\"asDropdown\":true,\"icon\":\"external link\",\"includeVars\":true,\"keepTime\":true,\"tags\":[\"terraform\"],\"targetBlank\":true,\"title\":\"example.com\",\"tooltip\":\"Example Link\",\"type\":\"link\",\"url\":\"http://example.com/\"}],\"rows\":[{\"collapse\":false,\"editable\":false,\"height\":\"250px\",\"panels\":[{\"id\":1,\"span\":6},{\"id\":2,\"span\":6}],\"showTitle\":false,\"title\":\"First Row\"}],\"schemaVersion\":6,\"sharedCrosshair\":false,\"tags\":[\"terraform\",\"acctest\"],\"templating\":{\"list\":[{\"allFormat\":\"regex wildcard\",\"datasource\":\"terraform-test\",\"hideLabel\":true,\"includeAll\":true,\"label\":\"Another Thing\",\"multi\":false,\"multiFormat\":\"regex values\",\"name\":\"another\",\"query\":\"SELECT foo FROM baz\",\"refresh_on_load\":true,\"type\":\"query\"}]},\"timezone\":\"browser\",\"title\":\"Terraform Acceptance Tests\"}"
