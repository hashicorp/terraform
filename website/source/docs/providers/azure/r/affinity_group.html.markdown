---
layout: "azure"
page_title: "Azure: azure_affinity_group"
sidebar_current: "docs-azure-affinity-group"
description: |-
    Creates a new affinity group on Azure.
---

# azure\_affinity\_group

Creates a new affinity group on Azure.

## Example Usage

```hcl
resource "azure_affinity_group" "terraform-main-group" {
  name        = "terraform-group"
  location    = "North Europe"
  label       = "tf-group-01"
  description = "Affinity group created by Terraform."
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the affinity group. Must be unique on your
    Azure subscription.

* `location` - (Required) The location where the affinity group should be created.
    For a list of all Azure locations, please consult [this link](https://azure.microsoft.com/en-us/regions/).

* `label` - (Required) A label to be used for tracking purposes.

* `description` - (Optional) A description for the affinity group.

## Attributes Reference

The following attributes are exported:

* `id` - The affinity group ID. Coincides with the given `name`.
