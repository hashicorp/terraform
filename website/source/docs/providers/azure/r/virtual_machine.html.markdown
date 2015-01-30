---
layout: "azure"
page_title: "Azure: azure_virtual_machine"
sidebar_current: "docs-azure-resource-virtual-machine"
description: |-
  Manages a Virtual Machine resource within Azure.
---

# azure\_virtual\_machine

Manages a Virtual Machine resource within Azure.

## Example Usage

```
resource "azure_virtual_machine" "default" {
    name = "test"
    location = "West US"
    image = "b39f27a8b8c64d52b05eac6a62ebad85__Ubuntu-14_04-LTS-amd64-server-20140724-en-us-30GB"
    size = "Basic_A1"
    username = "${var.username}"
    password = ""${var.password}"
    ssh_public_key_file = "${var.azure_ssh_public_key_file}"
    endpoint {
        name = "http"
        protocol = "tcp"
        port = 80
        local_port = 80
    }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A name for the virtual machine. It must use between 3 and
   24 lowercase letters and numbers and it must be unique within Azure.

* `location` - (Required) The location that the virtual machine should be created in.

* `image` - (Required) A image to be used to create the virtual machine.

* `size` - (Required) Size that you want to use for the virtual machine.

* `username` - (Required) Name of the account that you will use to administer
    the virtual machine. You cannot use root for the user name.

* `password` - (Optional) Password for the admin account.

* `ssh_public_key_file` - (Optional) SSH key (PEM format).

* `ssh_port` - (Optional) SSH port.

* `endpoint` - (Optional) Can be specified multiple times for each
   endpoint rule. Each endpoint block supports fields documented below.

The `endpoint` block supports:

* `name` - (Required) The name of the endpoint.
* `protocol` - (Required) The protocol.
* `port` - (Required) The public port.
* `local_port` - (Required) The private port.

## Attributes Reference

The following attributes are exported:

* `url` - The URL for the virtual machine deployment.
* `ip_address` - The internal IP address of the virtual machine.
* `vip_address` - The public Virtual IP address of the virtual machine.
