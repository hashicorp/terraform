package pagerduty

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccPagerDutyService_import(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyServiceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccPagerDutyServiceConfigImported,
			},
			resource.TestStep{
				ResourceName:      "pagerduty_service.foo",
				ImportState:       true,
				ImportStateVerify: false,
			},
		},
	})
}

const testAccPagerDutyServiceConfigImported = `
resource "pagerduty_service" "foo" {
  name = "foo"
  description = "foo"
	acknowledgement_timeout = "1800"
	auto_resolve_timeout = "14400"
  escalation_policy = "PGOMBUU"
}
`
