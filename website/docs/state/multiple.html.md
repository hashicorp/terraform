---
layout: "language"
page_title: "State: Multiple States per Configuration"
sidebar_current: "docs-state-multiple"
description: |-
  For unusual situations you can create additional states associated with the same configuration.
---

# Multiple States per Configuration

The normal way to use Terraform is for each configuration to be associated with
exactly one state. In that case, the typical workflow commands treat the
state as largely implicit, reading it before operations and updating it
automatically with the result of operations.

In some unusual cases it can be useful to temporarily create additional states
for a configuration. For example, if you are working on a non-trivial change
to a configuration in a feature branch of your version control system, you
might find it convenient to temporarily create an additional state named after
that branch where you can create a separate set of resources to test your
changes against, without affecting the objects tracked by the main state.

When you use multiple states with your configuration, the state that would
normally have been the _only_ state is always named "default". Other states
you create will each have a different name chosen by you. You cannot delete
the "default" state, but you can create and delete as many other states
as you need.

Terraform stores the state for a configuration using that configuration's
configured [backend](/docs/backends/index.html), and so you can only create
additional states if your chosen backend supports multiple states. Because
additional states belong to the backend, they are visible to anyone else who
works with that same backend configuration.

-> **Note:** Before Terraform v0.15, we used the term "workspace" to refer
to each of the states associated with a configuration. We no longer use that
term because it was confusing with the idea of workspaces used in
[Terraform Cloud](/cloud). In future, Terraform CLI will only use the term
"workspaces" in the sense meant by Terraform Cloud, aside from deprecated
features preserved for backward-compatibility.

## When to use Multiple States

~> **Most users should not use multiple states.** This feature is available
as a pragmatic answer to some unusual use-cases, but the standard Terraform
workflow never requires multiple states associated with the same configuration
and using this feature can add considerable extra complexity to your Terraform
usage.

Current versions of Terraform have support for multiple states primarily for
backward compatibility with existing uses. This mechanism will not see any
further features added and we do not recommend using multiple states in new
systems.

One situation where someone might have used multiple states is to temporarily
create a parallel, distinct "copy" of a set of infrastructure in order to test
a set of changes before modifying the main infrastructure.

In that situation, additional states are commonly associated informally with
feature branches in version control. Similarly, the main state which is always
named "default" typically corresponds with your version control _main_ branch.

Once your candidate change is merged to the main branch and applied to the
primary state, you can destroy the temporary test infrastructure and delete
the temporary state, returning to the stable state of having only one
state.

<!-- TODO: Once we have an officially-recommended pattern for writing
     integration tests for shared modules, this would be a good place to
     recommend that as an alternative to multiple states, because
     each test run will effectively create an implied separate state
     only locally, and then clean it up automatically afterwards. -->

Multiple states are _not_ an appropriate mechanism for decomposing a large
system into smaller parts, because each state associated with a particular
configuration always declares the same set of objects. Instead, you should
write separate Terraform configurations for each component and connect
downstream configurations with upstream configurations using data sources.

Multiple states are also _not_ good mechanism for representing different
long-lived deployment stages or environments. Instead, factor out the common
elements across stages into shared modules and then write a separate
configuration for each deployment stage that has its own separate backend
configuration but calls into the same shared modules.

When considering application deployment it's often recommended to deploy
exactly the same artifact across all deployment stages. That strategy does
not apply to infrastructure management with Terraform, because the equivalent
of an artifact in Terraform is the long-lived object in the remote system
rather than the configuration that declared it, and each deployment stage
should generally have its own distinct set of long-lived objects. Creating
an additional state could be said to correspond with building an application
from a feature branch rather than the main branch, rather than with deploying
the same artifact to two locations.

Where multiple configurations are representing distinct system components
rather than multiple deployment stages, you can pass data from one configuration
to another using paired resources types and data sources. For example:

