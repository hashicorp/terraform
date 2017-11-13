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
however in production environments it is more responsible to run
Terraform remotely and store a master Terraform state remotely.

Terraform supports a feature known as [remote backends](/docs/backends)
to support this. Backends are the recommended way to use Terraform in
a team environment.

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
credentials off of developer machines and releasing local machines
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

HashiCorp (the makers of Terraform) also provide a commercial solution which
functions as a Terraform backend as well as enabling many other features such
as remote apply, run history, state history, state diffing, and more.

This section will guide you through a demo of Terraform Enterprise. Note that
this is commercial software. If you are not interested at this time, you may
skip this section.

First, [create an account here](https://atlas.hashicorp.com/account/new?utm_source=oss&utm_medium=getting-started&utm_campaign=terraform) unless you already have one.

Terraform uses your access token to securely communicate with Terraform
Enterprise. To generate a token: select your username in the left side
navigation menu, click "Accounts Settings", "click "Tokens", then click
"Generate".

For the purposes of this tutorial you can use this token by exporting it to
your local shell session:

```
$ export ATLAS_TOKEN=ATLAS_ACCESS_TOKEN
```

Replace `ATLAS_ACCESS_TOKEN` with the token generated earlier. Next,
configure the Terraform Enterprise backend:

```hcl
terraform {
  backend "atlas" {
    name = "USERNAME/getting-started"
  }
}
```

Replace `USERNAME` with your Terraform Enterprise username. Note that the
backend name is "atlas" for legacy reasons and will be renamed soon.

Remember to run `terraform init`. At this point, Terraform is using Terraform
Enterprise for everything shown before with Consul. Next, we'll show you some
additional functionality Terraform Enterprise enables.

Before you [push](/docs/commands/push.html) your Terraform configuration to
Terraform Enterprise you'll need to start a local version control system with
at least one commit. Here is an example using `git`.

```
$ git init
$ git add example.tf
$ git commit -m "init commit"
```

Next, [push](/docs/commands/push.html) your Terraform configuration:

```
$ terraform push
```

This will automatically trigger a `terraform plan`, which you can
review in the [Terraform page](https://atlas.hashicorp.com/terraform).
If the plan looks correct, hit "Confirm & Apply" to execute the
infrastructure changes.

Running Terraform in Terraform Enterprise creates a complete history of
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

We've now concluded the getting started guide, however
there are a number of [next steps](/intro/getting-started/next-steps.html)
to get started with Terraform.
