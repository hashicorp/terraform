---
layout: "intro"
page_title: "Two-Tier AWS Architecture"
sidebar_current: "examples-aws"
description: |-
  This provides a template for running a simple two-tier architecture on Amazon Web services. The premise is that you have stateless app servers running behind an ELB serving traffic.
---

# Two-Tier AWS Architecture

[**Example Source Code**](https://github.com/terraform-providers/terraform-provider-aws/tree/master/examples/two-tier)

This provides a template for running a simple two-tier architecture on Amazon
Web Services. The premise is that you have stateless app servers running behind
an ELB serving traffic.

To simplify the example, it intentionally ignores deploying and
getting your application onto the servers. However, you could do so either via
[provisioners](/docs/provisioners/index.html) and a configuration
management tool, or by pre-baking configured AMIs with
[Packer](https://www.packer.io).

After you run `terraform apply` on this configuration, it will
automatically output the DNS address of the ELB. After your instance
registers, this should respond with the default Nginx web page.

As with all the examples, just copy and paste the example and run
`terraform apply` to see it work.
