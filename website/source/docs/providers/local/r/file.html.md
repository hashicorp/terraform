---
layout: "local"
page_title: "Local: local_file"
sidebar_current: "docs-local-resource-file"
description: |-
  Generates a local file from content.
---

# local\_file

Generates a local file from a given content.

## Example Usage

```
data "local_file" "foo" {
    content     = "foo!"
    filename = "${path.module}/foo.bar"
}
```

## Argument Reference

The following arguments are supported:

* `content` - (required) The content of file to create.

* `filename` - (required) The path of the file to create.

NOTE: Any required parent folders are created automatically. Additionally, any existing file will get overwritten.