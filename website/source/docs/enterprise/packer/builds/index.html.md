---
layout: "enterprise"
page_title: "Packer Builds - Terraform Enterprise"
sidebar_current: "docs-enterprise-packerbuilds"
description: |-
  Builds are instances of `packer build` being run within Terraform Enterprise.
---

# About Builds

Builds are instances of `packer build` being run within Terraform Enterprise.
Every build belongs to a build configuration.

__Build configurations__ represent a set of Packer configuration versions and
builds run. It is used as a namespace within Terraform Enterprise, Packer
commands and URLs. Packer configuration sent to Terraform Enterprise are stored
and versioned under these build configurations.

These __versions__ of Packer configuration can contain:

- The Packer template, a JSON file which define one or more builds by
  configuring the various components of Packer

- Any provisioning scripts or packages used by the template

- Applications that use the build as part of the pipeline and merged into the
  version prior to running Packer on it

When a new version of Packer configuration and associated scripts from GitHub or
`packer push` is received, it automatically starts a new Packer build. That
Packer build runs in an isolated machine environment with the contents of that
version available to it.

You can be alerted of build events with
[Build Notifications](/docs/enterprise/packer/builds/notifications.html).
