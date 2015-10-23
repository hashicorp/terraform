package datadog

import (
	"fmt"
	"strconv"
	"strings"
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
	for _, r := range s.RootModule().Resources {
		i, _ := strconv.Atoi(r.Primary.ID)
		_, err := client.GetMonitor(i)
		if err != nil {
			// 404 is what we want, anything else is an error. Sadly our API will return a string like so:
			// return errors.New("API error: " + resp.Status)
			// For now we'll use unfold :|
			if strings.EqualFold(err.Error(), "API error: 404 Not Found") {
				continue
			} else {
				fmt.Errorf("Received an error retreieving monitor %s", err)
			}
		} else {
			fmt.Errorf("Monitor still exists. %s", err)
		}
	}
	return nil
}

func testAccCheckDatadogServiceCheckExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*datadog.Client)
		for _, r := range s.RootModule().Resources {
			i, _ := strconv.Atoi(r.Primary.ID)
			_, err := client.GetMonitor(i)
			if err != nil {
				return fmt.Errorf("Received an error retrieving monitor %s", err)
			}
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
