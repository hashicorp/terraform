---
layout: "opc"
page_title: "Oracle: opc_compute_vnic"
sidebar_current: "docs-opc-datasource-vnic"
description: |-
  Gets information about the configuration of a Virtual NIC.
---

# opc\_compute\_vnic

Use this data source to access the configuration of a Virtual NIC.

## Example Usage

```hcl
data "opc_compute_vnic" "current" {
  name = "my_vnic_name"
}

output "mac_address" {
  value = "${data.opc_compute_vnic.current.mac_address}"
}
```

## Argument Reference
* `name` is the name of the Virtual NIC.

## Attributes Reference

* `description` is a description of the Virtual NIC.

* `mac_address` is the MAC Address of the Virtual NIC.

* `tags` is a list of Tags associated with the Virtual NIC.

* `transit_flag` is `true` if the Virtual NIC is of the type `transit`.

* `uri` is the Unique Resource Locator of the Virtual NIC.
