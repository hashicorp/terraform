---
layout: "cf"
page_title: "Cloud Foundry: cf_domain"
sidebar_current: "docs-cf-datasource-domain"
description: |-
  Get information on a Cloud Foundry Domain.
---

# cf\_domain

Gets information on a Cloud Foundry domain.

## Example Usage

The following example looks up a name in the current deployment with the host name `local` within the local application domain.

```
data "cf_domain" "l" {
    sub_domain = "local"
}
```

## Argument Reference

The following arguments are supported and will be used to perform the lookup:

* `name` - (Optional) This value will be computed based on the `sub-domain` or `domain` attributes. If provided then this argument will be used as the full domain name.
* `sub-domain` - (Optional) The sub-domain of the full domain name
* `domain` - (Optional) The domain name

## Attributes Reference

The following attributes are exported:

* `id` - The GUID of the domain
* `name` - The full domain name if not provided as an argument
* `domain`- The part of the domain name if not provided as an argument
* `org` - The org if this is a private domain owned by an org
