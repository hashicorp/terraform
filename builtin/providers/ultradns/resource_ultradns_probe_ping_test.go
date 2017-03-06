package ultradns

import (
	"fmt"
	"testing"

	"github.com/Ensighten/udnssdk"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccUltradnsProbePing(t *testing.T) {
	var record udnssdk.RRSet
	domain := "ultradns.phinze.com"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccTcpoolCheckDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testCfgProbePingRecord, domain, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUltradnsRecordExists("ultradns_tcpool.test-probe-ping-record", &record),
					// Specified
					resource.TestCheckResourceAttr("ultradns_probe_ping.it", "zone", domain),
					resource.TestCheckResourceAttr("ultradns_probe_ping.it", "name", "test-probe-ping-record"),
					resource.TestCheckResourceAttr("ultradns_probe_ping.it", "pool_record", "10.3.0.1"),
					resource.TestCheckResourceAttr("ultradns_probe_ping.it", "agents.0", "DALLAS"),
					resource.TestCheckResourceAttr("ultradns_probe_ping.it", "agents.1", "AMSTERDAM"),
					resource.TestCheckResourceAttr("ultradns_probe_ping.it", "interval", "ONE_MINUTE"),
					resource.TestCheckResourceAttr("ultradns_probe_ping.it", "threshold", "2"),
					resource.TestCheckResourceAttr("ultradns_probe_ping.it", "ping_probe.0.packets", "15"),
					resource.TestCheckResourceAttr("ultradns_probe_ping.it", "ping_probe.0.packet_size", "56"),
					resource.TestCheckResourceAttr("ultradns_probe_ping.it", "ping_probe.0.limit.#", "2"),

					// hashLimits(): lossPercent -> 3375621462
					resource.TestCheckResourceAttr("ultradns_probe_ping.it", "ping_probe.0.limit.3375621462.name", "lossPercent"),
					resource.TestCheckResourceAttr("ultradns_probe_ping.it", "ping_probe.0.limit.3375621462.warning", "1"),
					resource.TestCheckResourceAttr("ultradns_probe_ping.it", "ping_probe.0.limit.3375621462.critical", "2"),
					resource.TestCheckResourceAttr("ultradns_probe_ping.it", "ping_probe.0.limit.3375621462.fail", "3"),

					// hashLimits(): total -> 3257917790
					resource.TestCheckResourceAttr("ultradns_probe_ping.it", "ping_probe.0.limit.3257917790.name", "total"),
					resource.TestCheckResourceAttr("ultradns_probe_ping.it", "ping_probe.0.limit.3257917790.warning", "2"),
					resource.TestCheckResourceAttr("ultradns_probe_ping.it", "ping_probe.0.limit.3257917790.critical", "3"),
					resource.TestCheckResourceAttr("ultradns_probe_ping.it", "ping_probe.0.limit.3257917790.fail", "4"),
				),
			},
			resource.TestStep{
				Config: fmt.Sprintf(testCfgProbePingPool, domain, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUltradnsRecordExists("ultradns_tcpool.test-probe-ping-pool", &record),
					// Specified
					resource.TestCheckResourceAttr("ultradns_probe_ping.it", "zone", domain),
					resource.TestCheckResourceAttr("ultradns_probe_ping.it", "name", "test-probe-ping-pool"),
					resource.TestCheckResourceAttr("ultradns_probe_ping.it", "agents.0", "DALLAS"),
					resource.TestCheckResourceAttr("ultradns_probe_ping.it", "agents.1", "AMSTERDAM"),
					resource.TestCheckResourceAttr("ultradns_probe_ping.it", "interval", "ONE_MINUTE"),
					resource.TestCheckResourceAttr("ultradns_probe_ping.it", "threshold", "2"),
					resource.TestCheckResourceAttr("ultradns_probe_ping.it", "ping_probe.0.packets", "15"),
					resource.TestCheckResourceAttr("ultradns_probe_ping.it", "ping_probe.0.packet_size", "56"),
					resource.TestCheckResourceAttr("ultradns_probe_ping.it", "ping_probe.0.limit.#", "2"),

					// hashLimits(): lossPercent -> 3375621462
					resource.TestCheckResourceAttr("ultradns_probe_ping.it", "ping_probe.0.limit.3375621462.name", "lossPercent"),
					resource.TestCheckResourceAttr("ultradns_probe_ping.it", "ping_probe.0.limit.3375621462.warning", "1"),
					resource.TestCheckResourceAttr("ultradns_probe_ping.it", "ping_probe.0.limit.3375621462.critical", "2"),
					resource.TestCheckResourceAttr("ultradns_probe_ping.it", "ping_probe.0.limit.3375621462.fail", "3"),

					// hashLimits(): total -> 3257917790
					resource.TestCheckResourceAttr("ultradns_probe_ping.it", "ping_probe.0.limit.3257917790.name", "total"),
					resource.TestCheckResourceAttr("ultradns_probe_ping.it", "ping_probe.0.limit.3257917790.warning", "2"),
					resource.TestCheckResourceAttr("ultradns_probe_ping.it", "ping_probe.0.limit.3257917790.critical", "3"),
					resource.TestCheckResourceAttr("ultradns_probe_ping.it", "ping_probe.0.limit.3257917790.fail", "4"),
				),
			},
		},
	})
}

