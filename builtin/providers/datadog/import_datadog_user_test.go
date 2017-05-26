package datadog

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestDatadogUser_import(t *testing.T) {
	resourceName := "datadog_user.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDatadogUserDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDatadogUserConfigImported,
			},
			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

const testAccCheckDatadogUserConfigImported = `
resource "datadog_user" "foo" {
  email  = "test@example.com"
  handle = "test@example.com"
  name   = "Test User"
}
`
