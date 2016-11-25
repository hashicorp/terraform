package newrelic

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccNewRelicApplication_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNewRelicApplicationConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccNewRelicApplication("data.newrelic_application.content"),
				),
			},
		},
	})
}

func testAccNewRelicApplication(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		r := s.RootModule().Resources[n]
		a := r.Primary.Attributes

		if a["id"] == "" {
			return fmt.Errorf("Expected to get an application from New Relic")
		}

		if a["name"] != testAccExpectedApplicationName {
			return fmt.Errorf("Expected the application name to be: %s, but got: %s", testAccExpectedApplicationName, a["name"])
		}

		return nil
	}
}

// To test this you must create an application with this name manually in New Relic.

const testAccExpectedApplicationName = "service-content"
const testAccNewRelicApplicationConfig = `
data "newrelic_application" "content" {
	name = "service-content"
}
`
