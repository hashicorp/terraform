---
layout: "intro"
page_title: "Terraform Remote"
sidebar_current: "gettingstarted-remote"
description: |-
  We've now seen how to build, change, and destroy infrastructure from a local machine. However, you can use Atlas by HashiCorp to run Terraform remotely to version and audit the history of your infrastructure.
---

# Remote Backends

We've now seen how to build, change, and destroy infrastructure
from a local machine. This is great for testing and development,
but in production environments it is more responsible to share responsibility
for infrastructure. The best way to do this is by running Terraform in a remote
environment with shared access to state.

Terraform supports team-based workflows with a feature known as [remote
backends](/docs/backends). Remote backends allow Terraform to use a shared
storage space for state data, so any member of your team can use Terraform to
manage the same infrastructure.

Depending on the features you wish to use, Terraform has multiple remote
backend options. You could use Consul for state storage, locking, and
environments. This is a free and open source option. You can use S3 which
only supports state storage, for a low cost and minimally featured solution.

[Terraform Enterprise](https://www.hashicorp.com/products/terraform/?utm_source=oss&utm_medium=getting-started&utm_campaign=terraform)
is HashiCorp's commercial solution and also acts as a remote backend.
Terraform Enterprise allows teams to easily version, audit, and collaborate
on infrastructure changes. Each proposed change generates
a Terraform plan which can be reviewed and collaborated on as a team.
When a proposed change is accepted, the Terraform logs are stored,
resulting in a linear history of infrastructure states to
help with auditing and policy enforcement. Additional benefits to
running Terraform remotely include moving access
credentials off of developer machines and freeing local machines
from long-running Terraform processes.

## How to Store State Remotely

First, we'll use [Consul](https://www.consul.io) as our backend. Consul
is a free and open source solution that provides state storage, locking, and
environments. It is a great way to get started with Terraform backends.

We'll use the [demo Consul server](https://demo.consul.io) for this guide.
This should not be used for real data. Additionally, the demo server doesn't
permit locking. If you want to play with [state locking](/docs/state/locking.html),
you'll have to run your own Consul server or use a backend that supports locking.

First, configure the backend in your configuration:

```hcl
terraform {
  backend "consul" {
    address = "demo.consul.io"
    path    = "getting-started-RANDOMSTRING"
    lock    = false
    scheme  = "https"
  }
}
```

Please replace "RANDOMSTRING" with some random text. The demo server is
public and we want to try to avoid overlapping with someone else running
through the getting started guide.

The `backend` section configures the backend you want to use. After
configuring a backend, run `terraform init` to setup Terraform. It should
ask if you want to migrate your state to Consul. Say "yes" and Terraform
will copy your state.

Now, if you run `terraform apply`, Terraform should state that there are
no changes:

```
$ terraform apply
# ...

No changes. Infrastructure is up-to-date.

This means that Terraform did not detect any differences between your
configuration and real physical resources that exist. As a result, Terraform
doesn't need to do anything.
```

Terraform is now storing your state remotely in Consul. Remote state
storage makes collaboration easier and keeps state and secret information
off your local disk. Remote state is loaded only in memory when it is used.

If you want to move back to local state, you can remove the backend configuration
block from your configuration and run `terraform init` again. Terraform will
once again ask if you want to migrate your state back to local.

## Terraform Enterprise

[Terraform Enterprise](https://www.hashicorp.com/products/terraform/?utm_source=oss&utm_medium=getting-started&utm_campaign=terraform) is a commercial solution which combines a predictable and reliable shared run environment with tools to help you work together on Terraform configurations and modules.

Although Terraform Enterprise can act as a standard remote backend to support Terraform runs on local machines, it works even better as a remote run environment. It supports two main workflows for performing Terraform runs:

- A VCS-driven workflow, in which it automatically queues plans whenever changes are committed to your configuration's VCS repo.
- An API-driven workflow, in which a CI pipeline or other automated tool can upload configurations directly.

For a hands-on introduction to Terraform Enterprise, [follow the Terraform Enterprise getting started guide](/docs/enterprise/getting-started/index.html).


## Next
You now know how to create, modify, destroy, version, and
collaborate on infrastructure. With these building blocks,
you can effectively experiment with any part of Terraform.

We've now concluded the getting started guide, however
there are a number of [next steps](/intro/getting-started/next-steps.html)
to get started with Terraform.
