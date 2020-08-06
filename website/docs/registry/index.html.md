---
layout: "registry"
page_title: "Terraform Registry"
sidebar_current: "docs-registry-home"
description: |-
  The Terraform Registry is a repository of providers and modules written by the Terraform community.
---

# Terraform Registry

The [Terraform Registry](https://registry.terraform.io) is an interactive resource for discovering a wide selection of integrations (Providers) and configuration packages (Modules) for use with Terraform. The Registry includes solutions developed by HashiCorp, Third-party vendors, and those created by our Terraform community. The Registry aims to connect our users with solutions, and to help new users get started with Terraform more quickly, by sharing examples of how Terraform is written, and find pre-made modules for infrastructure components you require.

![screenshot: terraform registry landing page](./images/registry1.png)

The Terraform Registry is integrated [directly into Terraform](https://www.terraform.io/docs/configuration/providers.html) to make consuming Providers and modules easy. Anyone can publish both Providers and Modules on the Registry – you may use the [Public Registry](https://registry.terraform.io) for viewing and publishing public providers and modules; For private modules, you can use a [Private Registry](https://www.terraform.io/docs/registry/private.html), or [reference repositories and other sources directly](https://www.terraform.io/docs/modules/sources.html).

Use the navigation to the left to learn more about using the registry.

## Navigating the Registry

As the Terraform Ecosystem continues to grow, the Registry is designed to make it easy to discover and search through integrations and solutions across dozens of categories. Select a Provider or Module card to learn more, use filters to select the tier (see tiers), or use the search at the top of the Registry to find what you’re looking for. Note that search supports keyboard navigation:

![screenshot: terraform registry browse](./images/registry2.png)

## User Account

Anyone interested in publishing a Provider or Module can create an account and sign in to the Terraform Registry using a GitHub account. Choose Sign-in, and follow the login prompts. Once you have authorized the use of your GitHub account and are signed in, you are able to publish both Providers and Modules, directly from one of the Repositories you manage. To learn more, see [Publishing to the Registry](https://www.terraform.io/docs/registry/providers/publishing.html).

![screenshot: terraform registry sign in](./images/user-account.png)

## Getting Help

We welcome any feedback you have throughout the process and encourage you to reach out if you have any questions or issues with the Terraform Registry by sending us an [email](mailto:terraform-registry-beta@hashicorp.com). The providers and modules in The Terraform Registry are published and maintained either directly by HashiCorp, by trusted HashiCorp partners and the Terraform Community ([see tiers & namespaces](./providers/overview.html#provider-tiers-amp-namespaces)). If you run into issues or have additional contributions to make to a provider or module, you can submit a GitHub issue by selecting the "Report an issue" link on the detail view:

![Provider report issue link](./images/registry-issue.png)
