---
layout: "dns"
page_title: "DNS: dns_cname_record_set"
sidebar_current: "docs-dns-datasource-cname-record-set"
description: |-
  Get DNS CNAME record set.
---

# dns_cname_record_set

Use this data source to get DNS CNAME record set of the host.

## Example Usage

```hcl
data "dns_cname_record_set" "hashicorp" {
  host = "www.hashicorp.com"
}

output "hashi_cname" {
  value = "${data.dns_cname_record_set.hashi.cname}"
}
```

## Argument Reference

The following arguments are supported:

 * `host` - (required): Host to look up

## Attributes Reference

The following attributes are exported:

 * `id` - Set to `host`.

 * `cname` - A CNAME record associated with host.
