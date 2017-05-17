---
layout: "dns"
page_title: "DNS: dns_ptr_record"
sidebar_current: "docs-dns-ptr-record"
description: |-
  Creates a PTR type DNS record.
---

# dns_ptr_record

Creates a PTR type DNS record.

## Example Usage

```hcl
resource "dns_ptr_record" "dns-sd" {
  zone = "example.com."
  name = "r._dns-sd"
  ptr  = "example.com."
  ttl  = 300
}
```

## Argument Reference

The following arguments are supported:

* `zone` - (Required) DNS zone the record belongs to. It must be an FQDN, that is, include the trailing dot.
* `name` - (Required) The name of the record. The `zone` argument will be appended to this value to create the full record path.
* `ptr` - (Required) The canonical name this record will point to.
* `ttl` - (Optional) The TTL of the record set. Defaults to `3600`.

## Attributes Reference

The following attributes are exported:

* `zone` - See Argument Reference above.
* `name` - See Argument Reference above.
* `ptr` - See Argument Reference above.
* `ttl` - See Argument Reference above.
