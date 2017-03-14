---
layout: "ibmcloud"
page_title: "IBM Cloud: ibmcloud_infra_ssh_key"
sidebar_current: "docs-ibmcloud-datasource-infra-ssh-key"
description: |-
  Get information about an IBM Cloud Infrastructure SSH key.
---

# ibmcloud\_infra\_ssh\_key

Import the details of an existing SSH key as a read-only data source. The fields of the data source can then be referenced by other resources within the same configuration by using interpolation syntax. 

## Example Usage

```hcl
data "ibmcloud_infra_ssh_key" "public_key" {
    label = "Terraform Public Key"
}
```

The following example shows how you can use this data source to reference the SSH key IDs in the `ibmcloud_infra_virtual_guest` resource, since the numeric IDs are often unknown.

```hcl
resource "ibmcloud_infra_virtual_guest" "vm1" {
    ...
    ssh_key_ids = ["${data.ibmcloud_infra_ssh_key.public_key.id}"]
    ...
}
```

## Argument Reference

The following arguments are supported:

* `label` - (Required) The label of the SSH key, as it was defined in SoftLayer.
* `most_recent` - (Optional) If more than one SSH key matches the label, you can use this argument to import only the most recent key. Default value: `false`.

**NOTE**: If more or less than a single match is returned by the search, Terraform will fail. Ensure that your label is specific enough to return a single SSH key only, or use the `most_recent` argument.

## Attributes Reference

The following attributes are exported:

* `id` - The unique identifier of the SSH key.  
* `fingerprint` - Sequence of bytes to authenticate or look up a longer SSH key.
* `public_key` - The public key contents.
* `notes` - Notes stored with the SSH key.
