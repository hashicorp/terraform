---
layout: "enterprise"
page_title: "GitHub - VCS Integrations - Terraform Enterprise"
sidebar_current: "docs-enterprise-vcs-github"
description: |-
  GitHub repositories can be integrated with Terraform Enterprise by using push command.
---

# GitHub Integration

GitHub can be used to import Terraform configuration, automatically queuing runs
when changes are merged into a repository's default branch. Additionally, plans
are run when a pull request is created or updated. Terraform Enterprise will
update the pull request with the result of the Terraform plan providing quick
feedback on proposed changes.

## Setup

Terraform Enterprise environments are linked to individual GitHub repositories.
However, a single GitHub repository can be linked to multiple environments
allowing a single set of Terraform configuration to be used across multiple
environments.

Environments can be linked when they're initially created using the New
Environment process. Existing environments can be linked by setting GitHub
details in their **Integrations**.

To link a Terraform Enterprise environment to a GitHub repository, you need
three pieces of information:

- **GitHub repository** - The location of the repository being imported in the
format _username/repository_.

- **GitHub branch** - The branch from which to ingress new versions. This
defaults to the value GitHub provides as the default branch for this repository.

- **Path to directory of Terraform files** - The repository's subdirectory that
contains its terraform files. This defaults to the root of the repository.
