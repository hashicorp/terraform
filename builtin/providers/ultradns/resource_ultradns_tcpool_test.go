package ultradns

import (
	"fmt"
	"testing"

	"github.com/Ensighten/udnssdk"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccUltradnsTcpool(t *testing.T) {
	var record udnssdk.RRSet
	domain := "ultradns.phinze.com"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckUltradnsTcpoolDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckUltraDNSRecordTcpoolMinimal, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUltraDNSRecordExists("ultradns_tcpool.minimal", &record),
					// Specified
					resource.TestCheckResourceAttr("ultradns_tcpool.minimal", "name", "tcpool-minimal"),
					resource.TestCheckResourceAttr("ultradns_tcpool.minimal", "zone", domain),
					resource.TestCheckResourceAttr("ultradns_tcpool.minimal", "ttl", "300"),
					resource.TestCheckResourceAttr("ultradns_tcpool.minimal", "rdata.0.host", "192.168.0.10"),
					// Defaults
					resource.TestCheckResourceAttr("ultradns_tcpool.minimal", "act_on_probes", "true"),
					resource.TestCheckResourceAttr("ultradns_tcpool.minimal", "description", "Minimal TC Pool"),
					resource.TestCheckResourceAttr("ultradns_tcpool.minimal", "max_to_lb", "0"),
					resource.TestCheckResourceAttr("ultradns_tcpool.minimal", "run_probes", "true"),
					resource.TestCheckResourceAttr("ultradns_tcpool.minimal", "rdata.0.failover_delay", "0"),
					resource.TestCheckResourceAttr("ultradns_tcpool.minimal", "rdata.0.priority", "1"),
					resource.TestCheckResourceAttr("ultradns_tcpool.minimal", "rdata.0.run_probes", "true"),
					resource.TestCheckResourceAttr("ultradns_tcpool.minimal", "rdata.0.state", "NORMAL"),
					resource.TestCheckResourceAttr("ultradns_tcpool.minimal", "rdata.0.threshold", "1"),
					resource.TestCheckResourceAttr("ultradns_tcpool.minimal", "rdata.0.weight", "2"),
					// Generated
					resource.TestCheckResourceAttr("ultradns_tcpool.minimal", "id", "tcpool-minimal.ultradns.phinze.com"),
					resource.TestCheckResourceAttr("ultradns_tcpool.minimal", "hostname", "tcpool-minimal.ultradns.phinze.com."),
				),
			},
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckUltraDNSRecordTcpoolMaximal, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUltraDNSRecordExists("ultradns_tcpool.maximal", &record),
					// Specified
					resource.TestCheckResourceAttr("ultradns_tcpool.maximal", "name", "tcpool-maximal"),
					resource.TestCheckResourceAttr("ultradns_tcpool.maximal", "zone", domain),
					resource.TestCheckResourceAttr("ultradns_tcpool.maximal", "ttl", "300"),
					resource.TestCheckResourceAttr("ultradns_tcpool.maximal", "description", "traffic controller pool with all settings tuned"),

					resource.TestCheckResourceAttr("ultradns_tcpool.maximal", "act_on_probes", "false"),
					resource.TestCheckResourceAttr("ultradns_tcpool.maximal", "max_to_lb", "2"),
					resource.TestCheckResourceAttr("ultradns_tcpool.maximal", "run_probes", "false"),

					resource.TestCheckResourceAttr("ultradns_tcpool.maximal", "rdata.0.host", "192.168.0.10"),
					resource.TestCheckResourceAttr("ultradns_tcpool.maximal", "rdata.0.failover_delay", "30"),
					resource.TestCheckResourceAttr("ultradns_tcpool.maximal", "rdata.0.priority", "1"),
					resource.TestCheckResourceAttr("ultradns_tcpool.maximal", "rdata.0.run_probes", "true"),
					resource.TestCheckResourceAttr("ultradns_tcpool.maximal", "rdata.0.state", "ACTIVE"),
					resource.TestCheckResourceAttr("ultradns_tcpool.maximal", "rdata.0.threshold", "1"),
					resource.TestCheckResourceAttr("ultradns_tcpool.maximal", "rdata.0.weight", "2"),

					resource.TestCheckResourceAttr("ultradns_tcpool.maximal", "rdata.1.host", "192.168.0.11"),
					resource.TestCheckResourceAttr("ultradns_tcpool.maximal", "rdata.1.failover_delay", "30"),
					resource.TestCheckResourceAttr("ultradns_tcpool.maximal", "rdata.1.priority", "2"),
					resource.TestCheckResourceAttr("ultradns_tcpool.maximal", "rdata.1.run_probes", "true"),
					resource.TestCheckResourceAttr("ultradns_tcpool.maximal", "rdata.1.state", "INACTIVE"),
					resource.TestCheckResourceAttr("ultradns_tcpool.maximal", "rdata.1.threshold", "1"),
					resource.TestCheckResourceAttr("ultradns_tcpool.maximal", "rdata.1.weight", "4"),

					resource.TestCheckResourceAttr("ultradns_tcpool.maximal", "rdata.2.host", "192.168.0.12"),
					resource.TestCheckResourceAttr("ultradns_tcpool.maximal", "rdata.2.failover_delay", "30"),
					resource.TestCheckResourceAttr("ultradns_tcpool.maximal", "rdata.2.priority", "3"),
					resource.TestCheckResourceAttr("ultradns_tcpool.maximal", "rdata.2.run_probes", "false"),
					resource.TestCheckResourceAttr("ultradns_tcpool.maximal", "rdata.2.state", "NORMAL"),
					resource.TestCheckResourceAttr("ultradns_tcpool.maximal", "rdata.2.threshold", "1"),
					resource.TestCheckResourceAttr("ultradns_tcpool.maximal", "rdata.2.weight", "8"),
					// Generated
					resource.TestCheckResourceAttr("ultradns_tcpool.maximal", "id", "tcpool-maximal.ultradns.phinze.com"),
					resource.TestCheckResourceAttr("ultradns_tcpool.maximal", "hostname", "tcpool-maximal.ultradns.phinze.com."),
				),
			},
		},
	})
}

func testAccCheckUltradnsTcpoolDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*udnssdk.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "ultradns_tcpool" {
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

const testAccCheckUltraDNSRecordTcpoolMinimal = `
resource "ultradns_tcpool" "minimal" {
  zone        = "%s"
  name        = "tcpool-minimal"
  ttl         = 300
  description = "Minimal TC Pool"

  rdata {
    host = "192.168.0.10"
  }
}
`

const testAccCheckUltraDNSRecordTcpoolMaximal = `
resource "ultradns_tcpool" "maximal" {
  zone        = "%s"
  name        = "tcpool-maximal"
  ttl         = 300
  description = "traffic controller pool with all settings tuned"

  act_on_probes = false
  max_to_lb     = 2
  run_probes    = false

  rdata {
    host           = "192.168.0.10"
    failover_delay = 30
    priority       = 1
    run_probes     = true
    state          = "ACTIVE"
    threshold      = 1
    weight         = 2
  }

  rdata {
    host           = "192.168.0.11"
    failover_delay = 30
    priority       = 2
    run_probes     = true
    state          = "INACTIVE"
    threshold      = 1
    weight         = 4
  }

  rdata {
    host           = "192.168.0.12"
    failover_delay = 30
    priority       = 3
    run_probes     = false
    state          = "NORMAL"
    threshold      = 1
    weight         = 8
  }

  backup_record_rdata          = "192.168.0.11"
  backup_record_failover_delay = 30
}
`
