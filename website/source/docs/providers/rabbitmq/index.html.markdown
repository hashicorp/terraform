---
layout: "rabbitmq"
page_title: "Provider: RabbitMQ"
sidebar_current: "docs-rabbitmq-index"
description: |-
  A provider for a RabbitMQ Server.
---

# RabbitMQ Provider

[RabbitMQ](http://www.rabbitmq.com) is an AMQP message broker server. The
RabbitMQ provider exposes resources used to manage the configuration of
resources in a RabbitMQ server.

Use the navigation to the left to read about the available resources.

## Example Usage

The following is a minimal example:

```hcl
# Configure the RabbitMQ provider
provider "rabbitmq" {
  endpoint = "http://127.0.0.1"
  username = "guest"
  password = "guest"
}

# Create a virtual host
resource "rabbitmq_vhost" "vhost_1" {
  name = "vhost_1"
}
```

## Requirements

The RabbitMQ management plugin must be enabled to use this provider. You can
enable the plugin by doing something similar to:

```
$ sudo rabbitmq-plugins enable rabbitmq_management
```

## Argument Reference

The following arguments are supported:

* `endpoint` - (Required) The HTTP URL of the management plugin on the
  RabbitMQ server. The RabbitMQ management plugin *must* be enabled in order
  to use this provder. _Note_: This is not the IP address or hostname of the
  RabbitMQ server that you would use to access RabbitMQ directly.
* `username` - (Required) Username to use to authenticate with the server.
* `password` - (Optional) Password for the given user.
* `insecure` - (Optional) Trust self-signed certificates.
* `cacert_file` - (Optional) The path to a custom CA / intermediate certificate.
