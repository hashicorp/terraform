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
    ami = "ami-1234"
    instance_type = "m1.small"
    tags {
        Name = "HelloWorld"
    }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the instance. Changing this forces a new
    resource to be created.

* `display_name` - (Optional) The display name of the instance.

* `service_offering` - (Required) The service offering used for this instance.

* `network` - (Optional) The name of the network to connect this instance to.
    Changing this forces a new resource to be created.

* `ipaddress` - (Optional) The IP address to assign to this instance. Changing
    this forces a new resource to be created.

* `template` - (Required) The name of the template used for this instance.
    Changing this forces a new resource to be created.

* `zone` - (Required) The name of the zone where this instance will be created.
    Changing this forces a new resource to be created.

* `user_data` - (Optional) The user data to provide when launching the instance.

* `expunge` - (Optional) This determines if the instance is expunged when it is
    destroyed (defaults false)

## Attributes Reference

The following attributes are exported:

* `id` - The instance ID.
* `display_name` - The display name of the instance.
