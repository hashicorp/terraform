package ultradns

import (
	"fmt"
	"testing"

	"github.com/Ensighten/udnssdk"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccUltradnsProbeHTTP(t *testing.T) {
	var record udnssdk.RRSet
	domain := "ultradns.phinze.com"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccTcpoolCheckDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testCfgProbeHTTPMinimal, domain, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUltradnsRecordExists("ultradns_tcpool.test-probe-http-minimal", &record),
					// Specified
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "zone", domain),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "name", "test-probe-http-minimal"),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "pool_record", "10.2.0.1"),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "agents.4091180299", "DALLAS"),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "agents.2144410488", "AMSTERDAM"),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "interval", "ONE_MINUTE"),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "threshold", "2"),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "http_probe.0.transaction.0.method", "GET"),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "http_probe.0.transaction.0.url", "http://localhost/index"),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "http_probe.0.transaction.0.limit.#", "2"),

					resource.TestCheckResourceAttr("ultradns_probe_http.it", "http_probe.0.transaction.0.limit.1959786783.name", "connect"),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "http_probe.0.transaction.0.limit.1959786783.warning", "20"),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "http_probe.0.transaction.0.limit.1959786783.critical", "20"),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "http_probe.0.transaction.0.limit.1959786783.fail", "20"),

					resource.TestCheckResourceAttr("ultradns_probe_http.it", "http_probe.0.transaction.0.limit.1349952704.name", "run"),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "http_probe.0.transaction.0.limit.1349952704.warning", "60"),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "http_probe.0.transaction.0.limit.1349952704.critical", "60"),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "http_probe.0.transaction.0.limit.1349952704.fail", "60"),
				),
			},
			resource.TestStep{
				Config: fmt.Sprintf(testCfgProbeHTTPMaximal, domain, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUltradnsRecordExists("ultradns_tcpool.test-probe-http-maximal", &record),
					// Specified
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "zone", domain),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "name", "test-probe-http-maximal"),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "pool_record", "10.2.1.1"),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "agents.4091180299", "DALLAS"),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "agents.2144410488", "AMSTERDAM"),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "interval", "ONE_MINUTE"),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "threshold", "2"),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "http_probe.0.transaction.0.method", "POST"),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "http_probe.0.transaction.0.url", "http://localhost/index"),

					resource.TestCheckResourceAttr("ultradns_probe_http.it", "http_probe.0.transaction.0.limit.#", "4"),

					resource.TestCheckResourceAttr("ultradns_probe_http.it", "http_probe.0.transaction.0.limit.1349952704.name", "run"),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "http_probe.0.transaction.0.limit.1349952704.warning", "1"),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "http_probe.0.transaction.0.limit.1349952704.critical", "2"),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "http_probe.0.transaction.0.limit.1349952704.fail", "3"),

					resource.TestCheckResourceAttr("ultradns_probe_http.it", "http_probe.0.transaction.0.limit.2720402232.name", "avgConnect"),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "http_probe.0.transaction.0.limit.2720402232.warning", "4"),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "http_probe.0.transaction.0.limit.2720402232.critical", "5"),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "http_probe.0.transaction.0.limit.2720402232.fail", "6"),

					resource.TestCheckResourceAttr("ultradns_probe_http.it", "http_probe.0.transaction.0.limit.896769211.name", "avgRun"),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "http_probe.0.transaction.0.limit.896769211.warning", "7"),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "http_probe.0.transaction.0.limit.896769211.critical", "8"),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "http_probe.0.transaction.0.limit.896769211.fail", "9"),

					resource.TestCheckResourceAttr("ultradns_probe_http.it", "http_probe.0.transaction.0.limit.1959786783.name", "connect"),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "http_probe.0.transaction.0.limit.1959786783.warning", "10"),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "http_probe.0.transaction.0.limit.1959786783.critical", "11"),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "http_probe.0.transaction.0.limit.1959786783.fail", "12"),

					resource.TestCheckResourceAttr("ultradns_probe_http.it", "http_probe.0.total_limits.0.warning", "13"),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "http_probe.0.total_limits.0.critical", "14"),
					resource.TestCheckResourceAttr("ultradns_probe_http.it", "http_probe.0.total_limits.0.fail", "15"),
				),
			},
		},
	})
}

const testCfgProbeHTTPMinimal = `
resource "ultradns_tcpool" "test-probe-http-minimal" {
  zone = "%s"
  name = "test-probe-http-minimal"

  ttl         = 30
  description = "traffic controller pool with probes"

  run_probes    = true
  act_on_probes = true
  max_to_lb     = 2

  rdata {
    host = "10.2.0.1"

    state          = "NORMAL"
    run_probes     = true
    priority       = 1
    failover_delay = 0
    threshold      = 1
    weight         = 2
  }

  rdata {
    host = "10.2.0.2"

    state          = "NORMAL"
    run_probes     = true
    priority       = 2
    failover_delay = 0
    threshold      = 1
    weight         = 2
  }

  backup_record_rdata = "10.2.0.3"
}

resource "ultradns_probe_http" "it" {
  zone = "%s"
  name = "test-probe-http-minimal"

  pool_record = "10.2.0.1"

  agents = ["DALLAS", "AMSTERDAM"]

  interval  = "ONE_MINUTE"
  threshold = 2

  http_probe {
    transaction {
      method = "GET"
      url    = "http://localhost/index"

      limit {
        name     = "run"
        warning  = 60
        critical = 60
        fail     = 60
      }

      limit {
        name     = "connect"
        warning  = 20
        critical = 20
        fail     = 20
      }
    }
  }

  depends_on = ["ultradns_tcpool.test-probe-http-minimal"]
}
`

const testCfgProbeHTTPMaximal = `
resource "ultradns_tcpool" "test-probe-http-maximal" {
  zone  = "%s"
  name  = "test-probe-http-maximal"

  ttl   = 30
  description = "traffic controller pool with probes"

  run_probes    = true
  act_on_probes = true
  max_to_lb     = 2

  rdata {
    host = "10.2.1.1"

    state          = "NORMAL"
    run_probes     = true
    priority       = 1
    failover_delay = 0
    threshold      = 1
    weight         = 2
  }

  rdata {
    host = "10.2.1.2"

    state          = "NORMAL"
    run_probes     = true
    priority       = 2
    failover_delay = 0
    threshold      = 1
    weight         = 2
  }

  backup_record_rdata = "10.2.1.3"
}

resource "ultradns_probe_http" "it" {
  zone = "%s"
  name = "test-probe-http-maximal"

  pool_record = "10.2.1.1"

  agents = ["DALLAS", "AMSTERDAM"]

  interval  = "ONE_MINUTE"
  threshold = 2

  http_probe {
    transaction {
      method           = "POST"
      url              = "http://localhost/index"
      transmitted_data = "{}"
      follow_redirects = true

      limit {
        name = "run"

        warning  = 1
        critical = 2
        fail     = 3
      }
      limit {
        name = "avgConnect"

        warning  = 4
        critical = 5
        fail     = 6
      }
      limit {
        name = "avgRun"

        warning  = 7
        critical = 8
        fail     = 9
      }
      limit {
        name = "connect"

        warning  = 10
        critical = 11
        fail     = 12
      }
    }

    total_limits {
      warning  = 13
      critical = 14
      fail     = 15
    }
  }

  depends_on = ["ultradns_tcpool.test-probe-http-maximal"]
}
`
