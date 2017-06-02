---
layout: "runscope"
page_title: "Runscope: runscope_test"
sidebar_current: "docs-runscope-resource-test"
description: |-
  Provides a Runscope test resource.
---

# runscope\_test

A [test](https://www.runscope.com/docs/api/tests) resource.
[Tests](https://www.runscope.com/docs/buckets) are made up of
a collection of [test steps](test_steps.html) and an
[environment](environment.html).

## Example Usage

```hcl
# Add a test to a bucket
resource "runscope_test" "api" {
  name         = "api-test"
  description  = "checks the api is up and running"
  bucket_id    = "${runscope_bucket.main}"
}

# Create a bucket
resource "runscope_bucket" "main" {
  name         = "terraform-ftw"
  team_uuid    = "870ed937-bc6e-4d8b-a9a5-d7f9f2412fa3"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (String, Required) The name of this test.
* `description` - (String, Optional) Human-readable description of the new test.
  is being created for.

## Attributes Reference

The following attributes are exported:

* `id` - The unique identifier for the test.
* `name` - The name of this test.
* `description` - Human-readable description of the new test.
