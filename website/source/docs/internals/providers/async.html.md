---
layout: "docs"
page_title: "Working with Async APIs"
sidebar_current: "docs-internals-provider-guide-async"
description: |-
  Tips for writing resources and data sources that consume an asynchronous API.
---

# Working with Async APIs

Some resources may take time to create. The convention and promise of Terraform
is to only consider resource creation or update as complete when these pending
operations are finished. Usually, APIs return one or more fields that allow
inspection of the state of the resource during its creation. The same principle
applies to deletion.

This means Terraform intentionally lets the user wait rather than provide
resources that may not be ready for use yet by the user or some other dependent
resources or provisioners.

There are two helpers to make the work with such APIs easier.

## Working with `StateChangeConf`

The
[`resource.StateChangeConf`](https://godoc.org/github.com/hashicorp/terraform/helper/resource#StateChangeConf)
type's `WaitForState` function waits for a string to match a specified value,
using the
[`resource.StateRefreshFunc`](https://godoc.org/github.com/hashicorp/terraform/helper/resource#StateRefreshFunc)
to update the current value of the string. It does exponential backoff by
default to avoid throttling and it can also deal with inconsistent or
eventually consistent APIs to certain extent.

## Working with `Retry`

The
[`resource.Retry`](https://godoc.org/github.com/hashicorp/terraform/helper/resource#Retry)
function is just a wrapper around the `resource.StateChangeConf` function.
It's useful when the API only reports success or failure, and the request
should be retried when it fails.
