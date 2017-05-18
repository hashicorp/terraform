package openstack

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/gophercloud/gophercloud/openstack/dns/v2/zones"
)

func TestAccDNSV2Zone_basic(t *testing.T) {
	var zone zones.Zone

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheckDNSZoneV2(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDNSV2ZoneDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDNSV2Zone_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDNSV2ZoneExists("openstack_dns_zone_v2.zone_1", &zone),
				),
			},
			resource.TestStep{
				Config: testAccDNSV2Zone_update,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openstack_dns_zone_v2.zone_1", "name", "example.com"),
					resource.TestCheckResourceAttr("openstack_dns_zone_v2.zone_1", "email", "email2@example.com"),
					resource.TestCheckResourceAttr("openstack_dns_zone_v2.zone_1", "ttl", "6000"),
				),
			},
		},
	})
}

func TestAccDNSV2Zone_timeout(t *testing.T) {
	var zone zones.Zone

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheckDNSZoneV2(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDNSV2ZoneDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDNSV2Zone_timeout,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDNSV2ZoneExists("openstack_dns_zone_v2.zone_1", &zone),
				),
			},
		},
	})
}

func testAccCheckDNSV2ZoneDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	dnsClient, err := config.dnsV2Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("Error creating OpenStack DNS client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_dns_zone_v2" {
			continue
		}

		_, err := zones.Get(dnsClient, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("Zone still exists")
		}
	}

	return nil
}

func testAccCheckDNSV2ZoneExists(n string, zone *zones.Zone) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)
		dnsClient, err := config.dnsV2Client(OS_REGION_NAME)
		if err != nil {
			return fmt.Errorf("Error creating OpenStack DNS client: %s", err)
		}

		found, err := zones.Get(dnsClient, rs.Primary.ID).Extract()
		if err != nil {
			return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("Zone not found")
		}

		*zone = *found

		return nil
	}
}

func testAccPreCheckDNSZoneV2(t *testing.T) {
	v := os.Getenv("OS_AUTH_URL")
	if v == "" {
		t.Fatal("OS_AUTH_URL must be set for acceptance tests")
	}
}

const testAccDNSV2Zone_basic = `
resource "openstack_dns_zone_v2" "zone_1" {
  name = "example.com."
	email = "email1@example.com"
  ttl = 3000
}
`

const testAccDNSV2Zone_update = `
resource "openstack_dns_zone_v2" "zone_1" {
  name = "example.com."
	email = "email2@example.com"
  ttl = 6000
}
`

const testAccDNSV2Zone_timeout = `
resource "openstack_dns_zone_v2" "zone_1" {
  name = "example.com."
	email = "email@example.com"
  ttl = 3000

  timeouts {
    create = "5m"
    delete = "5m"
  }
}
`
