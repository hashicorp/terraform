---
layout: "azure"
page_title: "Azure: azure_hosted_service"
sidebar_current: "docs-azure-hosted-service"
description: |-
    Creates a new hosted service on Azure with its own .cloudapp.net domain.
---

# azure\_hosted\_service

Creates a new hosted service on Azure with its own .cloudapp.net domain.

## Example Usage

```
resource "azure_hosted_service" "terraform-service" {
    name = "terraform-service"
    location = "North Europe"
    ephemeral_contents = false
    description = "Hosted service created by Terraform."
    label = "tf-hs-01"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the hosted service. Must be unique on Azure.

* `location` - (Required) The location where the hosted service should be created.
    For a list of all Azure locations, please consult [this link](https://azure.microsoft.com/en-us/regions/).

* `ephemeral_contents` - (Required) A boolean value (true|false), specifying
    whether all the resources present in the hosted hosted service should be
    destroyed following the hosted service's destruction.

* `reverse_dns_fqdn` - (Optional) The reverse of the fully qualified domain name
    for the hosted service.

* `label` - (Optional) A label to be used for tracking purposes. Must be
    non-void. Defaults to `Made by Terraform.`.

* `description` - (Optional) A description for the hosted service.

## Attributes Reference

The following attributes are exported:

* `id` - The hosted service ID. Coincides with the given `name`.
