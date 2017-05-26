---
layout: "icinga2"
page_title: "Icinga2: host"
sidebar_current: "docs-icinga2-resource-checkcommand"
description: |-
  Configures a checkcommand resource. This allows checkcommands to be configured, updated and deleted.
---

# icinga2\_checkcommand

Configures an Icinga2 checkcommand resource. This allows checkcommands to be configured, updated,
and deleted.

## Example Usage

```hcl
# Configure a new checkcommand on an Icinga2 Server, that can be used to monitor hosts and/or services
provider "icinga2" {
  api_url = "https://192.168.33.5:5665/v1"
}

resource "icinga2_checkcommand" "apache_status" {
  name      = "apache_status"
  templates = ["apache-status", "plugin-check-command", "plugin-check-command", "ipv4-or-ipv6"]
  command   = "/usr/lib64/nagios/plugins/check_apache_status.pl"

  arguments = {
    "-H" = "$apache_status_address$"
    "-c" = "$apache_status_critical$"
    "-p" = "$apache_status_port$"
  }
}
```

## Argument Reference

The following arguments are supported:

* `arguments` - (Optional) A mapping of arguments to include with the command.
* `command` - (Required) Path to the command te be executed.
* `name` - (Required) Name by which to reference the checkcommand
* `templates` - (Optional) A list of Icinga2 templates to assign to the host.
