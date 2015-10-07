---
layout: "packet"
page_title: "Packet: packet_ssh_key"
sidebar_current: "docs-packet-resource-project"
description: |-
  Provides a Packet Project resource.
---

# packet\_project

Provides a Packet Project resource to allow you manage devices
in your projects.

## Example Usage

```
# Create a new Project
resource "packet_project" "tf_project_1" {
    name = "Terraform Fun"
    payment_method = "payment-method-id"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the SSH key for identification
* `payment_method` - (Required) The id of the payment method on file to use for services created
on this project.

## Attributes Reference

The following attributes are exported:

* `id` - The unique ID of the key
* `payment_method` - The id of the payment method on file to use for services created
on this project.
* `created` - The timestamp for when the SSH key was created
* `updated` - The timestamp for the last time the SSH key was udpated
