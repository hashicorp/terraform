---
layout: "cloudstack"
page_title: "CloudStack: cloudstack_instance"
sidebar_current: "docs-cloudstack-resource-instance"
description: |-
  Creates and automatically starts a virtual machine based on a service offering, disk offering, and template.
---

# cloudstack\_instance

Creates and automatically starts a virtual machine based on a service offering,
disk offering, and template.

## Example Usage

```
resource "cloudstack_instance" "web" {
    name = "server-1"
    service_offering= "small"
    network_id = "6eb22f91-7454-4107-89f4-36afcdf33021"
    template = "CentOS 6.5"
    zone = "zone-1"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the instance.

* `display_name` - (Optional) The display name of the instance.

* `group` - (Optional) The group name of the instance.

* `service_offering` - (Required) The name or ID of the service offering used
    for this instance.

* `network_id` - (Optional) The ID of the network to connect this instance
    to. Changing this forces a new resource to be created.

* `network` - (Optional, Deprecated) The name or ID of the network to connect
    this instance to. Changing this forces a new resource to be created.

* `ip_address` - (Optional) The IP address to assign to this instance. Changing
    this forces a new resource to be created.

* `ipaddress` - (Optional, Deprecated) The IP address to assign to this instance.
    Changing this forces a new resource to be created.

* `template` - (Required) The name or ID of the template used for this
    instance. Changing this forces a new resource to be created.

* `project` - (Optional) The name or ID of the project to deploy this
    instance to. Changing this forces a new resource to be created.

* `zone` - (Required) The name or ID of the zone where this instance will be
    created. Changing this forces a new resource to be created.

* `user_data` - (Optional) The user data to provide when launching the
    instance.

* `keypair` - (Optional) The name of the SSH key pair that will be used to
    access this instance.

* `expunge` - (Optional) This determines if the instance is expunged when it is
    destroyed (defaults false)

## Attributes Reference

The following attributes are exported:

* `id` - The instance ID.
* `display_name` - The display name of the instance.
