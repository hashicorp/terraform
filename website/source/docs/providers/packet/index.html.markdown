---
layout: "packet"
page_title: "Provider: Packet"
sidebar_current: "docs-packet-index"
description: |-
  The Packet provider is used to interact with the resources supported by Packet. The provider needs to be configured with the proper credentials before it can be used.
---

# Packet Provider

The Packet provider is used to interact with the resources supported by Packet.
The provider needs to be configured with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```
# Configure the Packet Provider
provider "packet" {
		auth_token = "${var.auth_token}"
}

# Create a project
resource "packet_project" "cool_project" {
		name = "My First Terraform Project"
		payment_method = "PAYMENT_METHOD_ID" # Only required for a non-default payment method
}

# Create a device and add it to tf_project_1
resource "packet_device" "web1" {
		hostname = "tf.coreos2"
		plan = "baremetal_1"
		facility = "ewr1"
		operating_system = "coreos_stable"
		billing_cycle = "hourly"
		project_id = "${packet_project.cool_project.id}"
}
```

## Argument Reference

The following arguments are supported:

* `auth_token` - (Required) This is your Packet API Auth token. This can also be specified
  with the `PACKET_AUTH_TOKEN` shell environment variable.
