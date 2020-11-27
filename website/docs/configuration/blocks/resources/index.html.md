---
layout: "language"
page_title: "Resources Overview - Configuration Language"
---

# Resources

> **Hands-on:** Try the [Terraform: Get Started](https://learn.hashicorp.com/collections/terraform/aws-get-started?utm_source=WEBSITE&utm_medium=WEB_IO&utm_offer=ARTICLE_PAGE&utm_content=DOCS) collection on HashiCorp Learn.

_Resources_ are the most important element in the Terraform language.
Each resource block describes one or more infrastructure objects, such
as virtual networks, compute instances, or higher-level components such
as DNS records.

- [Resource Blocks](/docs/configuration/blocks/resources/syntax.html) documents
  the syntax for declaring resources.

- [Resource Behavior](/docs/configuration/resources/behavior.html) explains in
  more detail how Terraform handles resource declarations when applying a
  configuration.

- The Meta-Arguments section documents special arguments that can be used with
  every resource type, including
  [`depends_on`](/docs/configuration/meta-arguments/depends_on.html),
  [`count`](/docs/configuration/meta-arguments/count.html),
  [`for_each`](/docs/configuration/meta-arguments/for_each.html),
  [`provider`](/docs/configuration/meta-arguments/resource-provider.html),
  and [`lifecycle`](/docs/configuration/meta-arguments/lifecycle.html).

- [Provisioners](/docs/configuration/blocks/resources/provisioners/index.html)
  documents configuring post-creation actions for a resource using the
  `provisioner` and `connection` blocks. Since provisioners are non-declarative
  and potentially unpredictable, we strongly recommend that you treat them as a
  last resort.
