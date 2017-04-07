---
layout: "enterprise"
page_title: "Pushing - State - Terraform Enterprise"
sidebar_current: "docs-enterprise-state-pushing"
description: |-
  Pushing remote states.
---

# Pushing Terraform Remote State to Terraform Enterprise

Terraform Enterprise is one of a few options to store [remote state](/docs/enterprise/state).

Remote state gives you the ability to version and collaborate on Terraform
changes. It stores information about the changes Terraform makes based on
configuration.

To use Terraform Enterprise to store remote state, you'll first need to have the
`ATLAS_TOKEN` environment variable set and run the following command.

```shell
$ terraform remote config \
    -backend-config="name=$USERNAME/product"
```
