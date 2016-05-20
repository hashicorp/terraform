---
layout: "triton"
page_title: "Triton: triton_machine"
sidebar_current: "docs-triton-firewall"
description: |-
    The `triton_machine` resource represents a virtual machine or infrastructure container running in Triton. 
---

# triton\_machine

The `triton_machine` resource represents a virtual machine or infrastructure container running in Triton.

## Example Usages

Run a SmartOS base-64 machine.


```
resource "triton_machine" "test" {
    name = "example-machine"
    package = "g3-standard-0.25-smartos"
    image = "842e6fa6-6e9b-11e5-8402-1b490459e334"

    tags = {
        hello = "world"
    }
}
                
```

## Argument Reference

The following arguments are supported:

* `name` - (string)
    The friendly name for the machine. Triton will generate a name if one is not specified.

* `tags` - (map)
    A mapping of tags to apply to the machine.

* `package` - (string, Required)
    The name of the package to use for provisioning.

* `image` - (string, Required)
    The UUID of the image to provision.

* `networks` - (list of string)
    A list of the IDs of the desired networks for the machine.

* `firewall_enabled` - (boolean)  Default: `false`
    Whether the cloud firewall should be enabled for this machine.

* `root_authorized_keys` - (string)
    The public keys authorized for root access via SSH to the machine.

* `user_data` - (string)
    Data to be copied to the machine on boot.

* `user_script` - (string)
    The user script to run on boot (every boot on SmartMachines).

* `administrator_pw` - (string)
    The initial password for the Administrator user. Only used for Windows virtual machines.

## Attribute Reference

The following attributes are exported:

* `id` - (string) - The identifier representing the firewall rule in Triton. 
* `type` - (string) - The type of the machine (`smartmachine` or `virtualmachine`). 
* `state` - (string) - The current state of the machine. 
* `dataset` - (string) - The dataset URN with which the machine was provisioned. 
* `memory` - (int) - The amount of memory the machine has (in Mb). 
* `disk` - (int) - The amount of disk the machine has (in Gb). 
* `ips` - (list of strings) - IP addresses of the machine. 
* `primaryip` - (string) - The primary (public) IP address for the machine. 
* `created` - (string) - The time at which the machine was created. 
* `updated` - (string) - The time at which the machine was last updated. 
