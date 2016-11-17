---
layout: "akamai"
page_title: "Akamai: akamai_gtm_domain"
sidebar_current: "docs-akamai-resource-gtm-domain"
description: |-
  Provides access to a GTM domain managed by Akamai.
---

# akamai\_gtm\_domain

Provides access to a GTM domain managed by Akamai.

## Example Usage

```
resource "akamai_gtm_domain" "some_domain" {
    name = "some-example.akadns.net"
    type = "basic"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Name of the GTM domain.

* `type` - (Required) The type of GTM domain.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the GTM domain.

* `name` - The name of the GTM domain.

* `type` - The GTM domain's type.
