package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDnsRecordSet_basic(t *testing.T) {
	zoneName := fmt.Sprintf("dnszone-test-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDnsRecordSetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDnsRecordSet_basic(zoneName, "127.0.0.10", 300),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDnsRecordSetExists(
						"google_dns_record_set.foobar", zoneName),
				),
			},
		},
	})
}

func TestAccDnsRecordSet_modify(t *testing.T) {
	zoneName := fmt.Sprintf("dnszone-test-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDnsRecordSetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDnsRecordSet_basic(zoneName, "127.0.0.10", 300),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDnsRecordSetExists(
						"google_dns_record_set.foobar", zoneName),
				),
			},
			resource.TestStep{
				Config: testAccDnsRecordSet_basic(zoneName, "127.0.0.11", 300),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDnsRecordSetExists(
						"google_dns_record_set.foobar", zoneName),
				),
			},
			resource.TestStep{
				Config: testAccDnsRecordSet_basic(zoneName, "127.0.0.11", 600),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDnsRecordSetExists(
						"google_dns_record_set.foobar", zoneName),
				),
			},
		},
	})
}

func TestAccDnsRecordSet_changeType(t *testing.T) {
	zoneName := fmt.Sprintf("dnszone-test-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDnsRecordSetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDnsRecordSet_basic(zoneName, "127.0.0.10", 300),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDnsRecordSetExists(
						"google_dns_record_set.foobar", zoneName),
				),
			},
			resource.TestStep{
				Config: testAccDnsRecordSet_bigChange(zoneName, 600),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDnsRecordSetExists(
						"google_dns_record_set.foobar", zoneName),
				),
			},
		},
	})
}

func testAccCheckDnsRecordSetDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		// Deletion of the managed_zone implies everything is gone
		if rs.Type == "google_dns_managed_zone" {
			_, err := config.clientDns.ManagedZones.Get(
				config.Project, rs.Primary.ID).Do()
			if err == nil {
				return fmt.Errorf("DNS ManagedZone still exists")
			}
		}
	}

	return nil
}

func testAccCheckDnsRecordSetExists(resourceType, resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceType]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		dnsName := rs.Primary.Attributes["name"]
		dnsType := rs.Primary.Attributes["type"]

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		resp, err := config.clientDns.ResourceRecordSets.List(
			config.Project, resourceName).Name(dnsName).Type(dnsType).Do()
		if err != nil {
			return fmt.Errorf("Error confirming DNS RecordSet existence: %#v", err)
		}
		switch len(resp.Rrsets) {
		case 0:
			// The resource doesn't exist anymore
			return fmt.Errorf("DNS RecordSet not found")
		case 1:
			return nil
		default:
			return fmt.Errorf("Only expected 1 record set, got %d", len(resp.Rrsets))
		}
	}
}

func testAccDnsRecordSet_basic(zoneName string, addr2 string, ttl int) string {
	return fmt.Sprintf(`
	resource "google_dns_managed_zone" "parent-zone" {
		name = "%s"
		dns_name = "hashicorptest.com."
		description = "Test Description"
	}
	resource "google_dns_record_set" "foobar" {
		managed_zone = "${google_dns_managed_zone.parent-zone.name}"
		name = "test-record.hashicorptest.com."
		type = "A"
		rrdatas = ["127.0.0.1", "%s"]
		ttl = %d
	}
	`, zoneName, addr2, ttl)
}

func testAccDnsRecordSet_bigChange(zoneName string, ttl int) string {
	return fmt.Sprintf(`
	resource "google_dns_managed_zone" "parent-zone" {
		name = "%s"
		dns_name = "hashicorptest.com."
		description = "Test Description"
	}
	resource "google_dns_record_set" "foobar" {
		managed_zone = "${google_dns_managed_zone.parent-zone.name}"
		name = "test-record.hashicorptest.com."
		type = "CNAME"
		rrdatas = ["www.terraform.io."]
		ttl = %d
	}
	`, zoneName, ttl)
}
