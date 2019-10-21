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

## Destroy

Resources can be destroyed using the `terraform destroy` command, which is
similar to `terraform apply` but it behaves as if all of the resources have
been removed from the configuration.

```
$ terraform destroy
# ...

  - aws_instance.example
```

The `-` prefix indicates that the instance will be destroyed. As with apply,
Terraform shows its execution plan and waits for approval before making any
changes.

Answer `yes` to execute this plan and destroy the infrastructure:

```
# ...
aws_instance.example: Destroying...

Destroy complete! Resources: 1 destroyed.

# ...
```

Just like with `apply`, Terraform determines the order in which
things must be destroyed. In this case there was only one resource, so no
ordering was necessary. In more complicated cases with multiple resources,
Terraform will destroy them in a suitable order to respect dependencies,
as we'll see later in this guide.

## Next

You now know how to create, modify, and destroy infrastructure
from a local machine.

Next, we move on to features that make Terraform configurations
slightly more useful: [variables, resource dependencies, provisioning,
and more](/intro/getting-started/dependencies.html).
