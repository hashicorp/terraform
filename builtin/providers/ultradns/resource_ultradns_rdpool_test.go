package ultradns

import (
	"fmt"
	"testing"

	"github.com/Ensighten/udnssdk"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccUltradnsRdpool(t *testing.T) {
	var record udnssdk.RRSet
	domain := "ultradns.phinze.com"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccRdpoolCheckDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testCfgRdpoolMinimal, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUltradnsRecordExists("ultradns_rdpool.it", &record),
					// Specified
					resource.TestCheckResourceAttr("ultradns_rdpool.it", "zone", domain),
					resource.TestCheckResourceAttr("ultradns_rdpool.it", "name", "test-rdpool-minimal"),
					resource.TestCheckResourceAttr("ultradns_rdpool.it", "ttl", "300"),

					// hashRdatas(): 10.6.0.1 -> 2847814707
					resource.TestCheckResourceAttr("ultradns_rdpool.it", "rdata.2847814707.host", "10.6.0.1"),
					// Defaults
					resource.TestCheckResourceAttr("ultradns_rdpool.it", "description", "Minimal RD Pool"),
					resource.TestCheckResourceAttr("ultradns_rdpool.it", "rdata.2847814707.priority", "1"),
					// Generated
					resource.TestCheckResourceAttr("ultradns_rdpool.it", "id", "test-rdpool-minimal.ultradns.phinze.com"),
					resource.TestCheckResourceAttr("ultradns_rdpool.it", "hostname", "test-rdpool-minimal.ultradns.phinze.com."),
				),
			},
			resource.TestStep{
				Config: fmt.Sprintf(testCfgRdpoolMaximal, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUltradnsRecordExists("ultradns_rdpool.it", &record),
					// Specified
					resource.TestCheckResourceAttr("ultradns_rdpool.it", "zone", domain),
					resource.TestCheckResourceAttr("ultradns_rdpool.it", "name", "test-rdpool-maximal"),
					resource.TestCheckResourceAttr("ultradns_rdpool.it", "ttl", "300"),
					resource.TestCheckResourceAttr("ultradns_rdpool.it", "description", "traffic controller pool with all settings tuned"),

					resource.TestCheckResourceAttr("ultradns_rdpool.it", "act_on_probes", "false"),
					resource.TestCheckResourceAttr("ultradns_rdpool.it", "max_to_lb", "2"),
					resource.TestCheckResourceAttr("ultradns_rdpool.it", "run_probes", "false"),

					// hashRdatas(): 10.6.1.1 -> 2826722820
					resource.TestCheckResourceAttr("ultradns_rdpool.it", "rdata.2826722820.host", "10.6.1.1"),
					resource.TestCheckResourceAttr("ultradns_rdpool.it", "rdata.2826722820.priority", "1"),

					// hashRdatas(): 10.6.1.2 -> 829755326
					resource.TestCheckResourceAttr("ultradns_rdpool.it", "rdata.829755326.host", "10.6.1.2"),
					resource.TestCheckResourceAttr("ultradns_rdpool.it", "rdata.829755326.priority", "2"),

					// Generated
					resource.TestCheckResourceAttr("ultradns_rdpool.it", "id", "test-rdpool-maximal.ultradns.phinze.com"),
					resource.TestCheckResourceAttr("ultradns_rdpool.it", "hostname", "test-rdpool-maximal.ultradns.phinze.com."),
				),
			},
		},
	})
}

const testCfgRdpoolMinimal = `
resource "ultradns_rdpool" "it" {
  zone        = "%s"
  name        = "test-rdpool-minimal"
  ttl         = 300
  description = "Minimal RD Pool"

  rdata {
    host = "10.6.0.1"
  }
}
`

const testCfgRdpoolMaximal = `
resource "ultradns_rdpool" "it" {
  zone        = "%s"
  name        = "test-rdpool-maximal"
  order       = "ROUND_ROBIN"
  ttl         = 300
  description = "traffic controller pool with all settings tuned"
  rdata {
    host = "10.6.1.1"
    priority       = 1
  }

  rdata {
    host = "10.6.1.2"
    priority       = 2
  }
}
`
