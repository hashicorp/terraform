package softlayer

import (
	"fmt"
	"strconv"
	"testing"

	datatypes "github.com/TheWeatherCompany/softlayer-go/data_types"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccSoftLayerDnsDomain_Basic(t *testing.T) {
	var dns_domain datatypes.SoftLayer_Dns_Domain

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSoftLayerDnsDomainDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckSoftLayerDnsDomainConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSoftLayerDnsDomainExists("softlayer_dns_domain.acceptance_test_dns_domain-1", &dns_domain),
					testAccCheckSoftLayerDnsDomainAttributes(&dns_domain),
					testAccCheckSoftLayerDnsDomainRecordDomainId("softlayer_dns_domain_record.recordA", &dns_domain),
					testAccCheckSoftLayerDnsDomainRecordDomainId("softlayer_dns_domain_record.recordAAAA", &dns_domain),
					saveSoftLayerDnsDomainId(&dns_domain, &firstDnsId),
					resource.TestCheckResourceAttr(
						"softlayer_dns_domain.acceptance_test_dns_domain-1", "name", test_dns_domain_name),
					resource.TestCheckResourceAttr(
						"softlayer_dns_domain_record.recordA", "host", "hosta.com"),
					resource.TestCheckResourceAttr(
						"softlayer_dns_domain_record.recordA", "record_data", "127.0.0.1"),
					resource.TestCheckResourceAttr(
						"softlayer_dns_domain_record.recordA", "record_type", "a"),
					resource.TestCheckResourceAttr(
						"softlayer_dns_domain_record.recordAAAA", "host", "hosta-2.com"),
					resource.TestCheckResourceAttr(
						"softlayer_dns_domain_record.recordAAAA", "record_data", "FE80:0000:0000:0000:0202:B3FF:FE1E:8329"),
					resource.TestCheckResourceAttr(
						"softlayer_dns_domain_record.recordAAAA", "record_type", "aaaa"),
					testAccCheckSoftLayerDnsDomainRecordsExists("softlayer_dns_domain.acceptance_test_dns_domain-1", 5),
				),
				Destroy: false,
			},
			resource.TestStep{
				Config: testAccCheckSoftLayerDnsDomainConfig_changed,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSoftLayerDnsDomainExists("softlayer_dns_domain.acceptance_test_dns_domain-1", &dns_domain),
					testAccCheckSoftLayerDnsDomainAttributes(&dns_domain),
					testAccCheckSoftLayerDnsDomainRecordDomainId("softlayer_dns_domain_record.recordA", &dns_domain),
					testAccCheckSoftLayerDnsDomainRecordDomainId("softlayer_dns_domain_record.recordAAAA", &dns_domain),
					resource.TestCheckResourceAttr(
						"softlayer_dns_domain.acceptance_test_dns_domain-1", "name", changed_dns_domain_name),
					resource.TestCheckResourceAttr(
						"softlayer_dns_domain_record.recordA", "host", "hosta.com"),
					resource.TestCheckResourceAttr(
						"softlayer_dns_domain_record.recordA", "record_data", "127.0.0.1"),
					resource.TestCheckResourceAttr(
						"softlayer_dns_domain_record.recordA", "record_type", "a"),
					resource.TestCheckResourceAttr(
						"softlayer_dns_domain_record.recordAAAA", "host", "hosta-2.com"),
					resource.TestCheckResourceAttr(
						"softlayer_dns_domain_record.recordAAAA", "record_data", "FE80:0000:0000:0000:0202:B3FF:FE1E:8329"),
					resource.TestCheckResourceAttr(
						"softlayer_dns_domain_record.recordAAAA", "record_type", "aaaa"),
					testAccCheckSoftLayerDnsDomainRecordsExists("softlayer_dns_domain.acceptance_test_dns_domain-1", 5),
					testAccCheckSoftLayerDnsDomainChanged(&dns_domain),
				),
				Destroy: false,
			},
		},
	})
}

func testAccCheckSoftLayerDnsDomainDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Client).dnsDomainService

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "softlayer_dns_domain" {
			continue
		}

		dnsId, _ := strconv.Atoi(rs.Primary.ID)

		// Try to find the domain
		_, err := client.GetObject(dnsId)

		if err != nil {
			return fmt.Errorf("Dns Domain with id %d does not exist", dnsId)
		}
	}

	return nil
}

