---
layout: "docs"
page_title: "Resources vs. Data Sources"
sidebar_current: "docs-internals-provider-guide-resources-v-data-sources"
description: |-
  Understanding whether to use a resource or a data source.
---

# Resources vs Data Sources
Resources and Data Sources are incredibly similar, and for the most part, can be treated as working the same&mdash;they’re implemented using the same type, `*Resource`. An easy shorthand is to think of Data Sources as read-only Resources. If you find yourself creating a Resource that only has useful behaviour in the Read function, it should probably be a Data Source.

Data Sources shouldn’t be considered part of provisioning an environment&mdash;Terraform will make no effort to ensure they exist before it tries to reference them, and it doesn’t have any concept of how they should be configured. They’re simply a way to allow configurations to reference information that another tool&mdash;your CI tool, your cloud provider, etc.&mdash;owns.
