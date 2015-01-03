---
layout: "intro"
page_title: "Basic Two-Tier AWS Architecture"
sidebar_current: "examples-aws"
description: |-
  This provides a template for running a simple two-tier architecture on Amazon Web services. The premise is that you have stateless app servers running behind an ELB serving traffic.
---

# Basic Two-Tier AWS Architecture

[**Example Contents**](https://github.com/hashicorp/terraform/tree/master/examples/aws-two-tier)

This provides a template for running a simple two-tier architecture on Amazon
Web services. The premise is that you have stateless app servers running behind
an ELB serving traffic.

To simplify the example, this intentionally ignores deploying and
getting your application onto the servers. However, you could do so either via
[provisioners](/docs/provisioners/index.html) and a configuration
management tool, or by pre-baking configured AMIs with
[Packer](https://www.packer.io).

After you run `terraform apply` on this configuration, it will
automatically output the DNS address of the ELB. After your instance
registers, this should respond with the default nginx web page.

As with all examples, just copy and paste the example and run
`terraform apply` to see it work.
