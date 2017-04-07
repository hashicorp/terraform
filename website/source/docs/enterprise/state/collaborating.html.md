---
layout: "enterprise"
page_title: "Collaborating - State - Terraform Enterprise"
sidebar_current: "docs-enterprise-state-collaborating"
description: |-
  How to collaborate on states.
---

# Collaborating on Terraform Remote State

Terraform Enterprise is one of a few options to store [remote state](/docs/enterprise/state).

Remote state gives you the ability to version and collaborate on Terraform
changes. It stores information about the changes Terraform makes based on
configuration.

In order to collaborate safely on remote state, we recommend
[creating an organization](/docs/enterprise/organizations/create.html) to
manage teams of users.

Then, following a [remote state push](/docs/enterprise/state) you can view state
versions in the changes tab of the environment created under the same name as
the remote state.
