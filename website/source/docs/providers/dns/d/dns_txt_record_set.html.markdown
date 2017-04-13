---
layout: "dns"
page_title: "DNS: dns_txt_record_set"
sidebar_current: "docs-dns-datasource-txt-record-set"
description: |-
  Get DNS TXT record set.
---

# dns_txt_record_set

Use this data source to get DNS TXT record set of the host.

## Example Usage

```hcl
data "dns_txt_record_set" "hashicorp" {
  host = "www.hashicorp.com"
}

output "hashi_txt" {
  value = "${data.dns_txt_record_set.hashi.record}"
}

output "hashi_txts" {
  value = "${join(",", data.dns_txt_record_set.hashi.records})"
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
