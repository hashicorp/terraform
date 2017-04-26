---
layout: "profitbricks"
page_title: "ProfitBricks: profitbricks_server"
sidebar_current: "docs-profitbricks-resource-server"
description: |-
  Creates and manages ProfitBricks Server objects.
---

# profitbricks\_server

Manages a Servers on ProfitBricks

## Example Usage

This resource will create an operational server. After this section completes, the provisioner can be called.

```hcl
resource "profitbricks_server" "example" {
  name              = "server"
  datacenter_id     = "${profitbricks_datacenter.example.id}"
  cores             = 1
  ram               = 1024
  availability_zone = "ZONE_1"
  cpu_family        = "AMD_OPTERON"

  volume {
    name           = "new"
    image_name     = "${var.ubuntu}"
    size           = 5
    disk_type      = "SSD"
    ssh_key_path   = "${var.private_key_path}"
    image_password = "test1234"
  }

  nic {
    lan             = "${profitbricks_lan.example.id}"
    dhcp            = true
    ip              = "${profitbricks_ipblock.example.ip}"
    firewall_active = true

    firewall {
      protocol         = "TCP"
      name             = "SSH"
      port_range_start = 22
      port_range_end   = 22
    }
  }
}
```

##Argument reference

* `name` - (Required) [string] The name of the server.
* `datacenter_id` - (Required)[string]
* `cores` - (Required)[integer] Number of server cores.
* `ram` - (Required)[integer] The amount of memory for the server in MB.
* `availability_zone` - (Optional)[string] The availability zone in which the server should exist.
* `licence_type` - (Optional)[string] Sets the OS type of the server.
* `cpuFamily` - (Optional)[string] Sets the CPU type. "AMD_OPTERON" or "INTEL_XEON". Defaults to "AMD_OPTERON".
* `volume` -  (Required) See Volume section.
* `nic` - (Required) See NIC section.
* `firewall` - (Optional) See Firewall Rule section.
