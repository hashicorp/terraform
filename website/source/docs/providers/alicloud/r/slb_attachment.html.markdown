---
layout: "alicloud"
page_title: "Alicloud: alicloud_slb_attachment"
sidebar_current: "docs-alicloud-resource-slb-attachment"
description: |-
  Provides an Application Load Banlancer Attachment resource.
---

# alicloud\_slb\_attachment

Provides an Application Load Balancer Attachment resource.

## Example Usage

```
# Create a new load balancer attachment for classic
resource "alicloud_slb" "default" {
  # Other parameters...
}

resource "alicloud_instance" "default" {
  # Other parameters...
}

resource "alicloud_slb_attachment" "default" {
  slb_id    = "${alicloud_slb.default.id}"
  instances = ["${alicloud_instance.default.id}"]
}
```

## Argument Reference

The following arguments are supported:

* `slb_id` - (Required) The ID of the SLB..
* `instances` - (Required) A list of instance ids to added backend server in the SLB. If dettachment instances then this value set [].

## Attributes Reference

The following attributes are exported:

* `backend_servers` - The backend servers of the load balancer.
