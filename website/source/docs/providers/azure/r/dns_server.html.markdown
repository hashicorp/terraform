---
layout: "azure"
page_title: "Azure: azure_dns_server"
sidebar_current: "docs-azure-resource-dns-server"
description: |-
    Creates a new DNS server definition to be used internally in Azure.
---

# azure\_dns\_server

Creates a new DNS server definition to be used internally in Azure.

## Example Usage

```hcl
resource "azure_dns_server" "google-dns" {
  name        = "google"
  dns_address = "8.8.8.8"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the DNS server reference. Changing this
    forces a new resource to be created.

* `dns_address` - (Required) The IP address of the DNS server.

## Attributes Reference

The following attributes are exported:

* `id` - The DNS server definition ID. Coincides with the given `name`.
