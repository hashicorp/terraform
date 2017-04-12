---
layout: "alicloud"
page_title: "Alicloud: alicloud_instance_types"
sidebar_current: "docs-alicloud-datasource-instance-types"
description: |-
    Provides a list of Ecs Instance Types for use in alicloud_instance resource.
---

# alicloud_instance_types

The Instance Types data source list the ecs_instance_types of Alicloud.

## Example Usage

```hcl
# Declare the data source
data "alicloud_instance_types" "1c2g" {
  cpu_core_count = 1
  memory_size    = 2
}

# Create ecs instance with the first matched instance_type
resource "alicloud_instance" "instance" {
  instance_type = "${data.alicloud_instance_types.1c2g.instance_types.0.id}"

  # ...
}
```

## Argument Reference

The following arguments are supported:

* `cpu_core_count` - (Optional) Limit search to specific cpu core count.
* `memory_size` - (Optional) Limit search to specific memory size.
* `instance_type_family` - (Optional) Allows to filter list of Instance Types based on their
family name, for example 'ecs.n1'.

## Attributes Reference

The following attributes are exported:

* `id` - ID of the instance type.
* `cpu_core_count` - Number of CPU cores.
* `memory_size` - Size of memory, measured in GB.
* `family` - The instance type family.
