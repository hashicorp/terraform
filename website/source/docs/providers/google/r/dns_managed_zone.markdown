---
layout: "google"
page_title: "Google: google_dns_managed_zone"
sidebar_current: "docs-google-dns-managed-zone"
description: |-
  Manages a zone within Google Cloud DNS.
---

# google\_dns\_managed_zone

Manages a zone within Google Cloud DNS.

## Example Usage

```js
resource "google_dns_managed_zone" "prod" {
  name        = "prod-zone"
  dns_name    = "prod.mydomain.com."
  description = "Production DNS zone"
}
```

## Argument Reference

The following arguments are supported:

* `dns_name` - (Required) The DNS name of this zone, e.g. "terraform.io".

* `name` - (Required) A unique name for the resource, required by GCE.
    Changing this forces a new resource to be created.

- - -

* `description` - (Optional) A textual description field. Defaults to 'Managed by Terraform'.

* `project` - (Optional) The project in which the resource belongs. If it
    is not provided, the provider project is used.

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are
exported:

* `name_servers` - The list of nameservers that will be authoritative for this
    domain. Use NS records to redirect from your DNS provider to these names,
    thus making Google Cloud DNS authoritative for this zone.
