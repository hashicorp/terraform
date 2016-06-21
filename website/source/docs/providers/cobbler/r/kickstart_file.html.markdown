---
layout: "cobbler"
page_title: "Cobbler: cobbler_kickstart_file"
sidebar_current: "docs-cobbler-resource-kickstart_file"
description: |-
  Manages a Kickstart File within Cobbler.
---

# cobbler\_kickstart\_file

Manages a Kickstart File within Cobbler.

## Example Usage

```
resource "cobbler_kickstart_file" "my_kickstart" {
  name = "/var/lib/cobbler/kickstarts/my_kickstart.ks"
  body = "<content of kickstart file>"
}
```

## Argument Reference

The following arguments are supported:

* `body` - (Required) The body of the kickstart file.

* `name` - (Required) The name of the kickstart file. This must be
  the full path, including `/var/lib/cobbler/kickstarts`.
