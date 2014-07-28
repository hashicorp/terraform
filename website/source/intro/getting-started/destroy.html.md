---
layout: "intro"
page_title: "Destroy Infrastructure"
sidebar_current: "gettingstarted-destroy"
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

For Terraform to destroy our infrastructure, we need to ask
Terraform to generate a destroy execution plan. This is a special
kind of execution plan that only destroys all Terraform-managed
infrastructure, and doesn't create or update any components.

```
$ terraform plan -destroy -out=terraform.tfplan
...

- aws_instance.example
```

The plan command is given two new flags.

The first flag, `-destroy` tells Terraform to create an execution
plan to destroy the infrastructure. You can see in the output that
our one EC2 instance will be destroyed.

The second flag, `-out` tells Terraform to save the execution plan
to a file. We haven't seen this before, but it isn't limited to
only destroys. Any plan can be saved to a file. Terraform can then
apply a plan, ensuring that only exactly the plan you saw is executed.
For destroys, you must save into a plan, since there is no way to
tell `apply` to destroy otherwise.

## Apply

Let's apply the destroy:

```
$ terraform apply terraform.tfplan
aws_instance.example: Destroying...

Apply complete! Resources: 0 added, 0 changed, 1 destroyed.

...
```

Done. Terraform destroyed our one instance, and if you run a
`terraform show`, you'll see that the state file is now empty.

For this command, we gave an argument to `apply` for the first
time. You can give apply a specific plan to execute.

## Next

You now know how to create, modify, and destroy infrastructure.
With these building blocks, you can effectively experiment with
any part of Terraform.

Next, we move on to features that make Terraform configurations
slightly more useful: [variables, resource dependencies, provisioning,
and more](/intro/getting-started/dependencies.html).
