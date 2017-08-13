---
layout: "netapp"
page_title: "NetApp: netapp_cloud_workenv"
sidebar_current: "docs-netapp-datasource-cloud-workenv"
description: |-
  Gets information about a working environment
---

# netapp\_cloud\_workenv

Use this data source to retrieve a working environment

## Example Usage

```hcl
data "netapp_cloud_workenv" "vsa-workenv" {
  name = "vsaenv"
}

resource "netapp_cloud_volume" "vsa-cifs-volume" {
  workenv_id = "${data.netapp_cloud_workenv.vsa-workenv.public_id}"
  svm_name = "${data.netapp_cloud_workenv.vsa-workenv.svm_name}"
  name = "cifs_vol"
  ...
}
```

## Argument Reference
* `name` is the name of the working environment.

## Attributes Reference

* `public_id` - Public ID for the environment.
* `tenant_id` - ID of the tenant for the environment.
* `svm_name` - The SVM name assigned to the environment.
* `is_ha` - Flag indicating if environment is HA (true) or VSA (false).
