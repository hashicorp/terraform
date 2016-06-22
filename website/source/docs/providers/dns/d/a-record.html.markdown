---
layout: "dns"
page_title: "DNS: dns_a_record"
sidebar_current: "docs-dns-datasource-a-record"
description: |-
  Get DNS A records.
---

# dns\_a\_record

Use this data source to get DNS A records of the host.

## Example Usage

```
data "dns_a_record" "google" {
  host = "google.com"
}

output "google_addrs" {
  value = "${join(",", data.dns_a_record.google.addrs)}"
}
```

## Argument Reference

The following arguments are supported:

 * `host` - (required): Host to look up

 * `ipv4` - (optional) A flag to only use IPv4 records. By default, `ipv4 = false`.

 * `sort` - (optional) A flag to sort IPv4 records or allow round-robin retrieval. By default, `sort = true`,

## Attributes Reference

The following attributes are exported:

 * `id` - Set to `host`.

 * `addrs` - A list of IP addresses.
