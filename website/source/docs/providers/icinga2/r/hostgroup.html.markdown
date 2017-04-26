---
layout: "icinga2"
page_title: "Icinga2: hostgroup"
sidebar_current: "docs-icinga2-resource-hostgroup"
description: |-
  Configures a hostgroup resource. This allows hostgroup to be configured, updated and deleted.
---

# icinga2\_hostgroup

Configures an Icinga2 hostgroup resource. This allows hostgroup to be configured, updated,
and deleted.

## Example Usage

```hcl
# Configure a new hostgroup to be monitored by an Icinga2 Server
provider "icinga2" {
  api_url = "https://192.168.33.5:5665/v1"
}

resource "icinga2_hostgroup" "my-hostgroup" {
  name         = "terraform-hostgroup-1"
  display_name = "Terraform Test HostGroup"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the hostgroup.
* `display_name` - (Required) The name of the hostgroup to display in the Icinga2 interface.

