---
layout: "dns"
page_title: "DNS: dns_cname_record"
sidebar_current: "docs-dns-datasource-cname-record"
description: |-
  Get DNS CNAME records.
---

# dns\_cname\_record

Use this data source to get DNS CNAME records of the host.

## Example Usage

```
data "dns_cname_record" "hashicorp" {
  host = "www.hashicorp.com"
}

output "hashi_cname" {
  value = "${data.dns_cname_record.hashi.cname}"
}
```

## Argument Reference

The following arguments are supported:

 * `host` - (required): Host to look up

## Attributes Reference

The following attributes are exported:

 * `id` - Set to `host`.

 * `cname` - A CNAME record associated with host.
