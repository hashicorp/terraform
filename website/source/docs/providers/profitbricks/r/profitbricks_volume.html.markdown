---
layout: "profitbricks"
page_title: "ProfitBricks: profitbricks_server"
sidebar_current: "docs-profitbricks-resource-volume"
description: |-
  Creates and manages ProfitBricks Volume objects.
---

# profitbricks\_volume

Manages a Volumes on ProfitBricks

## Example Usage

A primary volume will be created with the server. If there is a need for additional volume, this resource handles it.

```hcl
resource "profitbricks_volume" "example" {
  datacenter_id = "${profitbricks_datacenter.example.id}"
  server_id     = "${profitbricks_server.example.id}"
  image_name    = "${var.ubuntu}"
  size          = 5
  disk_type     = "HDD"
  ssh_key_path  = "${var.private_key_path}"
  bus           = "VIRTIO"
}
```

##Argument reference

* `datacenter_id` - (Required) [string] <sup>[1](#myfootnote1)</sup>
* `server_id` - (Required)[string] <sup>[1](#myfootnote1)</sup>
* `disk_type` - (Required) [string] The volume type, HDD or SSD.
* `bus` - (Required) [boolean] The bus type of the volume.
* `size` -  (Required)[integer] The size of the volume in GB.
* `ssh_key_path` -  (Required)[list] List of paths to files containing a public SSH key that will be injected into ProfitBricks provided Linux images. Required if `image_password` is not provided.
* `image_password` - [string] Required if `sshkey_path` is not provided.
* `image_name` - [string] The image or snapshot ID. It is required if `licence_type` is not provided.
* `licence_type` - [string] Required if `image_name` is not provided.
* `name` - (Optional) [string] The name of the volume.
* `availability_zone` - (Optional) [string] The storage availability zone assigned to the volume. AUTO, ZONE_1, ZONE_2, or ZONE_3