const testCfgProbePingRecord = `
resource "ultradns_tcpool" "test-probe-ping-record" {
  zone  = "%s"
  name  = "test-probe-ping-record"

  ttl   = 30
  description = "traffic controller pool with probes"

  run_probes    = true
  act_on_probes = true
  max_to_lb     = 2

  rdata {
    host = "10.3.0.1"

    state          = "NORMAL"
    run_probes     = true
    priority       = 1
    failover_delay = 0
    threshold      = 1
    weight         = 2
  }

  rdata {
    host = "10.3.0.2"

    state          = "NORMAL"
    run_probes     = true
    priority       = 2
    failover_delay = 0
    threshold      = 1
    weight         = 2
  }

  backup_record_rdata = "10.3.0.3"
}

resource "ultradns_probe_ping" "it" {
  zone  = "%s"
  name  = "test-probe-ping-record"

  pool_record = "10.3.0.1"

  agents = ["DALLAS", "AMSTERDAM"]

  interval  = "ONE_MINUTE"
  threshold = 2

  ping_probe {
    packets    = 15
    packet_size = 56

    limit {
      name     = "lossPercent"
      warning  = 1
      critical = 2
      fail     = 3
    }

    limit {
      name     = "total"
      warning  = 2
      critical = 3
      fail     = 4
    }
  }

  depends_on = ["ultradns_tcpool.test-probe-ping-record"]
}
`

const testCfgProbePingPool = `
resource "ultradns_tcpool" "test-probe-ping-pool" {
  zone  = "%s"
  name  = "test-probe-ping-pool"

  ttl   = 30
  description = "traffic controller pool with probes"

  run_probes    = true
  act_on_probes = true
  max_to_lb     = 2

  rdata {
    host = "10.3.0.1"

    state          = "NORMAL"
    run_probes     = true
    priority       = 1
    failover_delay = 0
    threshold      = 1
    weight         = 2
  }

  rdata {
    host = "10.3.0.2"

    state          = "NORMAL"
    run_probes     = true
    priority       = 2
    failover_delay = 0
    threshold      = 1
    weight         = 2
  }

  backup_record_rdata = "10.3.0.3"
}

resource "ultradns_probe_ping" "it" {
  zone  = "%s"
  name  = "test-probe-ping-pool"

  agents = ["DALLAS", "AMSTERDAM"]

  interval  = "ONE_MINUTE"
  threshold = 2

  ping_probe {
    packets    = 15
    packet_size = 56

    limit {
      name     = "lossPercent"
      warning  = 1
      critical = 2
      fail     = 3
    }

    limit {
      name     = "total"
      warning  = 2
      critical = 3
      fail     = 4
    }
  }

  depends_on = ["ultradns_tcpool.test-probe-ping-pool"]
}
`
