---
layout: "icinga2"
page_title: "Icinga2: host"
sidebar_current: "docs-icinga2-resource-host"
description: |-
  Configures a host resource. This allows hosts to be configured, updated and deleted.
---

# icinga2\_host

Configures an Icinga2 host resource. This allows hosts to be configured, updated,
and deleted.

## Example Usage

```hcl
# Configure a new host to be monitored by an Icinga2 Server
provider "icinga2" {
  api_url = "https://192.168.33.5:5665/v1"
}

resource "icinga2_host" "host" {
  hostname      = "terraform-host-1"
  address       = "10.10.10.1"
  check_command = "hostalive"
  templates     = ["bp-host-web"]

  vars {
    os        = "linux"
    osver     = "1"
    allowance = "none"
  }
}
```

## Argument Reference

The following arguments are supported:

* `address`  - (Required) The address of the host.
* `check_command` - (Required) The name of an existing Icinga2 CheckCommand object that is used to determine if the host is available or not.
* `hostname` - (Required) The hostname of the host.
* `templates` - (Optional) A list of Icinga2 templates to assign to the host.
* `vars` - (Optional) A mapping of variables to assign to the host.

