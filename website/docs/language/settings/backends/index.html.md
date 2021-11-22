---
layout: "language"
page_title: "Backend Overview - Configuration Language"
description: "A backend defines where and how Terraform performs operations, such as where it stores state files. Learn about recommended backends and how backends work."
---

# Backends

Each Terraform configuration can specify a backend, which defines where
and how operations are performed, where [state](/docs/language/state/index.html)
snapshots are stored, etc.

The rest of this page introduces the concept of backends; the other pages in
this section document how to configure and use backends.

- [Backend Configuration](/docs/language/settings/backends/configuration.html) documents the form
  of a `backend` block, which selects and configures a backend for a
  Terraform configuration.
- This section also includes a page for each of Terraform's built-in backends,
  documenting its behavior and available settings. See the navigation sidebar
  for a complete list.

## Recommended Backends

- If you are still learning how to use Terraform, we recommend using the default
  `local` backend, which requires no configuration.
- If you and your team are using Terraform to manage meaningful infrastructure,
  we recommend using the `remote` backend with [Terraform Cloud](/docs/cloud/index.html)
  or [Terraform Enterprise](/docs/enterprise/index.html).

## Where Backends are Used

Backend configuration is only used by [Terraform CLI](/docs/cli/index.html).
Terraform Cloud and Terraform Enterprise always use their own state storage when
performing Terraform runs, so they ignore any backend block in the
configuration.

But since it's common to
[use Terraform CLI alongside Terraform Cloud](/docs/cloud/run/cli.html)
(and since certain state operations, like [tainting](/docs/cli/commands/taint.html),
can only be performed on the CLI), we recommend that Terraform Cloud users
include a backend block in their configurations and configure the `remote`
backend to use the relevant Terraform Cloud workspace(s).

## Where Backends Come From

Terraform includes a built-in selection of backends; this selection has changed
over time, but does not change very often.

The built-in backends are the only backends. You cannot load additional backends
as plugins.

## What Backends Do

There are two areas of Terraform's behavior that are determined by the backend:

- Where state is stored.
- Where operations are performed.

### State

Terraform uses persistent [state](/docs/language/state/index.html) data to keep track of
the resources it manages. Since it needs the state in order to know which
real-world infrastructure objects correspond to the resources in a
configuration, everyone working with a given collection of infrastructure
resources must be able to access the same state data.

The `local` backend stores state as a local file on disk, but every other
backend stores state in a remote service of some kind, which allows multiple
people to access it. Accessing state in a remote service generally requires some
kind of access credentials, since state data contains extremely sensitive
information.

Some backends act like plain "remote disks" for state files; others support
_locking_ the state while operations are being performed, which helps prevent
conflicts and inconsistencies.

### Operations

"Operations" refers to performing API requests against infrastructure services
in order to create, read, update, or destroy resources. Not every `terraform`
subcommand performs API operations; many of them only operate on state data.

Only two backends actually perform operations: `local` and `remote`.

The `local` backend performs API operations directly from the machine where the
`terraform` command is run. Whenever you use a backend other than `local` or
`remote`, Terraform uses the `local` backend for operations; it only uses the
configured backend for state storage.

The `remote` backend can perform API operations remotely, using Terraform Cloud
or Terraform Enterprise. When running remote operations, the local `terraform`
command displays the output of the remote actions as though they were being
performed locally, but only the remote system requires cloud credentials or
network access to the resources being managed.

Remote operations are optional for the `remote` backend; the settings for the
target Terraform Cloud workspace determine whether operations run remotely or
locally. If local operations are configured, Terraform uses the `remote` backend
for state and the `local` backend for operations, like with the other state
backends.

### Backend Types

Terraform's backends are divided into two main types, according to how they
handle state and operations:

- **Enhanced** backends can both store state and perform operations. There are
  only two enhanced backends: `local` and `remote`.
- **Standard** backends only store state, and rely on the `local` backend for
  performing operations.
