---
layout: "intro"
page_title: "Terraform Remote"
sidebar_current: "gettingstarted-remote"
description: |-
  We've now seen how to build, change, and destroy infrastructure from a local machine. However, you can use Atlas by HashiCorp to run Terraform remotely to version and audit the history of your infrastructure.
---

# Why Use Terraform Remotely?
We've now seen how to build, change, and destroy infrastructure
from a local machine. This is great for testing and development,
however in production environments it is more responsible to run
Terraform remotely and store a master Terraform state remotely.

[Atlas](https://atlas.hashicorp.com/?utm_source=oss&utm_medium=getting-started&utm_campaign=terraform)
is HashiCorp's solution for Terraform remote runs and
infrastructure version control. Running Terraform
in Atlas allows teams to easily version, audit, and collaborate
on infrastructure changes. Each proposed change generates
a Terraform plan which can be reviewed and collaborated on as a team.
When a proposed change is accepted, the Terraform logs are stored
in Atlas, resulting in a linear history of infrastructure states to
help with auditing and policy enforcement. Additional benefits to
running Terraform remotely include moving access
credentials off of developer machines and releasing local machines
from long-running Terraform processes.

# How to Use Terraform Remotely
You can learn how to use Terraform remotely with our [interactive tutorial](https://atlas.hashicorp.com/tutorial/terraform/?utm_source=oss&utm_medium=getting-started&utm_campaign=terraform)
or you can follow the outlined steps below.

First, configure [Terraform remote state storage](/docs/commands/remote.html)
with the command:

```
$ terraform remote config -backend-config="name=ATLAS_USERNAME/getting-started"
```

Replace `ATLAS_USERNAME` with your Atlas username. If you don't have one, you can
[create an account here](https://atlas.hashicorp.com/account/new?utm_source=oss&utm_medium=getting-started&utm_campaign=terraform).

Next, [push](/docs/commands/push.html) your Terraform configuration to Atlas with:

```
$ terraform push -name="ATLAS_USERNAME/getting-started"
```

This will automatically trigger a `terraform plan`, which you can
review in the [Environments tab in Atlas](https://atlas.hashicorp.com/environments).
If the plan looks correct, hit "Confirm & Apply" to execute the
infrastructure changes.

# Version Control for Infrastructure
Running Terraform in Atlas creates a complete history of
infrastructure changes, a sort of version control
for infrastructure. Similar to application version control
systems such as Git or Subversion, this makes changes to 
infrastructure an auditable, repeatable,
and collaborative process. With so much relying on the
stability of your infrastructure, version control is a
responsible choice for minimizing downtime.

## Next
You now know how to create, modify, destroy, version, and
collaborate on infrastructure. With these building blocks,
you can effectively experiment with any part of Terraform.

Next, we move on to features that make Terraform configurations
slightly more useful: [variables, resource dependencies, provisioning,
and more](/intro/getting-started/dependencies.html).
