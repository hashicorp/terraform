---
layout: "logentries"
page_title: "Logentries: logentries_logset"
sidebar_current: "docs-logentries-logset"
description: |-
  Creates a Logentries logset.
---

# logentries\_logset

Provides a Logentries logset resource. A logset is a collection of `logentries_log` resources.

## Example Usage

```hcl
# Create a log set
resource "logentries_logset" "host_logs" {
  name     = "${var.server}-logs"
  location = "www.example.com"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The log set name, which should be short and descriptive. For example, www, db1.
* `location` - (Optional, default "nonlocation") A location is for your convenience only. You can specify a DNS entry such as web.example.com, IP address or arbitrary comment.