func testAccCheckSoftLayerDnsDomainRecordDomainId(n string, dns_domain *datatypes.SoftLayer_Dns_Domain) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		id, _ := strconv.Atoi(rs.Primary.Attributes["domain_id"])
		if dns_domain.Id != id {
			return fmt.Errorf("Dns domain id (%d) and Dns domain record domain id (%d) should be equal", dns_domain.Id, id)
		}

		return nil
	}
}

func testAccCheckSoftLayerDnsDomainAttributes(dns *datatypes.SoftLayer_Dns_Domain) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if dns.Name == "" {
			return fmt.Errorf("Empty dns domain name")
		}

		if dns.Serial == 0 {
			return fmt.Errorf("Bad dns domain serial: %d", dns.Serial)
		}

		if dns.Id == 0 {
			return fmt.Errorf("Bad dns domain id: %d", dns.Id)
		}

		return nil
	}
}

func saveSoftLayerDnsDomainId(dns *datatypes.SoftLayer_Dns_Domain, id_holder *int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		*id_holder = dns.Id

		return nil
	}
}

func testAccCheckSoftLayerDnsDomainChanged(dns *datatypes.SoftLayer_Dns_Domain) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*Client).dnsDomainService

		response, _ := client.GetObject(firstDnsId)
		if response.Id == firstDnsId {
			return fmt.Errorf("Dns domain with id %d still exists", firstDnsId)
		}

		return nil
	}
}

func testAccCheckSoftLayerDnsDomainExists(n string, dns_domain *datatypes.SoftLayer_Dns_Domain) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		dns_id, _ := strconv.Atoi(rs.Primary.ID)

		client := testAccProvider.Meta().(*Client).dnsDomainService
		found_domain, err := client.GetObject(dns_id)

		if err != nil {
			return err
		}

		if strconv.Itoa(int(found_domain.Id)) != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}

		*dns_domain = found_domain

		return nil
	}
}

func testAccCheckSoftLayerDnsDomainRecordsExists(dn string, expected_record_count int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[dn]

		if !ok {
			return fmt.Errorf("Not found: %s", dn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		dns_id, _ := strconv.Atoi(rs.Primary.ID)

		client := testAccProvider.Meta().(*Client).dnsDomainService
		found_domain, err := client.GetObject(dns_id)

		if err != nil {
			return err
		}

		if found_domain.ResourceRecordCount != expected_record_count {
			return fmt.Errorf("Wrong record count:%d, expected:%d", found_domain.ResourceRecordCount, expected_record_count)
		}

		if strconv.Itoa(int(found_domain.Id)) != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}

		return nil
	}
}

var testAccCheckSoftLayerDnsDomainConfig_basic = fmt.Sprintf(`
resource "softlayer_dns_domain" "acceptance_test_dns_domain-1" {
	name = "%s"
}

resource "softlayer_dns_domain_record" "recordA" {
    record_data = "127.0.0.1"
    domain_id = "${softlayer_dns_domain.acceptance_test_dns_domain-1.id}"
    host = "hosta.com"
    contact_email = "user@softlaer.com"
    ttl = 900
    record_type = "a"
}

resource "softlayer_dns_domain_record" "recordAAAA" {
    record_data = "FE80:0000:0000:0000:0202:B3FF:FE1E:8329"
    domain_id = "${softlayer_dns_domain.acceptance_test_dns_domain-1.id}"
    host = "hosta-2.com"
    contact_email = "user2changed@softlaer.com"
    ttl = 1000
    record_type = "aaaa"
}
`, test_dns_domain_name)

var testAccCheckSoftLayerDnsDomainConfig_changed = fmt.Sprintf(`
resource "softlayer_dns_domain" "acceptance_test_dns_domain-1" {
	name = "%s"
}

resource "softlayer_dns_domain_record" "recordA" {
    record_data = "127.0.0.1"
    domain_id = "${softlayer_dns_domain.acceptance_test_dns_domain-1.id}"
    host = "hosta.com"
    contact_email = "user@softlaer.com"
    ttl = 900
    record_type = "a"
}

resource "softlayer_dns_domain_record" "recordAAAA" {
    record_data = "FE80:0000:0000:0000:0202:B3FF:FE1E:8329"
    domain_id = "${softlayer_dns_domain.acceptance_test_dns_domain-1.id}"
    host = "hosta-2.com"
    contact_email = "user2changed@softlaer.com"
    ttl = 1000
    record_type = "aaaa"
}
`, changed_dns_domain_name)

var test_dns_domain_name = "zxczcxzxc.com"
var changed_dns_domain_name = "vbnvnvbnv.com"
var firstDnsId = 0
