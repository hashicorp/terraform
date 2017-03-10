package pagerduty

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataSourcePagerDutyVendor_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyScheduleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataSourcePagerDutyVendorConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourcePagerDutyVendor("data.pagerduty_vendor.foo"),
				),
			},
		},
	})
}

func TestAccDataSourcePagerDutyVendorLegacy_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyScheduleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataSourcePagerDutyVendorLegacyConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourcePagerDutyVendorLegacy("data.pagerduty_vendor.foo"),
				),
			},
		},
	})
}

func testAccDataSourcePagerDutyVendor(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		r := s.RootModule().Resources[n]
		a := r.Primary.Attributes

		if a["id"] == "" {
			return fmt.Errorf("Expected to get a vendor ID from PagerDuty")
		}

		if a["id"] != "PZQ6AUS" {
			return fmt.Errorf("Expected the Datadog Vendor ID to be: PZQ6AUS, but got: %s", a["id"])
		}

		if a["name"] != "Amazon Cloudwatch" {
			return fmt.Errorf("Expected the Datadog Vendor Name to be: Datadog, but got: %s", a["name"])
		}

		if a["type"] != "api" {
			return fmt.Errorf("Expected the Datadog Vendor Type to be: api, but got: %s", a["type"])
		}

		return nil
	}
}

func testAccDataSourcePagerDutyVendorLegacy(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		r := s.RootModule().Resources[n]
		a := r.Primary.Attributes

		if a["id"] == "" {
			return fmt.Errorf("Expected to get a vendor ID from PagerDuty")
		}

		if a["id"] != "PAM4FGS" {
			return fmt.Errorf("Expected the Datadog Vendor ID to be: PAM4FGS, but got: %s", a["id"])
		}

		if a["name"] != "Datadog" {
			return fmt.Errorf("Expected the Datadog Vendor Name to be: Datadog, but got: %s", a["name"])
		}

		if a["type"] != "generic_events_api_inbound_integration" {
			return fmt.Errorf("Expected the Datadog Vendor Type to be: generic_events_api_inbound_integration, but got: %s", a["type"])
		}

		return nil
	}
}

const testAccDataSourcePagerDutyVendorConfig = `
data "pagerduty_vendor" "foo" {
  name = "cloudwatch"
}
`

const testAccDataSourcePagerDutyVendorLegacyConfig = `
data "pagerduty_vendor" "foo" {
  name_regex = "Datadog"
}
`
