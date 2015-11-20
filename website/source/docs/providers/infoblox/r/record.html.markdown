---
layout: "infoblox"
page_title: "Infoblox: infoblox_record"
sidebar_current: "docs-infoblox-resource-record"
description: |-
  Provides a Infoblox record resource.
---

# infoblox\_record

Provides a Infoblox record A resource.

## Example Usage

```
# Add a record to the domain
resource "infoblox_record" "foobar" {
	ipv4addr = "192.168.0.10"
	name = "terraform"
}
```

## Argument Reference

See [related part of Infoblox Docs](https://godoc.org/github.com/fanatic/go-infoblox) for details about valid values.

The following arguments are supported:

* `ipv4addr` - (Required) The ipv4address to add the record to
* `name` - (Required) The name of the record

## Attributes Reference

The following attributes are exported:

* `id` - The record ID
* `fqdn` - The FQDN of the record
