---
layout: "ultradns"
page_title: "UltraDNS: ultradns_tcpool"
sidebar_current: "docs-ultradns-resource-tcpool"
description: |-
  Provides an UltraDNS Traffic Controller pool resource.
---

# ultradns\_tcpool

Provides an UltraDNS Traffic Controller pool resource.

## Example Usage

```hcl
# Create a Traffic Controller pool
resource "ultradns_tcpool" "pool" {
  zone        = "${var.ultradns_domain}"
  name        = "terraform-tcpool"
  ttl         = 300
  description = "Minimal TC Pool"

  rdata {
    host = "192.168.0.10"
  }
}
```

## Argument Reference

See [related part of UltraDNS Docs](https://restapi.ultradns.com/v1/docs#post-rrset) for details about valid values.

The following arguments are supported:

* `zone` - (Required) The domain to add the record to
* `name` - (Required) The name of the record
* `rdata` - (Required) a list of rdata blocks, one for each member in the pool. Record Data documented below.
* `description` - (Required) Description of the Traffic Controller pool. Valid values are strings less than 256 characters.
* `ttl` - (Optional) The TTL of the record. Default: `3600`.
* `run_probes` - (Optional) Boolean to run probes for this pool. Default: `true`.
* `act_on_probes` - (Optional) Boolean to enable and disable pool records when probes are run. Default: `true`.
* `max_to_lb` - (Optional) Determines the number of records to balance between. Valid values are integers  `0` - `len(rdata)`. Default: `0`.
* `backup_record_rdata` - (Optional) IPv4 address or CNAME for the backup record. Default: `nil`.
* `backup_record_failover_delay` - (Optional) Time in minutes that Traffic Controller waits after detecting that the pool record has failed before activating primary records. Valid values are integers `0` - `30`. Default: `0`.

Record Data blocks support the following:

* `host` - (Required) IPv4 address or CNAME for the pool member.
* `failover_delay` - (Optional) Time in minutes that Traffic Controller waits after detecting that the pool record has failed before activating secondary records. `0` will activate the secondary records immediately. Integer. Range: `0` - `30`. Default: `0`.
* `priority` - (Optional) Indicates the serving preference for this pool record. Valid values are integers `1` or greater. Default: `1`.
* `run_probes` - (Optional) Whether probes are run for this pool record. Boolean. Default: `true`.
* `state` - (Optional) Current state of the pool record. String. Must be one of `"NORMAL"`, `"ACTIVE"`, or `"INACTIVE"`. Default: `"NORMAL"`.
* `threshold` - (Optional) How many probes must agree before the record state is changed. Valid values are integers `1` - `len(probes)`. Default: `1`.
* `weight` - (Optional) Traffic load to send to each server in the Traffic Controller pool. Valid values are integers `2` - `100`. Default: `2`

## Attributes Reference

The following attributes are exported:

* `id` - The record ID
* `hostname` - The FQDN of the record
