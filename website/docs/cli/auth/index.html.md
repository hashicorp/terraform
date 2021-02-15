---
layout: "docs"
page_title: "Authentication - Terraform CLI"
---

# CLI Authentication

> **Hands-on:** Try the [Authenticate the CLI with Terraform Cloud](https://learn.hashicorp.com/tutorials/terraform/cloud-login?in=terraform/cloud&utm_source=WEBSITE&utm_medium=WEB_IO&utm_offer=ARTICLE_PAGE&utm_content=DOCS) tutorial on HashiCorp Learn.

[Terraform Cloud](/docs/cloud/index.html) and
[Terraform Enterprise](/docs/enterprise/index.html) are platforms that perform
Terraform runs to provision infrastructure, offering a collaboration-focused
environment that makes it easier for teams to use Terraform together. (For
expediency, the content below refers to both products as "Terraform Cloud.")

Terraform CLI integrates with Terraform Cloud in several ways — it can be a
front-end for [CLI-driven runs](/docs/cloud/run/cli.html) in Terraform Cloud,
and can also use Terraform Cloud as a state backend and a private module
registry. All of these integrations require you to authenticate Terraform CLI
with your Terraform Cloud account.

The best way to handle CLI authentication is with the `login` and `logout`
commands, which help automate the process of getting an API token for your
Terraform Cloud user account.

For details, see:

- [The `terraform login` command](/docs/cli/commands/login.html)
- [The `terraform logout` command](/docs/cli/commands/logout.html)
