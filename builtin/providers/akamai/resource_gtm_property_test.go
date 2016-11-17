package akamai

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAkamaiGTMPropertyBasic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccAkamaiGTMPropertyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckAkamaiGTMPropertyConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAkamaiGTMPropertyExists("akamai_gtm_property.test_property"),
					resource.TestCheckResourceAttr("akamai_gtm_property.test_property", "domain", "terraform-test.akadns.net"),
					resource.TestCheckResourceAttr("akamai_gtm_property.test_property", "name", "test_property"),
					resource.TestCheckResourceAttr("akamai_gtm_property.test_property", "cname", "example.com"),
				),
			},
		},
	})
}

func testAccAkamaiGTMPropertyDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Clients).GTM

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "akamai_gtm_property" {
			continue
		}
		name := rs.Primary.ID

		// Try to find the property
		_, err := client.Property("terraform-test.akadns.net", name)
		if err == nil {
			return fmt.Errorf("Property still exists")
		}
	}

	return nil
}

func testAccCheckAkamaiGTMPropertyExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found %s", rs)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		client := testAccProvider.Meta().(*Clients).GTM

		readProp, err := client.Property("terraform-test.akadns.net", rs.Primary.ID)

		if err != nil {
			return err
		}

		if readProp.Name != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}

		return nil
	}
}

const testAccCheckAkamaiGTMPropertyConfigBasic = `
resource "akamai_gtm_domain" "property_test_domain" {
	name = "terraform-test.akadns.net"
	type = "basic"
}

resource "akamai_gtm_data_center" "property_test_dc1" {
	name = "property_test_dc1"
	domain = "${akamai_gtm_domain.property_test_domain.name}"
	country = "GB"
	continent = "EU"
	city = "Downpatrick"
	longitude = -5.582
	latitude = 54.367
	depends_on = [
		"akamai_gtm_domain.property_test_domain"
	]
}

resource "akamai_gtm_data_center" "property_test_dc2" {
	name = "property_test_dc2"
	domain = "${akamai_gtm_domain.property_test_domain.name}"
	country = "IS"
	continent = "EU"
	city = "Snæfellsjökull"
	longitude = -23.776
	latitude = 64.808
	depends_on = [
		"akamai_gtm_data_center.property_test_dc1"
	]
}

resource "akamai_gtm_property" "test_property" {
	cname = "example.com"
	domain = "${akamai_gtm_domain.property_test_domain.name}"
	type = "weighted-round-robin"
	name = "test_property"
	balance_by_download_score = false
	dynamic_ttl = 300
	failover_delay = 0
	failback_delay = 0
	handout_mode = "normal"
	health_threshold = 0
	health_max = 0
	health_multiplier = 0
	load_imbalance_percentage = 10
	ipv6 = false
	score_aggregation_type = "mean"
	static_ttl = 600
	stickiness_bonus_percentage = 50
	stickiness_bonus_constant = 0
	use_computed_targets = false
  liveness_test {
    name = "terraform-provider-akamai automated acceptance tests"
    test_object = "/status"
    test_object_protocol = "HTTP"
    test_interval = 60
    disable_nonstandard_port_warning = false
    http_error_4xx = true
    http_error_3xx = true
    http_error_5xx = true
    test_object_port = 80
    test_timeout = 25
  }
	traffic_target {
		enabled = true
		data_center_id = "${akamai_gtm_data_center.property_test_dc1.id}"
		weight = 50.0
		name = "${akamai_gtm_data_center.property_test_dc1.name}"
		servers = [
			"1.2.3.4",
			"1.2.3.5"
		]
	}
	traffic_target {
		enabled = true
		data_center_id = "${akamai_gtm_data_center.property_test_dc2.id}"
		weight = 50.0
		name = "${akamai_gtm_data_center.property_test_dc2.name}"
		servers = [
			"1.2.3.6",
			"1.2.3.7"
		]
	}
}
`
