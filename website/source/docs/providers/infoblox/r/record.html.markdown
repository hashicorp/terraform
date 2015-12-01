---
layout: "infoblox"
page_title: "Infoblox: infoblox_record"
sidebar_current: "docs-infoblox-resource-record"
description: |-
  Provides a Infoblox record resource.
---

# infoblox\_record

Provides a Infoblox record resource.

## Example Usage

```
# Add a record to the domain
resource "infoblox_record" "foobar" {
	value = "192.168.0.10"
	name = "terraform"
	domain = "mydomain.com"
	type = "A"
	ttl = 3600
}
```

## Argument Reference

See [related part of Infoblox Docs](https://godoc.org/github.com/fanatic/go-infoblox) for details about valid values.

The following arguments are supported:

* `domain` - (Required) The domain to add the record to
* `value` - (Required) The value of the record; its usage will depend on the `type` (see below)
* `name` - (Required) The name of the record
* `ttl` - (Integer, Optional) The TTL of the record
* `type` - (Required) The type of the record

## DNS Record Types

The type of record being created affects the interpretation of the `value` argument.

#### A Record

* `value` is the hostname

#### CNAME Record

* `value` is the alias name

#### AAAA Record

* `value` is the IPv6 address

## Attributes Reference

The following attributes are exported:

* `domain` - The domain of the record
* `value` - The value of the record
* `name` - The name of the record
* `type` - The type of the record
* `ttl` - The TTL of the record
