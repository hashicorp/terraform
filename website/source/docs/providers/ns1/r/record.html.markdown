---
layout: "ns1"
page_title: "NS1: ns1_record"
sidebar_current: "docs-ns1-resource-record"
description: |-
  Provides a NS1 Record resource.
---

# ns1\_record

Provides a NS1 Record resource. This can be used to create, modify, and delete records.

## Example Usage

```
resource "ns1_zone" "tld" {
  zone = "terraform.example"
}

resource "ns1_record" "www" {
  zone   = "${ns1_zone.tld.zone}"
  domain = "www.${ns1_zone.tld.zone}"
  type   = "CNAME"
  ttl    = 60

  answers = {
    answer = "sub1.${ns1_zone.tld.zone}"
  }

  answers = {
    answer = "sub2.${ns1_zone.tld.zone}"
  }

  filters = {
    filter = "select_first_n"

    config = {
      N = 1
    }
  }
}
```

## Argument Reference

The following arguments are supported:

* `zone` - (Required) The zone the record belongs to.
* `domain` - (Required) The records' domain.
* `type` - (Required) The records' RR type.
* `ttl` - (Optional) The records' time to live.
* `link` - (Optional) The target record to link to. This means this record is a 'linked' record, and it inherits all properties from its target.
* `use_client_subnet` - (Optional) Whether to use EDNS client subnet data when available(in filter chain).
* `answers` - (Optional) The list of the RDATA fields for the records' specified type. Answers are documented below.
* `filters` - (Optional) The list of NS1 filters for the record(order matters). Filters are documented below.

Answers (`answers`) support the following:

* `answer` - (Required) List of RDATA fields.
* `region` - (Required) The region this answer belongs to.

Filters (`filters`) support the following:

* `filter` - (Required) The type of filter.
* `disabled` - (Required) Determines whether the filter is applied in the filter chain.
* `config` - (Required) The filters' configuration. Simple key/value pairs determined by the filter type.
