package softlayer

import (
	"fmt"
	"strconv"
	"testing"

	datatypes "github.com/TheWeatherCompany/softlayer-go/data_types"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccSoftLayerDnsDomainRecord_Basic(t *testing.T) {
	var dns_domain datatypes.SoftLayer_Dns_Domain
	var dns_domain_record datatypes.SoftLayer_Dns_Domain_ResourceRecord

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSoftLayerDnsDomainDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckSoftLayerDnsDomainRecordConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSoftLayerDnsDomainExists("softlayer_dns_domain.test_dns_domain_records", &dns_domain),
					testAccCheckSoftLayerDnsDomainRecordExists("softlayer_dns_domain_record.recordA", &dns_domain_record),
					resource.TestCheckResourceAttr("softlayer_dns_domain_record.recordA", "record_data", "127.0.0.1"),
					resource.TestCheckResourceAttr("softlayer_dns_domain_record.recordA", "expire", "900"),
					resource.TestCheckResourceAttr("softlayer_dns_domain_record.recordA", "minimum_ttl", "90"),
					resource.TestCheckResourceAttr("softlayer_dns_domain_record.recordA", "mx_priority", "1"),
					resource.TestCheckResourceAttr("softlayer_dns_domain_record.recordA", "refresh", "1"),
					resource.TestCheckResourceAttr("softlayer_dns_domain_record.recordA", "host", "hosta.com"),
					resource.TestCheckResourceAttr("softlayer_dns_domain_record.recordA", "contact_email", "user@softlaer.com"),
					resource.TestCheckResourceAttr("softlayer_dns_domain_record.recordA", "ttl", "900"),
					resource.TestCheckResourceAttr("softlayer_dns_domain_record.recordA", "retry", "1"),
					resource.TestCheckResourceAttr("softlayer_dns_domain_record.recordA", "record_type", "a"),
				),
			},
		},
	})
}

func TestAccSoftLayerDnsDomainRecord_Types(t *testing.T) {
	var dns_domain datatypes.SoftLayer_Dns_Domain
	var dns_domain_record datatypes.SoftLayer_Dns_Domain_ResourceRecord

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSoftLayerDnsDomainDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckSoftLayerDnsDomainRecordConfig_all_types,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSoftLayerDnsDomainExists("softlayer_dns_domain.test_dns_domain_record_types", &dns_domain),
					testAccCheckSoftLayerDnsDomainRecordExists("softlayer_dns_domain_record.recordA", &dns_domain_record),
					testAccCheckSoftLayerDnsDomainRecordExists("softlayer_dns_domain_record.recordAAAA", &dns_domain_record),
					testAccCheckSoftLayerDnsDomainRecordExists("softlayer_dns_domain_record.recordCNAME", &dns_domain_record),
					testAccCheckSoftLayerDnsDomainRecordExists("softlayer_dns_domain_record.recordMX", &dns_domain_record),
					testAccCheckSoftLayerDnsDomainRecordExists("softlayer_dns_domain_record.recordNS", &dns_domain_record),
					testAccCheckSoftLayerDnsDomainRecordExists("softlayer_dns_domain_record.recordSPF", &dns_domain_record),
					testAccCheckSoftLayerDnsDomainRecordExists("softlayer_dns_domain_record.recordTXT", &dns_domain_record),
					testAccCheckSoftLayerDnsDomainRecordExists("softlayer_dns_domain_record.recordSRV", &dns_domain_record),
				),
				Destroy: false,
			},
		},
	})
}

func testAccCheckSoftLayerDnsDomainRecordExists(n string, dns_domain_record *datatypes.SoftLayer_Dns_Domain_ResourceRecord) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		dns_id, _ := strconv.Atoi(rs.Primary.ID)

		client := testAccProvider.Meta().(*Client).dnsDomainResourceRecordService
		found_domain_record, err := client.GetObject(dns_id)

		if err != nil {
			return err
		}

		if strconv.Itoa(int(found_domain_record.Id)) != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}

		*dns_domain_record = found_domain_record

		return nil
	}
}

var testAccCheckSoftLayerDnsDomainRecordConfig_basic = `
resource "softlayer_dns_domain" "test_dns_domain_records" {
	name = "domain.records.com"
}

resource "softlayer_dns_domain_record" "recordA" {
    record_data = "127.0.0.1"
    domain_id = "${softlayer_dns_domain.test_dns_domain_records.id}"
    expire = 900
    minimum_ttl = 90
    mx_priority = 1
    refresh = 1
    host = "hosta.com"
    contact_email = "user@softlaer.com"
    ttl = 900
    retry = 1
    record_type = "a"
}
`
var testAccCheckSoftLayerDnsDomainRecordConfig_all_types = `
resource "softlayer_dns_domain" "test_dns_domain_record_types" {
	name = "domaint.record.types.com"
}

resource "softlayer_dns_domain_record" "recordA" {
    record_data = "127.0.0.1"
    domain_id = "${softlayer_dns_domain.test_dns_domain_record_types.id}"
    host = "hosta.com"
    contact_email = "user@softlaer.com"
    ttl = 900
    record_type = "a"
}

resource "softlayer_dns_domain_record" "recordAAAA" {
    record_data = "FE80:0000:0000:0000:0202:B3FF:FE1E:8329"
    domain_id = "${softlayer_dns_domain.test_dns_domain_record_types.id}"
    host = "hosta-2.com"
    contact_email = "user2changed@softlaer.com"
    ttl = 1000
    record_type = "aaaa"
}

resource "softlayer_dns_domain_record" "recordCNAME" {
    record_data = "testsssaaaass.com"
    domain_id = "${softlayer_dns_domain.test_dns_domain_record_types.id}"
    host = "hosta-cname.com"
    contact_email = "user@softlaer.com"
    ttl = 900
    record_type = "cname"
}

resource "softlayer_dns_domain_record" "recordMX" {
    record_data = "email.example.com"
    domain_id = "${softlayer_dns_domain.test_dns_domain_record_types.id}"
    host = "hosta-mx.com"
    contact_email = "user@softlaer.com"
    ttl = 900
    record_type = "mx"
}

resource "softlayer_dns_domain_record" "recordNS" {
    record_data = "ns1.example.org"
    domain_id = "${softlayer_dns_domain.test_dns_domain_record_types.id}"
    host = "hosta-ns.com"
    contact_email = "user@softlaer.com"
    ttl = 900
    record_type = "ns"
}

resource "softlayer_dns_domain_record" "recordSPF" {
    record_data = "v=spf1 mx:mail.example.org ~all"
    domain_id = "${softlayer_dns_domain.test_dns_domain_record_types.id}"
    host = "hosta-spf"
    contact_email = "user@softlaer.com"
    ttl = 900
    record_type = "spf"
}

resource "softlayer_dns_domain_record" "recordTXT" {
    record_data = "127.0.0.1"
    domain_id = "${softlayer_dns_domain.test_dns_domain_record_types.id}"
    host = "hosta-txt.com"
    contact_email = "user@softlaer.com"
    ttl = 900
    record_type = "txt"
}

resource "softlayer_dns_domain_record" "recordSRV" {
    record_data = "ns1.example.org"
    domain_id = "${softlayer_dns_domain.test_dns_domain_record_types.id}"
    host = "hosta-srv.com"
    contact_email = "user@softlaer.com"
    ttl = 900
    record_type = "srv"
	port = 8080
	priority = 3
	protocol = "_tcp"
	weight = 3
	service = "_mail"
}
`
