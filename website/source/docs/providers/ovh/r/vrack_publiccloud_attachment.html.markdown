---
layout: "ovh"
page_title: "OVH: vrack_publiccloud_attachment"
sidebar_current: "docs-ovh-resource-vrack-publiccloud-attachment"
description: |-
  Attach an existing PublicCloud project to an existing VRack.
---

# ovh_vrack\_publiccloud\_attachment

Attach an existing PublicCloud project to an existing VRack.

## Example Usage

```
resource "ovh_vrack_publiccloud_attachment" "attach" {
  vrack_id   = "12345"
  project_id = "67890"
}
```

## Argument Reference

The following arguments are supported:

* `vrack_id` - (Required) The id of the vrack. If omitted, the `OVH_VRACK_ID`
    environment variable is used.

* `project_id` - (Required) The id of the public cloud project. If omitted,
    the `OVH_VRACK_ID` environment variable is used.

## Attributes Reference

The following attributes are exported:

* `vrack_id` - See Argument Reference above.
* `project_id` - See Argument Reference above.

## Notes

The vrack attachment isn't a proper resource with an ID. As such, the resource id will
be forged from the vrack and project ids and there's no correct way to import the
resource in terraform. When the resource is created by terraform, it first checks if the
attachment already exists within OVH infrastructure; if it exists it set the resource id
without modifying anything. Otherwise, it will try to attach the vrack with the public
cloud project.
