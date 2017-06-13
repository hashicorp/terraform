package coredns

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"k8s.io/kubernetes/federation/pkg/dnsprovider"
)

func TestAccCorednsRecord(t *testing.T) {
	var record dnsprovider.ResourceRecordSet
	fqdn := "coredns.skydns.local"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccRecordCheckDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testCfgRecordMinimal, fqdn),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCorednsRecordExists("coredns_record.it", &record),
					resource.TestCheckResourceAttr("coredns_record.it", "fqdn", fqdn),
					resource.TestCheckResourceAttr("coredns_record.it", "rdata.3258735021", "10.10.10.10"),
				),
			},
			resource.TestStep{
				Config: fmt.Sprintf(testCfgRecordUpdated, fqdn),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCorednsRecordExists("coredns_record.it", &record),
					resource.TestCheckResourceAttr("coredns_record.it", "fqdn", fqdn),
					resource.TestCheckResourceAttr("coredns_record.it", "rdata.3910208110", "10.10.10.20"),
				),
			},
		},
	})
}
func testAccCheckCorednsRecordExists(n string, record *dnsprovider.ResourceRecordSet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		dns := testAccProvider.Meta().(*dnsOp)
		k := recordKey{
			FQDN:       rs.Primary.Attributes["fqdn"],
			RecordType: rs.Primary.Attributes["type"],
		}

		foundRecord, err := dns.getRecord(k)

		if err != nil {
			return err
		}

		if foundRecord[0].Name() != rs.Primary.Attributes["hostname"] {
			return fmt.Errorf("Record not found: %+v,\n %+v\n", foundRecord, rs.Primary.Attributes)
		}

		*record = foundRecord[0]

		return nil
	}
}

func testAccRecordCheckDestroy(s *terraform.State) error {
	dns := testAccProvider.Meta().(*dnsOp)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "coredns_record" {
			continue
		}

		k := recordKey{
			FQDN:       rs.Primary.Attributes["fqdn"],
			RecordType: rs.Primary.Attributes["type"],
		}

		_, err := dns.getRecord(k)

		if err != nil {
			return err
		}
	}

	return nil
}

const testCfgRecordMinimal = `
resource "coredns_record" "it" {
  fqdn = "%s"
  type  = "A"
  rdata = ["10.10.10.10"]
  ttl   = 60
}
`

const testCfgRecordUpdated = `
resource "coredns_record" "it" {
  fqdn  = "%s"
  type  = "A"
  rdata = ["10.10.10.20"]
  ttl   = 60
}
`
