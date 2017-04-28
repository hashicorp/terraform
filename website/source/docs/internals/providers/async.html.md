---
layout: "docs"
page_title: "Working With Asynchronous APIs"
sidebar_current: "docs-internals-provider-guide-async"
description: |-
  Tips for writing resources and data sources that consume an asynchronous API.
---

# Working with Asynchronous APIs
Some resources may take time to create—it may take 30 seconds to spin up an instance, or 20 minutes to provision a hosted database instance like RDS. The convention and promise of Terraform is to only consider resource creation or update as done when these pending operations are actually done. Often the API returns one or more fields that allow you to inspect the state of the resource during its creation. The same principle applies to deletion.

This means we intentionally let the user wait rather than provide resources that may not be ready for use yet by the user or some other dependent resources or provisioners.

There are two helpers to make the work with such APIs easier.

## Working with `StateChangeConf`
The `StateChangeConf` function in the `helper/resource` package allows you to watch an object and wait for it to achieve the state specified in the configuration using the `Refresh()` function. It does exponential backoff by default to avoid throttling and it can also deal with inconsistent or eventually consistent APIs to certain extent.

## Working with `Retry`
The `Retry` function in the `helper/resource` package is just a wrapper around the `StateChangeConf` function. It’s useful for situations where you only need to retry, and there are only two states the resource can be in, based on the API response (success or failure).
