---
layout: "docs"
page_title: "Writing and Modifying Code - Terraform CLI"
---

# Writing and Modifying Terraform Code

The [Terraform language](/docs/language/index.html) is Terraform's primary
user interface, and all of Terraform's workflows rely on configurations written
in the Terraform language.

Terraform CLI includes several commands to make Terraform code more convenient
to work with. Integrating these commands into your editing workflow can
potentially save you time and effort.

- [The `terraform console` command](/docs/cli/commands/console.html) starts an
  interactive shell for evaluating Terraform
  [expressions](/docs/language/expressions/index.html), which can be a faster way
  to verify that a particular resource argument results in the value you expect.


- [The `terraform fmt` command](/docs/cli/commands/fmt.html) rewrites Terraform
  configuration files to a canonical format and style, so you don't have to
  waste time making minor adjustments for readability and consistency. It works
  well as a pre-commit hook in your version control system.

- [The `terraform validate` command](/docs/cli/commands/validate.html) validates the
  syntax and arguments of the Terraform configuration files in a directory,
  including argument and attribute names and types for resources and modules.
  The `plan` and `apply` commands automatically validate a configuration before
  performing any other work, so `validate` isn't a crucial part of the core
  workflow, but it can be very useful as a pre-commit hook or as part of a
  continuous integration pipeline.

- [The `0.13upgrade` command](/docs/cli/commands/0.13upgrade.html) and
  [the `0.12upgrade` command](/docs/cli/commands/0.12upgrade.html) can automatically
  modify the configuration files in a Terraform module to help deal with major
  syntax changes that occurred in the 0.13 and 0.12 releases of Terraform. Both
  of these commands are only available in the Terraform version they are
  associated with, and you are expected to upgrade older code to be compatible
  with 0.12 before attempting to make it compatible with 0.13. For more detailed
  information about updating code for new Terraform versions, see the [upgrade
  guides](/upgrade-guides/index.html) in the Terraform language docs.
