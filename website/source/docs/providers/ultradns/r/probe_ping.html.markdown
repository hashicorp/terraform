---
layout: "ultradns"
page_title: "UltraDNS: ultradns_probe_ping"
sidebar_current: "docs-ultradns-resource-probe-ping"
description: |-
  Provides an UltraDNS Ping Probe
---

# ultradns\_probe\_ping

Provides an UltraDNS ping probe

## Example Usage

```hcl
resource "ultradns_probe_ping" "probe" {
  zone        = "${ultradns_tcpool.pool.zone}"
  name        = "${ultradns_tcpool.pool.name}"
  pool_record = "10.3.0.1"

  agents = ["DALLAS", "AMSTERDAM"]

  interval  = "ONE_MINUTE"
  threshold = 1

  ping_probe {
    packets     = 15
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
}
```

## Argument Reference

The following arguments are supported:

* `zone` - (Required) The domain of the pool to probe.
* `name` - (Required) The name of the pool to probe.
- `pool_record` - (optional) IP address or domain. If provided, a record-level probe is created, otherwise a pool-level probe is created.
- `agents` - (Required) List of locations that will be used for probing. One or more values must be specified. Valid values are `"NEW_YORK"`, `"PALO_ALTO"`, `"DALLAS"` & `"AMSTERDAM"`.
- `threshold` - (Required) Number of agents that must agree for a probe state to be changed.
- `ping_probe` - (Required) a Ping Probe block.
- `interval` - (Optional) Length of time between probes in minutes. Valid values are `"HALF_MINUTE"`, `"ONE_MINUTE"`, `"TWO_MINUTES"`, `"FIVE_MINUTES"`, `"TEN_MINUTES"` & `"FIFTEEN_MINUTE"`. Default: `"FIVE_MINUTES"`.

Ping Probe block
- `packets` - (Optional) Number of ICMP packets to send. Default `3`.
- `packet_size` - (Optional) Size of packets in bytes. Default `56`.
- `limit` - (Required) One or more Limit blocks. Only one limit block may exist for each name.

Limit block
- `name` - (Required) Kind of limit. Valid values are `"lossPercent"`, `"total"`, `"average"`, `"run"` & `"avgRun"`.
- `warning` - (Optional) Amount to trigger a warning.
- `critical` - (Optional) Amount to trigger a critical.
- `fail` - (Optional) Amount to trigger a failure.
