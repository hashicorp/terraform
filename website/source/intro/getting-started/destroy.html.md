---
layout: "intro"
page_title: "Destroy Infrastructure"
sidebar_current: "gettingstarted-destroy"
description: |-
  We've now seen how to build and change infrastructure. Before we move on to creating multiple resources and showing resource dependencies, we're going to go over how to completely destroy the Terraform-managed infrastructure.
---

# Destroy Infrastructure

We've now seen how to build and change infrastructure. Before we
move on to creating multiple resources and showing resource
dependencies, we're going to go over how to completely destroy
the Terraform-managed infrastructure.

Destroying your infrastructure is a rare event in production
environments. But if you're using Terraform to spin up multiple
environments such as development, test, QA environments, then
destroying is a useful action.

## Plan

Before destroying our infrastructure, we can use the plan command
to see what resources Terraform will destroy.

```
$ terraform plan -destroy
# ...

- aws_instance.example
```

With the `-destroy` flag, we're asking Terraform to plan a destroy,
where all resources under Terraform management are destroyed. You can
use this output to verify exactly what resources Terraform is managing
and will destroy.

## Destroy

Let's destroy the infrastructure now:

```
$ terraform destroy
aws_instance.example: Destroying...

Apply complete! Resources: 0 added, 0 changed, 1 destroyed.

# ...
```

The `terraform destroy` command should ask you to verify that you
really want to destroy the infrastructure. Terraform only accepts the
literal "yes" as an answer as a safety mechanism. Once entered, Terraform
will go through and destroy the infrastructure.

Just like with `apply`, Terraform is smart enough to determine what order
things should be destroyed. In our case, we only had one resource, so there
wasn't any ordering necessary. But in more complicated cases with multiple
resources, Terraform will destroy in the proper order.

## Next

You now know how to create, modify, and destroy infrastructure
from a local machine.

Next, we move on to features that make Terraform configurations
slightly more useful: [variables, resource dependencies, provisioning,
and more](/intro/getting-started/dependencies.html).
