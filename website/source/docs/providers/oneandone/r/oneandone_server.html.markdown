---
layout: "oneandone"
page_title: "1&1: oneandone_server"
sidebar_current: "docs-oneandone-resource-server"
description: |-
  Creates and manages 1&1 Server.
---

# oneandone\_server

Manages a Server on 1&1

## Example Usage

```
resource "oneandone_server" "server" {
  name = "Example"
  description = "Terraform 1and1 tutorial"
  image = "ubuntu"
  datacenter = "GB"
  vcores = 1
  cores_per_processor = 1
  ram = 2
  ssh_key_path = "/path/to/prvate/ssh_key"
  hdds = [
    {
      disk_size = 60
      is_main = true
    }
  ]

  provisioner "remote-exec" {
    inline = [
      "apt-get update",
      "apt-get -y install nginx",
    ]
  }
}
```

## Argument Reference

The following arguments are supported:

* `cores_per_processor` -(Required)[integer] Number of cores per processor
* `datacenter` - (Optional)[string] Location of desired 1and1 datacenter ["DE", "GB", "US", "ES" ]
* `description` - (Optional)[string] Description of the server
* `firewall_policy_id` - (Optional)[string] ID of firewall policy
* `hdds` - (Required)[collection] List of HDDs. One HDD must be main.
* `*disk_size` -(Required)[integer] The size of HDD
* `*is_main` - (Optional)[boolean] Indicates if HDD is to be used as main hard disk of the server
* `image` -(Required)[string] The name of a desired image to be provisioned with the server
* `ip` - (Optional)[string] IP address for the server
* `loadbalancer_id` - (Optional)[string] ID of the load balancer
* `monitoring_policy_id` - (Optional)[string] ID of monitoring policy
* `name` -(Required)[string] The name of the server.
* `password` - (Optional)[string] Desired password.
* `ram` -(Required)[float] Size of ram.
* `ssh_key_path` - (Optional)[string] Path to private ssh key
* `vcores` -(Required)[integer] Number of virtual cores.
