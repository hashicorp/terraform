---
layout: "cf"
page_title: "Cloud Foundry: cf_asg"
sidebar_current: "docs-cf-resource-asg"
description: |-
  Provides a Cloud Foundry Appliction Security Group resource.
---

# cf\_asg

Provides an [application security group](https://docs.cloudfoundry.org/adminguide/app-sec-groups.html) 
resource for Cloud Foundry. This resource defines egress rules that can be applied to containers that 
stage and run applications.

## Example Usage

Basic usage

```
resource "cf_asg" "messaging" {

	name = "rmq-service"
	
    rule {
        protocol = "tcp"
        destination = "192.168.1.100"
        ports = "5671-5672,61613-61614,1883,8883"
		log = true
    }
    rule {
        protocol = "tcp"
        destination = "192.168.1.101"
        ports = "5671-5672,61613-61614,1883,8883"
		log = true
    }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the application security group
* `rule` - (Required) A list of egress rules with the following arguments
  - `protocol` - (Required) One of `tcp`, `udp`, or `all`
  - `destination` - (Required) The IP address or CIDR block that can receive traffic
  - `ports` - (Required) A single port, comma separated ports or range of ports that can receive traffic
  - `log` - (Optional) Set to `true` to enable logging. For more information on how to configure system logs to be sent to a syslog drain, review the [Using Log Management](https://docs.pivotal.io/pivotalcf/devguide/services/log-management.html) Services topic.

## Attributes Reference

The following attributes are exported:

* `id` - The GUID of the application security group
