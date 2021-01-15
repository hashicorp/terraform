---
layout: "language"
page_title: "Syntax Overview - Configuration Language"
---

# Syntax

The majority of the Terraform language documentation focuses on the practical
uses of the language and the specific constructs it uses. The pages in this
section offer a more abstract view of the Terraform language.

- [Configuration Syntax](/docs/configuration/syntax.html) describes the native
  grammar of the Terraform language.
- [JSON Configuration Syntax](/docs/configuration/syntax-json.html) documents
  how to represent Terraform language constructs in the pure JSON variant of the
  Terraform language. Terraform's JSON syntax is unfriendly to humans, but can
  be very useful when generating infrastructure as code with other systems that
  don't have a readily available HCL library.
- [Style Conventions](/docs/configuration/style.html) documents some commonly
  accepted formatting guidelines for Terraform code. These conventions can be
  enforced automatically with [`terraform fmt`](/docs/commands/fmt.html).
