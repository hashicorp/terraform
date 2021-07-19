---
layout: "language"
page_title: "Resources Overview - Configuration Language"
description: "`resources` describe infrastructure objects in Terraform configurations. Find documentation for resource syntax, behavior, and meta-arguments."
---

# Resources

> **Hands-on:** Try the [Terraform: Get Started](https://learn.hashicorp.com/collections/terraform/aws-get-started?utm_source=WEBSITE&utm_medium=WEB_IO&utm_offer=ARTICLE_PAGE&utm_content=DOCS) collection on HashiCorp Learn.

_Resources_ are the most important element in the Terraform language.
Each resource block describes one or more infrastructure objects, such
as virtual networks, compute instances, or higher-level components such
as DNS records.

- [Resource Blocks](/docs/language/resources/syntax.html) documents
  the syntax for declaring resources.

- [Resource Behavior](/docs/language/resources/behavior.html) explains in
  more detail how Terraform handles resource declarations when applying a
  configuration.

- The Meta-Arguments section documents special arguments that can be used with
  every resource type, including
  [`depends_on`](/docs/language/meta-arguments/depends_on.html),
  [`count`](/docs/language/meta-arguments/count.html),
  [`for_each`](/docs/language/meta-arguments/for_each.html),
  [`provider`](/docs/language/meta-arguments/resource-provider.html),
  and [`lifecycle`](/docs/language/meta-arguments/lifecycle.html).

- [Provisioners](/docs/language/resources/provisioners/index.html)
  documents configuring post-creation actions for a resource using the
  `provisioner` and `connection` blocks. Since provisioners are non-declarative
  and potentially unpredictable, we strongly recommend that you treat them as a
  last resort.
