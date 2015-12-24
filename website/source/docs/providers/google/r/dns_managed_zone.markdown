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

```
resource "google_dns_managed_zone" "prod" {
    name = "prod-zone"
    dns_name = "prod.mydomain.com."
    description = "Production DNS zone"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the resource, required by GCE.
    Changing this forces a new resource to be created.

* `dns_name` - (Required) The DNS name of this zone, e.g. "terraform.io".

* `description` - (Optional) A textual description field.

## Attributes Reference

The following attributes are exported:

* `name` - The name of the resource.
* `dns_name` - The DNS name of this zone.
* `name_servers` - The list of nameservers that will be authoritative for this
  domain.  Use NS records to redirect from your DNS provider to these names,
thus making Google Cloud DNS authoritative for this zone.
