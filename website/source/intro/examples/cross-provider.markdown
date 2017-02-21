---
layout: "intro"
page_title: "Cross Provider"
sidebar_current: "examples-cross-provider"
description: |-
  An example of the cross-provider capabilities of Terraform.
---

# Cross Provider Example

[**Example Source Code**](https://github.com/hashicorp/terraform/tree/master/examples/cross-provider)

This is a simple example of the cross-provider capabilities of
Terraform.

This creates a Heroku application and points a DNS
CNAME record at the result via DNSimple. A `host` query to the outputted
hostname should reveal the correct DNS configuration.

As with all the examples, just copy and paste the example and run
`terraform apply` to see it work.
