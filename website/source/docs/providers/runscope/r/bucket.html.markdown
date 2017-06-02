---
layout: "runscope"
page_title: "Runscope: runscope_bucket"
sidebar_current: "docs-runscope-resource-bucket"
description: |-
  Provides a Runscope bucket resource.
---

# runscope\_bucket

A [bucket](https://www.runscope.com/docs/api/buckets) resource.
[Buckets](https://www.runscope.com/docs/buckets) are a simple way to
organize your requests and tests.

## Example Usage

```hcl
# Add a bucket to your runscope account
resource "runscope_bucket" "main" {
  name      = "a-bucket"
  team_uuid = "870ed937-bc6e-4d8b-a9a5-d7f9f2412fa3"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (String, Required) The name of this bucket.
* `team_uuid` - (String, Required) Unique identifier for the team this bucket
  is being created for.

## Attributes Reference

The following attributes are exported:

* `name` - The name of this bucket.
* `team_uuid` - Unique identifier for the team this bucket belongs to.