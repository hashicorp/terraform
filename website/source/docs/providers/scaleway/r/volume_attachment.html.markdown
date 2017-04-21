---
layout: "scaleway"
page_title: "Scaleway: volume attachment"
sidebar_current: "docs-scaleway-resource-volume attachment"
description: |-
  Manages Scaleway Volume attachments for servers.
---

# scaleway\_volume\_attachment

This allows volumes to be attached to servers.

**Warning:** Attaching volumes requires the servers to be powered off. This will lead
to downtime if the server is already in use.

## Example Usage

```hcl
resource "scaleway_server" "test" {
  name  = "test"
  image = "aecaed73-51a5-4439-a127-6d8229847145"
  type  = "C2S"
}

resource "scaleway_volume" "test" {
  name       = "test"
  size_in_gb = 20
  type       = "l_ssd"
}

resource "scaleway_volume_attachment" "test" {
  server = "${scaleway_server.test.id}"
  volume = "${scaleway_volume.test.id}"
}
```

## Argument Reference

The following arguments are supported:

* `server` - (Required) id of the server
* `volume` - (Required) id of the volume to be attached

## Attributes Reference

The following attributes are exported:

* `id` - id of the new resource
