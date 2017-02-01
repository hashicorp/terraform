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

```
resource "profitbricks_volume" "example" {
  datacenter_id = "${profitbricks_datacenter.example.id}"
  server_id = "${profitbricks_server.example.id}"
  image_name = "${var.ubuntu}"
  size = 5
  disk_type = "HDD"
  sshkey_path = "${var.private_key_path}"
  bus = "VIRTIO"
}
```

##Argument reference

* `datacenter_id` - (Required) [string] <sup>[1](#myfootnote1)</sup>
* `server_id` - (Required)[string] <sup>[1](#myfootnote1)</sup>
* `disk_type` - (Required) [string] The volume type, HDD or SSD.
* `bus` - (Required) [boolean] The bus type of the volume.
* `size` -  (Required)[integer] The size of the volume in GB.
* `image_password` - [string] Required if `sshkey_path` is not provided.
* `image_name` - [string] The image or snapshot ID. It is required if `licence_type` is not provided.
* `licence_type` - [string] Required if `image_name` is not provided.
* `name` - (Optional) [string] The name of the volume.