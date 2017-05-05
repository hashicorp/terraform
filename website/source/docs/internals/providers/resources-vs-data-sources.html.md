---
layout: "docs"
page_title: "Resources vs. Data Sources"
sidebar_current: "docs-internals-provider-guide-resources-v-data-sources"
description: |-
  Understanding whether to use a resource or a data source.
---

# Resources vs Data Sources

Resources and data sources are very similar, and for the most part, can be
treated as working the same - they're implemented using the same type,
[`*schema.Resource`](https://godoc.org/github.com/hashicorp/terraform/helper/schema#Resource).
An easy shorthand is to think of data sources as read-only resources. Any
resource that only has useful behaviour in the `schema.ReadFunc` function
should probably be a data source.

Data sources shouldn't be considered part of provisioning an environment -
Terraform will make no effort to ensure they exist before it tries to reference
them, and it doesn't have any concept of how they should be configured. They're
simply a way to allow configurations to reference information that another tool
- your CI tool, your cloud provider, etc. - owns.
