---
layout: "docs"
page_title: "Command: taint"
sidebar_current: "docs-commands-taint"
description: |-
  The `terraform taint` command informs Terraform that a particular object
  is damaged or degraded.
---

# Command: taint

The `terraform taint` command informs Terraform that a particular object has
become degraded or damaged. Terraform represents this by marking the
object as "tainted" in the Terraform state, in which case Terraform will
propose to replace it in the next plan you create.

## Usage

Usage: `terraform taint [options] address`

The `address` argument is the address of the resource to mark as tainted.
The address is in
[the resource address syntax](/docs/cli/state/resource-addressing.html) syntax,
as shown in the output from other commands, such as:

 * `aws_instance.foo`
 * `aws_instance.bar[1]`
 * `aws_instance.baz[\"key\"]` (quotes in resource addresses must be escaped on the command line, so that they will not be interpreted by your shell)
 * `module.foo.module.bar.aws_instance.qux`

This command accepts the following options:

* `-allow-missing` - If specified, the command will succeed (exit code 0)
  even if the resource is missing. The command might still return an error
  for other situations, such as if there is a problem reading or writing
  the state.

* `-lock=false` - Disables Terraform's default behavior of attempting to take
  a read/write lock on the state for the duration of the operation.

* `-lock-timeout=DURATION` - Unless locking is disabled with `-lock=false`,
  instructs Terraform to retry acquiring a lock for a period of time before
  returning an error. The duration syntax is a number followed by a time
  unit letter, such as "3s" for three seconds.

* `-ignore-remote-version` - When using the enhanced remote backend with
  Terraform Cloud, continue even if remote and local Terraform versions differ.
  This may result in an unusable Terraform Cloud workspace, and should be used
  with extreme caution.

For configurations using
[the `local` backend](/docs/language/settings/backends/local.html) only,
`terraform taint` also accepts the legacy options
[`-state`, `-state-out`, and `-backup`](/docs/language/settings/backends/local.html#command-line-arguments).
