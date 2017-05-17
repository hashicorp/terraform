---
layout: "ignition"
page_title: "Provider: Ignition"
sidebar_current: "docs-ignition-index"
description: |-
  The Ignition provider is used to generate Ignition configuration files used by CoreOS Linux.
---

# Ignition Provider

The Ignition provider is used to generate [Ignition](https://coreos.com/ignition/docs/latest/) configuration files. _Ignition_ is the provisioning utility used by [CoreOS](https://coreos.com/) Linux.

The ignition provider is what we call a _logical provider_ and doesn't manage any _physical_ resources. It generates configurations files to be used by other resources.

Use the navigation to the left to read about the available resources.

## Example Usage

This config will write a single service unit (shown below) with the contents of an example service. This unit will be enabled as a dependency of multi-user.target and therefore start on boot

```hcl
# Systemd unit data resource containing the unit definition
data "ignition_systemd_unit" "example" {
  name = "example.service"
  content = "[Service]\nType=oneshot\nExecStart=/usr/bin/echo Hello World\n\n[Install]\nWantedBy=multi-user.target"
}

# Ingnition config include the previous defined systemd unit data resource
data "ignition_config" "example" {
  systemd = [
    "${data.ignition_systemd_unit.example.id}",
  ]
}

# Create a CoreOS server using the Igntion config.
resource "aws_instance" "web" {
  # ...

  user_data = "${data.ignition_config.example.rendered}"
}
```
