---
layout: "ovh"
page_title: "OVH: publiccloud_region"
sidebar_current: "docs-ovh-datasource-publiccloud-region"
description: |-
  Get information & status of a region associated with a public cloud project.
---

# publiccloud\_region

Use this data source to retrieve information about a region associated with a
public cloud project. The region must be associated with the project.

## Example Usage

```hcl
data "ovh_publiccloud_region" "GRA1" {
   project_id = "XXXXXX"
   region = "GRA1"
}
```

## Argument Reference


* `project_id` - (Required) The id of the public cloud project. If omitted,
    the `OVH_PROJECT_ID` environment variable is used.

* `region` - (Required) The name of the region associated with the public cloud
project.

## Attributes Reference

`id` is set to the ID of the project concatenated with the name of the region.
In addition, the following attributes are exported:

* `continentCode` - the code of the geographic continent the region is running.
E.g.: EU for Europe, US for America...
* `datacenterLocation` - The location code of the datacenter.
E.g.: "GRA", meaning Gravelines, for region "GRA1"
* `services` - The list of public cloud services running within the region
  * `name` - the name of the public cloud service
  * `status` - the status of the service
