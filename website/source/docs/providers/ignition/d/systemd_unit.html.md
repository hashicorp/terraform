---
layout: "ignition"
page_title: "Ignition: ignition_systemd_unit"
sidebar_current: "docs-ignition-datasource-systemd-unit"
description: |-
  Describes the desired state of the systemd units.
---

# ignition\_systemd\_unit

Describes the desired state of the systemd units.

## Example Usage

```hcl
data "ignition_systemd_unit" "example" {
	name = "example.service"
	content = "[Service]\nType=oneshot\nExecStart=/usr/bin/echo Hello World\n\n[Install]\nWantedBy=multi-user.target"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Tthe name of the unit. This must be suffixed with a valid unit type (e.g. _thing.service_).

* `enable` - (Optional) Whether or not the service shall be enabled. When true, the service is enabled. In order for this to have any effect, the unit must have an install section. (default true)

* `mask` - (Optional) Whether or not the service shall be masked. When true, the service is masked by symlinking it to _/dev/null_.

* `content` - (Required) The contents of the unit. Optional when a dropin is provided.

* `dropin` - (Optional) The list of drop-ins for the unit.

The `dropin` block supports:

* `name` - (Required) The name of the drop-in. This must be suffixed with _.conf_.

* `content` - (Optional) The contents of the drop-in.

## Attributes Reference

The following attributes are exported:

* `id` - ID used to reference this resource in _ignition_config_.