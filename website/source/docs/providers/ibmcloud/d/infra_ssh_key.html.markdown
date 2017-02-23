---
layout: "ibmcloud"
page_title: "IBM Cloud: ibmcloud_infra_ssh_key"
sidebar_current: "docs-ibmcloud-datasource-infra-ssh-key"
description: |-
  Get information on a IBM Cloud Infrastructure SSH Key
---

# ibmcloud\_infra_ssh_key

Use this data source to import the details of an *existing* SSH key as a read-only data source.

## Example Usage

```hcl
data "ibmcloud_infra_ssh_key" "public_key" {
    label = "Terraform Public Key"
}
```

The fields of the data source can then be referenced by other resources within the same configuration using
interpolation syntax. For example, when specifying SSH keys in a ibmcloud_infra_virtual_guest resource configuration,
the numeric "IDs" are often unknown. Using the above data source as an example, it would be possible to
reference the `id` property in a ibmcloud_infra_virtual_guest resource:

```hcl
resource "ibmcloud_infra_virtual_guest" "vm1" {
    ...
    ssh_key_ids = ["${data.ibmcloud_infra_ssh_key.public_key.id}"]
    ...
}
```

## Argument Reference

* `label` - (Required) The label of the SSH key, as it was defined in SoftLayer
* `most_recent` - (Optional) If more than SSH key matches the label, use the most recent key

NOTE: If more or less than a single match is returned by the search, Terraform will fail.
Ensure that your label is specific enough to return a single SSH key only,
or use *most_recent* to choose the most recent one.

## Attributes Reference

`id` is set to the ID of the SSH key.  In addition, the following attributes are exported:

* `fingerprint` - sequence of bytes to authenticate or lookup a longer ssh key
* `public_key` - the public key contents
* `notes` - notes stored with the SSH key
