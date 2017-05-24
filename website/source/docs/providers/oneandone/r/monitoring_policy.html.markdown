---
layout: "oneandone"
page_title: "1&1: oneandone_monitoring_policy"
sidebar_current: "docs-oneandone-resource-monitoring-policy"
description: |-
  Creates and manages 1&1 Monitoring Policy.
---

# oneandone\_server

Manages a Monitoring Policy on 1&1

## Example Usage

```hcl
resource "oneandone_monitoring_policy" "mp" {
  name = "test_mp"
  agent = true
  email = "jasmin@stackpointcloud.com"

  thresholds = {
    cpu = {
      warning = {
        value = 50,
        alert = false
      }
      critical = {
        value = 66,
        alert = false
      }

    }
    ram = {
      warning = {
        value = 70,
        alert = true
      }
      critical = {
        value = 80,
        alert = true
      }
    },
    ram = {
      warning = {
        value = 85,
        alert = true
      }
      critical = {
        value = 95,
        alert = true
      }
    },
    disk = {
      warning = {
        value = 84,
        alert = true
      }
      critical = {
        value = 94,
        alert = true
      }
    },
    transfer = {
      warning = {
        value = 1000,
        alert = true
      }
      critical = {
        value = 2000,
        alert = true
      }
    },
    internal_ping = {
      warning = {
        value = 3000,
        alert = true
      }
      critical = {
        value = 4000,
        alert = true
      }
    }
  }
  ports = [
    {
      email_notification = true
      port = 443
      protocol = "TCP"
      alert_if = "NOT_RESPONDING"
    },
    {
      email_notification = false
      port = 80
      protocol = "TCP"
      alert_if = "NOT_RESPONDING"
    },
    {
      email_notification = true
      port = 21
      protocol = "TCP"
      alert_if = "NOT_RESPONDING"
    }
  ]

  processes = [
    {
      email_notification = false
      process = "httpdeamon"
      alert_if = "RUNNING"
    },
    {
      process = "iexplorer",
      alert_if = "NOT_RUNNING"
      email_notification = true
    }]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the VPN.
* `description` - (Optional) Description for the VPN
* `email` - (Optional)  Email address to which notifications monitoring system will send
* `agent- (Required) Indicates which monitoring type will be used. True: To use this monitoring type, you must install an agent on the server.  False: Monitor a server without installing an agent. Note: If you do not install an agent, you cannot retrieve information such as free hard disk space or ongoing processes.

Monitoring Policy Thresholds (`thresholds`) support the following:

* `cpu - (Required) CPU thresholds
    * `warning - (Required)Warning alert
            * `value - (Required) Warning to be issued when the threshold is reached. from 1 to 100
            * `alert - (Required) If set true warning will be issued.
        * `critical - (Required) Critical alert
            * `value - (Required) Warning to be issued when the threshold is reached. from 1 to 100
            * `alert - (Required) If set true warning will be issued.
* `ram - (Required) RAM threshold
    * `warning - (Required) Warning alert
        * `value - (Required) Warning to be issued when the threshold is reached. from 1 to 100
        * `alert - (Required) If set true warning will be issued.
    * `critical - (Required) Critical alert
        * `value - (Required) Warning to be issued when the threshold is reached. from 1 to 100
        * `alert - (Required) If set true warning will be issued.
* `disk - (Required) Hard Disk threshold
    * `warning - (Required) Warning alert
        * `value - (Required) Warning to be issued when the threshold is reached. from 1 to 100
        * `alert - (Required) If set true warning will be issued.
    * `critical - (Required) Critical alert
        * `value - (Required) Warning to be issued when the threshold is reached. from 1 to 100
        * `alert - (Required) If set true warning will be issued.
* `transfer - (Required) Data transfer threshold
    * `warning - (Required) Warning alert
        * `value - (Required) Warning to be issued when the threshold is reached. from 1 to 100
        * `alert - (Required) If set true warning will be issued.
    * `critical - (Required) Critical alert
        * `value - (Required) Warning to be issued when the threshold is reached. from 1 to 100
        * `alert - (Required) If set true warning will be issued.
* `internal_ping - (Required) Ping threshold
    * `warning - (Required) Warning alert
        * `value - (Required) Warning to be issued when the threshold is reached. from 1 to 100
        * `alert - (Required) If set true warning will be issued.
    * `critical - (Required) Critical alert
        * `value - (Required) Warning to be issued when the threshold is reached. from 1 to 100
        * `alert - (Required) If set true warning will be issued.

Monitoring Policy Ports (`ports`) support the following:

* `email_notification - (Required) If set true email will be sent.
* `port - (Required) Port number.
* `protocol` - (Required) The protocol of the port. Allowed values are `TCP`, `UDP`, `TCP/UDP`, `ICMP` and `IPSEC`.
* `alert_if - (Required) Condition for the alert to be issued.

Monitoring Policy Ports (`processes`) support the following:

* `email_notification - (Required) If set true email will be sent.
* `process - (Required) Process name.
* `alert_if - (Required) Condition for the alert to be issued.
