---
layout: "dns"
page_title: "DNS: dns_txt_record"
sidebar_current: "docs-dns-datasource-txt-record"
description: |-
  Get DNS TXT records.
---

# dns\_txt\_record

Use this data source to get DNS TXT records of the host.

## Example Usage

```
data "dns_txt_record" "hashicorp" {
  host = "www.hashicorp.com"
}

output "hashi_txt" {
  value = "${data.dns_txt_record.hashi.record}"
}

output "hashi_txts" {
  value = "${join(",", data.dns_txt_record.hashi.records})"
}
```

## Argument Reference

The following arguments are supported:

 * `host` - (required): Host to look up

## Attributes Reference

The following attributes are exported:

 * `id` - Set to `host`.

 * `record` - The first TXT record.
 
 * `records` - A list of TXT records.
