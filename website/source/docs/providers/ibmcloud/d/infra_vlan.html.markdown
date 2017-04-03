---
layout: "ibmcloud"
page_title: "IBM Cloud: ibmcloud_infra_vlan"
sidebar_current: "docs-ibmcloud-datasource-infra-vlan"
description: |-
  Get information on a IBM Cloud Infrastructure vlan
---

# ibmcloud\_infra\_vlan


Import the details of an existing VLAN as a read-only data source. The fields of the data source can then be referenced by other resources within the same configuration using interpolation syntax. 


## Example Usage

```hcl
data "ibmcloud_infra_vlan" "vlan_foo" {
    name = "FOO"
}
```


The following example shows how you can use this data source to reference a VLAN ID in the _ibmcloud_infra_virtual_guest_ resource, since the numeric IDs are often unknown.

```hcl
resource "ibmcloud_infra_virtual_guest" "vg" {
    ...
    public_vlan_id = "${data.ibmcloud_infra_vlan.vlan_foo.id}"
    ...
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required if the number nor router hostname are provided) The name of the VLAN, as it was defined in SoftLayer. Names can be found in the SoftLayer portal, by navigating to [Network > IP Management > VLANs](https://control.softlayer.com/network/vlans).
* `number` - (Required if the name is not provided) The VLAN number, which can be found in the [SoftLayer portal](https://control.softlayer.com/network/vlans).
* `router_hostname` - (Required if the name is not provided) The primary VLAN router hostname, which can be found in the [SoftLayer portal](https://control.softlayer.com/network/vlans).

## Attributes Reference

The following attributes are exported:

`id` - Set to the ID of the VLAN.
