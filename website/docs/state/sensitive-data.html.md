---
layout: "docs"
page_title: "State: Sensitive Data"
sidebar_current: "docs-state-sensitive-data"
description: |-
  Sensitive data in Terraform state.
---

# Sensitive Data in State

Terraform state can contain sensitive data depending on the resources in-use
and your definition of "sensitive." The state contains resource IDs and all
resource attributes. For resources such as databases, this may contain initial
passwords.

When using local state, state is stored in plain-text JSON files. When
using [remote state](/docs/state/remote.html), state is only ever held in memory when used by Terraform.
It may be encrypted at rest but this depends on the specific remote state
backend.

It is important to keep this in mind if you do (or plan to) store sensitive
data (e.g. database passwords, user passwords, private keys) as it may affect
the risk of exposure of such sensitive data.

## Recommendations

Storing state remotely may provide you encryption at rest depending on the
backend you choose. As of Terraform 0.9, Terraform will only hold the state
value in memory when remote state is in use. It is never explicitly persisted
to disk.

For example, encryption at rest can be enabled with the S3 backend and IAM
policies and logging can be used to identify any invalid access. Requests for
the state go over a TLS connection.

[Terraform Cloud](https://www.hashicorp.com/products/terraform/) is
a commercial product from HashiCorp that also acts as a [backend](/docs/backends)
and provides encryption at rest for state. Terraform Cloud also knows
the identity of the user requesting state and maintains a history of state
changes. This can be used to provide access control and detect any breaches.

## Future Work

Long term, the Terraform project wants to further improve the ability to
secure sensitive data. There are plans to provide a
generic mechanism for specific state attributes to be encrypted or even
completely omitted from the state. These do not exist yet except on a
resource-by-resource basis if documented.
