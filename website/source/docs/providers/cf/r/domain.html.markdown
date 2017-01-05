---
layout: "cf"
page_title: "Cloud Foundry: cf_domain"
sidebar_current: "docs-cf-resource-domain"
description: |-
  Provides a Cloud Foundry Domain resource.
---

# cf\_domain

Provides a resource for managing shared or private 
[domains](https://docs.cloudfoundry.org/devguide/deploy-apps/routes-domains.html#domains) in Cloud Foundry.

## Example Usage

The following is an example of a shared domain for a sub-domain of the default application domain 
retrieved via a [domain data source](http://localhost:4567/docs/providers/cloudfoundry/d/domain.html).

```
resource "cf_domain" "shared" {
    sub_domain = "dev"
    domain = "${data.cf_domain.apps.domain}"
}
```

The following example creates a private domain owned by the Org referenced by `cf_org.pcfdev-org.id`.

```
resource "cf_domain" "private" {
    name = "pcfdev-org.io"
  org = "${cf_org.pcfdev-org.id}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - Full name of domain. If specified then the `sub_domain` and `domain` attributes will be computed from the `name` 
* `sub_domain` - (Optional) Sub-domain part of full domain name. If specified the `domain` argument needs to be provided and the `name` will be computed.
* `domain` - (Optional) Domain part of full domain name. If specified the `sub_domain` argument needs to be provided and the `name` will be computed.
* `org` - (Optional) The GUID of the Org that owns this domain. If provided then this will be a private domain.

## Attributes Reference

The following attributes are exported:

* `id` - The GUID of the domain
