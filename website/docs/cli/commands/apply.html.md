---
layout: "docs"
page_title: "Command: apply"
sidebar_current: "docs-commands-apply"
description: |-
  The `terraform apply` command executes the actions proposed in a Terraform plan.
---

# Command: apply

> **Hands-on:** Try the [Terraform: Get Started](https://learn.hashicorp.com/collections/terraform/aws-get-started?utm_source=WEBSITE&utm_medium=WEB_IO&utm_offer=ARTICLE_PAGE&utm_content=DOCS) collection on HashiCorp Learn.

The `terraform apply` command executes the actions proposed in a Terraform
plan.

The most straightforward way to use `terraform apply` is to run it without
any arguments at all, in which case it will automatically create a new
execution plan (as if you had run `terraform plan`) and then prompt you to
approve that plan, before taking the indicated actions.

Another way to use `terraform apply` is to pass it the filename of a saved
plan file you created earlier with `terraform plan -out=...`, in which case
Terraform will apply the changes in the plan without any confirmation prompt.
This two-step workflow is primarily intended for when
[running Terraform in automation](https://learn.hashicorp.com/tutorials/terraform/automate-terraform?in=terraform/automation&utm_source=WEBSITE&utm_medium=WEB_IO&utm_offer=ARTICLE_PAGE&utm_content=DOCS).

## Usage

Usage: `terraform apply [options] [plan]`

The behavior of `terraform apply` differs significantly depending on whether
you pass it the filename of a previously-saved plan file.

In the default case, with no saved plan file, `terraform apply` effectively
runs [`terraform plan`](./plan.html) internally itself in order to propose a
new plan. In that case, `terraform apply` supports all of the same
[Planning Modes](./plan.html#planning-modes) and
[Planning Options](./plan.html#planning-options) that `terraform plan`
would accept, so you can customize how Terraform will create the plan.
Terraform will prompt you to approve the plan before taking the described
actions, unless you override that prompt using the `-auto-approve` option.

If you pass the filename of a previously-saved plan file, none of the options
related to planning modes and planning options are supported, because Terraform
will instead use the options that you set on the earlier `terraform plan` call
that created the plan file.

The following options allow you to change various details about how the
apply command executes and reports on the apply operation. If you are running
`terraform apply` _without_ a previously-saved plan file, these options are
_in addition to_ the planning modes and planning options described for
[`terraform plan`](./plan.html).

* `-auto-approve` - Skips interactive approval of plan before applying. This
  option is ignored when you pass a previously-saved plan file, because
  Terraform considers you passing the plan file as the approval and so
  will never prompt in that case.

* `-compact-warnings` - Shows any warning messages in a compact form which
  includes only the summary messages, unless the warnings are accompanied by
  at least one error and thus the warning text might be useful context for
  the errors.

* `-input=false` - Disables all of Terraform's interactive prompts. Note that
  this also prevents Terraform from prompting for interactive approval of a
  plan, so Terraform will conservatively assume that you do not wish to
  apply the plan, causing the operation to fail. If you wish to run Terraform
  in a non-interactive context, see
  [Running Terraform in Automation](https://learn.hashicorp.com/tutorials/terraform/automate-terraform?in=terraform/automation&utm_source=WEBSITE&utm_medium=WEB_IO&utm_offer=ARTICLE_PAGE&utm_content=DOCS) for some
  different approaches.

* `-lock=false` - Don't hold a state lock during the operation. This is
   dangerous if others might concurrently run commands against the same
   workspace.

* `-lock-timeout=DURATION` - Unless locking is disabled with `-lock=false`,
  instructs Terraform to retry acquiring a lock for a period of time before
  returning an error. The duration syntax is a number followed by a time
  unit letter, such as "3s" for three seconds.

* `-no-color` - Disables terminal formatting sequences in the output. Use this
  if you are running Terraform in a context where its output will be
  rendered by a system that cannot interpret terminal formatting.

* `-parallelism=n` - Limit the number of concurrent operation as Terraform
  [walks the graph](/docs/internals/graph.html#walking-the-graph). Defaults to
  10.

For configurations using
[the `local` backend](/docs/language/settings/backends/local.html) only,
`terraform apply` also accepts the legacy options
[`-state`, `-state-out`, and `-backup`](/docs/language/settings/backends/local.html#command-line-arguments).

## Passing a Different Configuration Directory

Terraform v0.13 and earlier also accepted a directory path in place of the
plan file argument to `terraform apply`, in which case Terraform would use
that directory as the root module instead of the current working directory.

That usage was deprecated in Terraform v0.14 and removed in Terraform v0.15.
If your workflow relies on overriding the root module directory, use
[the `-chdir` global option](./#switching-working-directory-with-chdir)
instead, which works across all commands and makes Terraform consistently look
in the given directory for all files it would normally read or write in the
current working directory.

If your previous use of this legacy pattern was also relying on Terraform
writing the `.terraform` subdirectory into the current working directory even
though the root module directory was overridden, use
[the `TF_DATA_DIR` environment variable](/docs/cli/config/environment-variables.html#tf_data_dir)
to direct Terraform to write the `.terraform` directory to a location other
than the current working directory.
