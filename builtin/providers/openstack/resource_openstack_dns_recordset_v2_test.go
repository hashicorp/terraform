package openstack

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/gophercloud/gophercloud/openstack/dns/v2/recordsets"
)

func randomZoneName() string {
	return fmt.Sprintf("ACPTTEST-zone-%s.com.", acctest.RandString(5))
}

func TestAccDNSV2RecordSet_basic(t *testing.T) {
	var recordset recordsets.RecordSet
	zoneName := randomZoneName()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheckDNSRecordSetV2(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDNSV2RecordSetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDNSV2RecordSet_basic(zoneName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDNSV2RecordSetExists("openstack_dns_recordset_v2.recordset_1", &recordset),
					resource.TestCheckResourceAttr(
						"openstack_dns_recordset_v2.recordset_1", "description", "a record set"),
					resource.TestCheckResourceAttr(
						"openstack_dns_recordset_v2.recordset_1", "records.0", "10.1.0.0"),
				),
			},
			resource.TestStep{
				Config: testAccDNSV2RecordSet_update(zoneName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openstack_dns_recordset_v2.recordset_1", "name", zoneName),
					resource.TestCheckResourceAttr("openstack_dns_recordset_v2.recordset_1", "ttl", "6000"),
					resource.TestCheckResourceAttr("openstack_dns_recordset_v2.recordset_1", "type", "A"),
					resource.TestCheckResourceAttr(
						"openstack_dns_recordset_v2.recordset_1", "description", "an updated record set"),
					resource.TestCheckResourceAttr(
						"openstack_dns_recordset_v2.recordset_1", "records.0", "10.1.0.1"),
				),
			},
		},
	})
}

func TestAccDNSV2RecordSet_readTTL(t *testing.T) {
	var recordset recordsets.RecordSet
	zoneName := randomZoneName()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheckDNSRecordSetV2(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDNSV2RecordSetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDNSV2RecordSet_readTTL(zoneName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDNSV2RecordSetExists("openstack_dns_recordset_v2.recordset_1", &recordset),
					resource.TestMatchResourceAttr(
						"openstack_dns_recordset_v2.recordset_1", "ttl", regexp.MustCompile("^[0-9]+$")),
				),
			},
		},
	})
}

func TestAccDNSV2RecordSet_timeout(t *testing.T) {
	var recordset recordsets.RecordSet
	zoneName := randomZoneName()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheckDNSRecordSetV2(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDNSV2RecordSetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDNSV2RecordSet_timeout(zoneName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDNSV2RecordSetExists("openstack_dns_recordset_v2.recordset_1", &recordset),
				),
			},
		},
	})
}

func testAccCheckDNSV2RecordSetDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	dnsClient, err := config.dnsV2Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("Error creating OpenStack DNS client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_dns_recordset_v2" {
			continue
		}

		zoneID, recordsetID, err := parseDNSV2RecordSetID(rs.Primary.ID)
		if err != nil {
			return err
		}

		_, err = recordsets.Get(dnsClient, zoneID, recordsetID).Extract()
		if err == nil {
			return fmt.Errorf("Record set still exists")
		}
	}

	return nil
}

func testAccCheckDNSV2RecordSetExists(n string, recordset *recordsets.RecordSet) resource.TestCheckFunc {
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

		zoneID, recordsetID, err := parseDNSV2RecordSetID(rs.Primary.ID)
		if err != nil {
			return err
		}

		found, err := recordsets.Get(dnsClient, zoneID, recordsetID).Extract()
		if err != nil {
			return err
		}

		if found.ID != recordsetID {
			return fmt.Errorf("Record set not found")
		}

		*recordset = *found

		return nil
	}
}

func testAccPreCheckDNSRecordSetV2(t *testing.T) {
	if os.Getenv("OS_AUTH_URL") == "" {
		t.Fatal("OS_AUTH_URL must be set for acceptance tests")
	}
}

func testAccDNSV2RecordSet_basic(zoneName string) string {
	return fmt.Sprintf(`
		resource "openstack_dns_zone_v2" "zone_1" {
			name = "%s"
			email = "email2@example.com"
			description = "a zone"
			ttl = 6000
			type = "PRIMARY"
		}

		resource "openstack_dns_recordset_v2" "recordset_1" {
			zone_id = "${openstack_dns_zone_v2.zone_1.id}"
			name = "%s"
			type = "A"
			description = "a record set"
			ttl = 3000
			records = ["10.1.0.0"]
		}
	`, zoneName, zoneName)
}

func testAccDNSV2RecordSet_update(zoneName string) string {
	return fmt.Sprintf(`
		resource "openstack_dns_zone_v2" "zone_1" {
			name = "%s"
			email = "email2@example.com"
			description = "an updated zone"
			ttl = 6000
			type = "PRIMARY"
		}

		resource "openstack_dns_recordset_v2" "recordset_1" {
			zone_id = "${openstack_dns_zone_v2.zone_1.id}"
			name = "%s"
			type = "A"
			description = "an updated record set"
			ttl = 6000
			records = ["10.1.0.1"]
		}
	`, zoneName, zoneName)
}

func testAccDNSV2RecordSet_readTTL(zoneName string) string {
	return fmt.Sprintf(`
		resource "openstack_dns_zone_v2" "zone_1" {
			name = "%s"
			email = "email2@example.com"
			description = "an updated zone"
			ttl = 6000
			type = "PRIMARY"
		}

		resource "openstack_dns_recordset_v2" "recordset_1" {
			zone_id = "${openstack_dns_zone_v2.zone_1.id}"
			name = "%s"
			type = "A"
			records = ["10.1.0.2"]
		}
	`, zoneName, zoneName)
}

func testAccDNSV2RecordSet_timeout(zoneName string) string {
	return fmt.Sprintf(`
		resource "openstack_dns_zone_v2" "zone_1" {
			name = "%s"
			email = "email2@example.com"
			description = "an updated zone"
			ttl = 6000
			type = "PRIMARY"
		}

		resource "openstack_dns_recordset_v2" "recordset_1" {
			zone_id = "${openstack_dns_zone_v2.zone_1.id}"
			name = "%s"
			type = "A"
			ttl = 3000
			records = ["10.1.0.3"]

			timeouts {
				create = "5m"
				update = "5m"
				delete = "5m"
			}
		}
	`, zoneName, zoneName)
}
