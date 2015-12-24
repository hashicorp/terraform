package datadog

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/zorkian/go-datadog-api"
)

func TestAccDatadogServiceCheck_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDatadogServiceCheckDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDatadogServiceCheckConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDatadogServiceCheckExists("datadog_service_check.bar"),
					resource.TestCheckResourceAttr(
						"datadog_service_check.bar", "name", "name for service check bar"),
					resource.TestCheckResourceAttr(
						"datadog_service_check.bar", "message", "description for service check bar"),
					resource.TestCheckResourceAttr(
						"datadog_service_check.bar", "check", "datadog.agent.up"),
					resource.TestCheckResourceAttr(
						"datadog_service_check.bar", "notify_no_data", "false"),
					resource.TestCheckResourceAttr(
						"datadog_service_check.bar", "tags.0", "environment:foo"),
					resource.TestCheckResourceAttr(
						"datadog_service_check.bar", "tags.1", "host:bar"),
					resource.TestCheckResourceAttr(
						"datadog_service_check.bar", "tags.#", "2"),
					resource.TestCheckResourceAttr(
						"datadog_service_check.bar", "keys.0", "foo"),
					resource.TestCheckResourceAttr(
						"datadog_service_check.bar", "keys.1", "bar"),
					resource.TestCheckResourceAttr(
						"datadog_service_check.bar", "keys.#", "2"),
				),
			},
		},
	})
}

func testAccCheckDatadogServiceCheckDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*datadog.Client)

	if err := destroyHelper(s, client); err != nil {
		return err
	}
	return nil
}

func testAccCheckDatadogServiceCheckExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*datadog.Client)
		if err := existsHelper(s, client); err != nil {
			return err
		}
		return nil
	}
}

const testAccCheckDatadogServiceCheckConfigBasic = `
resource "datadog_service_check" "bar" {
  name = "name for service check bar"
  message = "description for service check bar"
  tags = ["environment:foo", "host:bar"]
  keys = ["foo", "bar"]
  check = "datadog.agent.up"
  check_count = 3

  notify_no_data = false
}
`
