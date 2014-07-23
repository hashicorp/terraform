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

While our infrastructure is simple, viewing the execution plan
of a destroy can be useful to make sure that it is destroying
only the resources you expect.

To ask Terraform to create an execution plan to destroy all
infrastructure, run the plan command with the `-destroy` flag.

```
$ terraform plan -destroy
...

- aws_instance.example
```

The output says that "aws\_instance.example" will be deleted.

The `-destroy` flag lets you destroy infrastructure without
modifying the configuration. You can also destroy infrastructure
by simply commenting out or deleting the contents of your
configuration, but usually you just want to destroy an instance
of your infrastructure rather than permanently deleting your
configuration as well. The `-destroy` flag is for this case.

## Apply

Let's apply the destroy:

```
$ terraform apply -destroy
aws_instance.example: Destroying...

Apply complete! Resources: 0 added, 0 changed, 1 destroyed.

...
```

Done. Terraform destroyed our one instance, and if you run a
`terraform show`, you'll see that the state file is now empty.

## Next

You now know how to create, modify, and destroy infrastructure.
With these building blocks, you can effectively experiment with
any part of Terraform.

Next, we move on to features that make Terraform configurations
slightly more useful: variables, resource dependencies, provisioning,
and more.
