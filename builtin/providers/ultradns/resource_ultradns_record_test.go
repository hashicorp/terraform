package ultradns

import (
	"fmt"
	"testing"

	"github.com/Ensighten/udnssdk"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccUltradnsRecord(t *testing.T) {
	var record udnssdk.RRSet
	// domain := os.Getenv("ULTRADNS_DOMAIN")
	domain := "ultradns.phinze.com"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccRecordCheckDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testCfgRecordMinimal, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUltradnsRecordExists("ultradns_record.it", &record),
					resource.TestCheckResourceAttr("ultradns_record.it", "zone", domain),
					resource.TestCheckResourceAttr("ultradns_record.it", "name", "test-record"),
					resource.TestCheckResourceAttr("ultradns_record.it", "rdata.3994963683", "10.5.0.1"),
				),
			},
			resource.TestStep{
				Config: fmt.Sprintf(testCfgRecordMinimal, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUltradnsRecordExists("ultradns_record.it", &record),
					resource.TestCheckResourceAttr("ultradns_record.it", "zone", domain),
					resource.TestCheckResourceAttr("ultradns_record.it", "name", "test-record"),
					resource.TestCheckResourceAttr("ultradns_record.it", "rdata.3994963683", "10.5.0.1"),
				),
			},
			resource.TestStep{
				Config: fmt.Sprintf(testCfgRecordUpdated, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUltradnsRecordExists("ultradns_record.it", &record),
					resource.TestCheckResourceAttr("ultradns_record.it", "zone", domain),
					resource.TestCheckResourceAttr("ultradns_record.it", "name", "test-record"),
					resource.TestCheckResourceAttr("ultradns_record.it", "rdata.1998004057", "10.5.0.2"),
				),
			},
		},
	})
}

func testAccRecordCheckDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*udnssdk.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "ultradns_record" {
			continue
		}

		k := udnssdk.RRSetKey{
			Zone: rs.Primary.Attributes["zone"],
			Name: rs.Primary.Attributes["name"],
			Type: rs.Primary.Attributes["type"],
		}

		_, err := client.RRSets.Select(k)

		if err == nil {
			return fmt.Errorf("Record still exists")
		}
	}

	return nil
}

const testCfgRecordMinimal = `
resource "ultradns_record" "it" {
  zone = "%s"
  name  = "test-record"

  rdata = ["10.5.0.1"]
  type  = "A"
  ttl   = 3600
}
`

const testCfgRecordUpdated = `
resource "ultradns_record" "it" {
  zone = "%s"
  name  = "test-record"

  rdata = ["10.5.0.2"]
  type  = "A"
  ttl   = 3600
}
`