* Where a shared [Consul](https://consul.io/) cluster is available, use
  [`consul_key_prefix`](/docs/providers/consul/r/key_prefix.html) to
  publish to the key/value store and [`consul_keys`](/docs/providers/consul/d/keys.html)
  to retrieve those values in other configurations.

* In systems that support user-defined labels or tags, use a tagging convention
  to make resources automatically discoverable. For example, use
  [the `aws_vpc` resource type](/docs/providers/aws/r/vpc.html)
  to assign suitable tags and then
  [the `aws_vpc` data source](/docs/providers/aws/d/vpc.html)
  to query by those tags in other configurations.

* For server addresses, use a provider-specific resource to create a DNS
  record with a predictable name and then either use that name directly or
  use [the `dns` provider](/docs/providers/dns/index.html) to retrieve
  the published addresses in other configurations.

* If a Terraform state for one configuration is stored in a remote backend
  that is accessible to other configurations then
  [`terraform_remote_state`](/docs/providers/terraform/d/remote_state.html)
  can be used to directly consume its root module outputs from those other
  configurations. This creates a tighter coupling between configurations,
  but avoids the need for the "producer" configuration to explicitly
  publish its results in a separate system.

## Using Multiple States

A normal Terraform configuration has only a single state. You typically don't
need to worry about the identity of that state, but once you create additional
states the initial one remains the primary state and always has the special
name "default".

You can manage multiple states with some of the subcommands of
`terraform state`. To create a new state and switch to it, you can use
`terraform state new`. For example:

```shellsession
$ terraform state new refactor
Created and selected a new state named "refactor".

You're now using a new, empty state that is separated from the state
you had previously selected. If you run "terraform plan" now, Terraform
will plan to create new objects for all of the resources declared in
your configuration.
```

When you have multiple states, Terraform tracks in your working directory
(using an internal file under the `.terraform` directory) the name of your
currently-selected state. Although the states themselves are shared in the
backend, each working directory can have a different _current_ state, and so
you can switch to a new state while your coworkers continue to work against
the primary state, named "default".

Other commands, like `terraform plan` as mentioned in the command output,
do their work against the currently-selected state. You can switch to another
state that already exists using `terraform state select`. For example, to
switch back to the main state that is always named "default":

```shellsession
$ terraform state select default
Switched to state "default".
```

Each state tracks the same resource addresses but allows each one to be
associated with a different remote object. For example, if you have a
`resource "aws_instance" "example"` block in your configuration then each
of your states will associate `aws_instance.example` with a different
remote EC2 instance.

Once you've finished with the unusual task that prompted you to create an
additional workspace, you should destroy all of the remote objects associated
with it and then delete it with `terraform state delete`:

```shellsession
$ terraform state select refactor
Created and selected a new state named "refactor".
$ terraform destroy
(...all of the usual destroy output)
$ terraform state delete refactor
```

Because each state has its own set of remote objects associated with your
resources, `terraform destroy` will only destroy the objects associated with
resources in the currently-selected state.

The main state, named "default", cannot be deleted.

## Varying Configuration by State Name

Remote systems often require the objects you create to each have a unique
name. Because multiple states associated with the same configuration will cause
Terraform to attempt to create the same objects again, statically-selected
names in the configuration will lead to naming collisions when creating
objects for a non-default workspace.

As a way to help avoid those naming collisions, the Terraform language has
an attribute `terraform.state` which evaluates to the name of the
currently-selected state. A typical way to use it is to refer to it only once
as part of the definition of a local value defining an object name prefix,
like this:

```hcl
locals {
  name_prefix = (
    terraform.state == "default" ?
    "componentname-" :
    "componentname-${terraform.state}-"
  )
}
```

This uses a conditional expression to use a short, concise prefix for the
primary state (always named "default") but to select a longer prefix that
includes the state name when you are working in one of your additional states.

Elsewhere in your configuration you can use `local.name_prefix` to refer
to that prefix. For example:

```hcl
  name = "${local.name_prefix}users-database"
```

If your configuration includes calls to shared modules, those modules may also
need to select unique object names for a remote system. A well-designed module
with that requirement should include an input variable that affects how it
generates such object names, and so you can pass values you've derived from
`terraform.state` into child modules just as with any other input variable:

```hcl
module "example" {
  source = "../modules/example"

  name_prefix = local.name_prefix
}
```

~> **Shared modules should never refer to `terraform.state`**. The set of
state names belongs to each configuration separately and so a well-designed
shared module should not attempt to vary its behavior directly by the current
state name. Instead, define input variables for all of the customizable
behaviors of your module and let the root module decide how to set those
variables, possibly by referring to `terraform.state` itself.

Naming collisions are one of the unfortunate complexities of using multiple
states associated with the same configuration, and are therefore one of the
reasons why we recommend thinking of multiple states as a last resort.

## Multiple State Internals

Creating a new named state is exactly the same as moving your state file
to a different filename and creating a new, empty one in its place. The
explicit capability for multiple states just aims to reduce the risk of
expensive mistakes when doing so, and to provide the same workflow for
both local state and remote state in various backends.

When you are using local state, Terraform keeps the main state in a local
file called `terraform.tfstate`. When you create additional named states,
Terraform will represent each one by creating a new subdirectory under the
directory `terraform.tfstate.d`.

For [remote state](/docs/state/remote.html), the handling of multiple states
varies between different backends. Some backends don't support additional states
at all, and those which do will often treat the additional states differently
than the primary state in order to give a simpler experience for the normal
case of only having a single state.

## Multiple States with Terraform Cloud

Multiple states associated with one configuration is a mechanism that was
originally designed for Terraform CLI long before it was integrated directly
with Terraform Cloud. Consequently, multiple states are not particularly
useful for Terraform Cloud users.

The primary way to use the `remote` backend is to associate your configuration
with a single remote workspace, by setting the `name` argument in the
`workspaces` configuration block:

```hcl
terraform {
  backend "remote" {
    hostname = "app.terraform.io"
    organization = "company"

    workspaces {
      name = "my-app-prod"
    }
  }
}
```

When configured this way, the remote backend does not support multiple
states at all. Each Terraform Cloud workspace has exactly one state associated
with it, so there is nowhere to record additional states.

If you are already using multiple states with Terraform CLI and wish to
preserve them when migrating to Terraform Cloud, you _must_ configure the
`remote` backend using the `prefix` argument instead, which associates your
configuration with multiple Terraform Cloud workspaces that all share a
common name prefix:

```hcl
terraform {
  backend "remote" {
    hostname = "app.terraform.io"
    organization = "company"

    workspaces {
      prefix = "my-app-"
    }
  }
}
```

If you have Terraform CLI states named "default" and "refactor" then to migrate
them to Terraform Cloud would require creating Terraform Cloud workspaces
called "my-app-default" and "my-app-refactor" in the organization named
"company".

When set up in this way, the remote backend treats each workspace matching
the given prefix as if it were a state, where the states are given names by
trimming off the prefix from the workspace name. In the above example, the
`remote` backend would trim off "my-app-" from the two workspace names
and thus recreate the original situation of two states named "default"
and "refactor" as far as Terraform CLI is concerned.

The remote backend supports this mode as a migration path for those who are
already using multiple states and need to preserve them while adopting
Terraform Cloud, but if you are not already using multiple states then you
should write all of your configurations to correspond with only a single
Terraform Cloud workspace, and thus have only a single state each. For more
information on the Terraform Cloud workspace concept, see
[_Workspaces_ in the Terraform Cloud documentation](/docs/cloud/workspaces/).
