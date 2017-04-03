---
layout: "ibmcloud"
page_title: "IBM Cloud: vlan"
sidebar_current: "docs-ibmcloud-resource-infra-vlan"
description: |-
  Manages IBM Cloud infrastructure VLAN.
---

# ibmcloud\_infra\_vlan

Provides a `VLAN` resource. This allows public and private VLANs to be created, updated, and cancelled. 

The default SoftLayer account does not have permission to create a VLAN via SoftLayer API. To create a VLAN with Terraform, you should have permissions to create a VLAN in advance. Contact a SoftLayer sales person or open a ticket.

Existed VLANs can be managed by Terraform with the `terraform import` command. It requires a SoftLayer VLAN ID from  [VLANs](https://control.softlayer.com/network/vlans). Once the IDs are imported, they provides useful information such as subnets and child resource counts. When `terraform destroy` is run, the VLANs' billing item will be deleted. However, the VLAN remains in SoftLayer until resources such as virtual guests, secondary subnets, and firewalls on the VLAN are deleted. 

For additional details please refer to [SoftLayer API docs](http://sldn.softlayer.com/reference/datatypes/SoftLayer_Network_Vlan).

## Example Usage

```hcl
# Create a VLAN
resource "ibmcloud_infra_vlan" "test_vlan" {
   name = "test_vlan"
   datacenter = "dal06"
   type = "PUBLIC"
   subnet_size = 8
   router_hostname = "fcr01a.dal06"
}

```

## Argument Reference

The following arguments are supported:

* `datacenter` - (Required, string) The data center in which the VLAN resides.
* `type` - (Required, string) The type of VLAN. Accepted values are `PRIVATE` and `PUBLIC`.
* `subnet_size` - (Required, integer) The size of the primary subnet for the VLAN. Accepted values are `8`, `16`, `32`, and `64`.
* `name` - (Optional, string) The name of the VLAN.
* `router_hostname` - (Optional, string) The hostname of the primary router that the VLAN is associated with.

## Attributes Reference

The following attributes are exported:

* `id` - ID of the VLAN.
* `vlan_number` - The VLAN number as recorded within the SoftLayer network. This is configured directly on SoftLayer's networking equipment.
* `softlayer_managed` - Whether the VLAN is managed by SoftLayer or not. If the VLAN is created by SoftLayer automatically while other resources are created, set to `true`. If the VLAN is created by a user via the SoftLayer API, portal, or ticket, set to `false`.
* `child_resource_count` - A count of all of the resources, such as virtual servers and other network components, that are connected to the VLAN. 
* `subnets` - Collection of subnets associated with the VLAN.
