---
layout: "akamai"
page_title: "Akamai: akamai_gtm_data_center"
sidebar_current: "docs-akamai-resource-gtm-data-center"
description: |-
  Provides access to a GTM data center managed by Akamai.
---

# akamai\_gtm\_data\_center

Provides access to a GTM data center managed by Akamai.

## Example Usage

```
resource "akamai_gtm_data_center" "some_dc" {
    name = "some_dc"
    domain = "some-domain.akadns.net"
    country = "GB"
    continent = "EU"
    city = "Downpatrick"
    longitude = -5.582
    latitude = 54.367
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Specifies a name for the data center.

* `domain` - (Required) Specifies the GTM domain with which to associate the data center.

* `city` - (Optional) Specifies the name of the city where the data center is located.

* `country` - (Optional) Specifies the two-letter ISO 3166 country code where the data center is located.

* `state_or_province` - (Optional) Specifies the state or province where the data center is located.

* `continent` - (Optional) Specifies the two-letter continent code where the data center is located. Valid values are `AF`, `AS`, `EU`, `NA`, `OC`, `OT`, or `SA`.

* `latitude` - (Optional) Specifies the latitude where the data center is located.

* `longitude` - (Optional) Specifies the longitude where the data center is located.

* `virtual` - (Optional) Specifies to clients whether the data center is virtual or physical.
