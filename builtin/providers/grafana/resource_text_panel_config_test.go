package grafana

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccTextPanelConfig(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccTextPanelConfigConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"grafana_text_panel_config.test", "json", testAccTextPanelConfigExpected,
					),
				),
			},
		},
	})
}

const testAccTextPanelConfigConfig = `
resource "grafana_text_panel_config" "test" {
    title = "Text Panel"
    content = "Foo bar baz"
}
`

const testAccTextPanelConfigExpected = "{\"content\":\"Foo bar baz\",\"mode\":\"markdown\",\"style\":{},\"title\":\"Text Panel\",\"type\":\"text\"}"
