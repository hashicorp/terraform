---
layout: "ignition"
page_title: "Ignition: ignition_networkd_unit"
sidebar_current: "docs-ignition-datasource-networkd-unit"
description: |-
  Describes the desired state of the networkd units.
---

# ignition\_networkd\_unit

Describes the desired state of the networkd units.

## Example Usage

```hcl
data "ignition_networkd_unit" "example" {
	name = "00-eth0.network"
	content = "[Match]\nName=eth0\n\n[Network]\nAddress=10.0.1.7"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the file. This must be suffixed with a valid unit type (e.g. _00-eth0.network_).

* `content` - (Required) The contents of the networkd file.

## Attributes Reference

The following attributes are exported:

* `id` - ID used to reference this resource in _ignition_config_.