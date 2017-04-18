---
layout: "ultradns"
page_title: "UltraDNS: ultradns_probe_http"
sidebar_current: "docs-ultradns-resource-probe-http"
description: |-
  Provides an UltraDNS HTTP probe
---

# ultradns\_probe\_http

Provides an UltraDNS HTTP probe

## Example Usage

```hcl
resource "ultradns_probe_http" "probe" {
  zone        = "${ultradns_tcpool.pool.zone}"
  name        = "${ultradns_tcpool.pool.name}"
  pool_record = "10.2.1.1"

  agents = ["DALLAS", "AMSTERDAM"]

  interval  = "ONE_MINUTE"
  threshold = 1

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
}
```

## Argument Reference

The following arguments are supported:

* `zone` - (Required) The domain of the pool to probe.
* `name` - (Required) The name of the pool to probe.
- `pool_record` - (optional) IP address or domain. If provided, a record-level probe is created, otherwise a pool-level probe is created.
- `agents` - (Required) List of locations that will be used for probing. One or more values must be specified. Valid values are `"NEW_YORK"`, `"PALO_ALTO"`, `"DALLAS"` & `"AMSTERDAM"`.
- `threshold` - (Required) Number of agents that must agree for a probe state to be changed.
- `http_probe` - (Required) an HTTP Probe block.
- `interval` - (Optional) Length of time between probes in minutes. Valid values are `"HALF_MINUTE"`, `"ONE_MINUTE"`, `"TWO_MINUTES"`, `"FIVE_MINUTES"`, `"TEN_MINUTES"` & `"FIFTEEN_MINUTE"`. Default: `"FIVE_MINUTES"`.

HTTP Probe block
- `transaction` - (Optional) One or more Transaction blocks.
- `total_limits` - (Optional) A Limit block, but with no `name` attribute.

Transaction block
- `method` - (Required) HTTP method. Valid values are`"GET"`, `"POST"`.
- `url` - (Required) URL to probe.
- `transmitted_data` - (Optional) Data to send to URL.
- `follow_redirects` - (Optional) Whether to follow redirects.
- `limit` - (Required) One or more Limit blocks. Only one limit block may exist for each name.

Limit block
- `name` - (Required) Kind of limit. Valid values are `"lossPercent"`, `"total"`, `"average"`, `"run"` & `"avgRun"`.
- `warning` - (Optional) Amount to trigger a warning.
- `critical` - (Optional) Amount to trigger a critical.
- `fail` - (Optional) Amount to trigger a failure.
