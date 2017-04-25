---
layout: "enterprise"
page_title: "Rebuilding - Packer Builds - Terraform Enterprise"
sidebar_current: "docs-enterprise-packerbuilds-rebuilding"
description: |-
  Sometimes builds fail due to temporary or remotely controlled conditions.
---

# Rebuilding Builds

Sometimes builds fail due to temporary or remotely controlled conditions.

In this case, it may make sense to "rebuild" a Packer build. To do so, visit the
build you wish to run again and click the Rebuild button. This will take that
exact version of configuration and run it again.

You can rebuild at any point in history, but this may cause side effects that
are not wanted. For example, if you were to rebuild an old version of a build,
it may create the next version of an artifact that is then released, causing a
rollback of your configuration to occur.
