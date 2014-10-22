---
layout: "docs"
page_title: "Plugins"
sidebar_current: "docs-plugins"
description: |-
  Terraform is built on a plugin-based architecture. All providers and provisioners that are used in Terraform configurations are plugins, even the core types such as AWS and Heroku. Users of Terraform are able to write new plugins in order to support new functionality in Terraform.
---

# Plugins

Terraform is built on a plugin-based architecture. All providers and
provisioners that are used in Terraform configurations are plugins, even
the core types such as AWS and Heroku. Users of Terraform are able to
write new plugins in order to support new functionality in Terraform.

This section of the documentation gives a high-level overview of how
to write plugins for Terraform. It does not hold your hand through the
process, however, and expects a relatively high level of understanding
of Go, provider semantics, Unix, etc.

~> **Advanced topic!** Plugin development is a highly advanced
topic in Terraform, and is not required knowledge for day-to-day usage.
If you don't plan on writing any plugins, we recommend not reading
this section of the documentation.
