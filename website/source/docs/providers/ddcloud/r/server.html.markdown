---
layout: "ddcloud"
page_title: "Dimension Data Managed Cloud Platform: server"
sidebar_current: "docs-ddcloud-resource-server"
description: |-
  Allows Terraform to manage a Managed Cloud Platform server.
---

# ddcloud\_server

A Server is a virtual machine. It is deployed in a network domain, and each of its network adapters are connected to a VLAN. Each server is created from an OS or customer image that also specifies sensible defaults for the server configuration. Only OS (built-in) images are currently supported.

## Example Usage

```
resource "ddcloud_server" "my-server" {
    name                    = "terraform-server"
    description             = "This is my Terraform test server."
    admin_password          = "password"

    memory_gb               = 8
    cpu_count               = 2

    networkdomain           = "${ddcloud_networkdomain.test-domain.id}"
    primary_adapter_ipv4    = "192.168.17.10"
    dns_primary             = "8.8.8.8"
    dns_secondary           = "8.8.4.4"

    osimage_name            = "CentOS 7 64-bit 2 CPU"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A name for the server.
* `description` - (Optional) A description for the server.
* `admin_password` - (Required) The initial administrative password for the server.
* `memory_gb` - (Optional) The amount of memory (in GB) allocated to the server. Defaults to the memory specified by the image from which the server is created.
* `cpu_count` - (Optional) The number of CPUs allocated to the server. Defaults to the CPU count specified by the image from which the server is created.
* `networkdomain` - (Required) The Id of the network domain in which the server is deployed.
* `primary_adapter_ipv4` - (Optional) The IPv4 address for the server's primary network adapter. Must specify one of either `primary_adapter_ipv4` or `primary_adapter_vlan`.
* `primary_adapter_vlan` - (Optional) The Id of the VLAN to which the server's primary network adapter will be attached (the first available IPv4 address will be allocated). Must specify one of either `primary_adapter_ipv4` or `primary_adapter_vlan`.
* `dns_primary` - (Required) The IP address of the server's primary DNS.
* `dns_secondary` - (Required) The IP address of the server's secondary DNS.
* `osimage_id` - (Required) The Id of the OS (built-in) image from which the server will be created. Must specify exactly one of `osimage_id` or `osimage_name`.
* `osimage_name` - (Required) The name of the OS (built-in) image from which the server will be created (the name must be unique within the data center in which the network domain is deployed). Must specify exactly one of `osimage_id` or `osimage_name`.
* `auto_start` - (Optional) Automatically start the server once it is deployed (default is false).

## Attributes Reference

* `osimage_id` - Calculated if `osimage_name` is specified.
* `osimage_name` - Calculated if `osimage_id` is specified.
* `primary_adapter_ipv4` - Calculated if `primary_adapter_vlan` is specified.
* `primary_adapter_vlan` - Calculated if `primary_adapter_ipv4` is specified.
