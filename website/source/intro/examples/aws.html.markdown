---
layout: "intro"
page_title: "Basic Two-Tier AWS Architecture"
sidebar_current: "examples-aws"
---

# Basic Two-Tier AWS Architecture

This provides a template for running a simple two-tier architecture on Amazon
Web services.

The basic premise is you have stateless app servers running behind
and ELB serving traffic. State for your application is stored in an RDS
database.

This ignores deploying and getting data onto the application
servers intentionally to simplify. However, you could do so either via
[provisioners](/docs/provisioners/index.html) or by pre-baking configured
AMIs with [Packer](http://www.packer.io).

## Configuration

```
FOOBAR
```
