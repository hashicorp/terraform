---
layout: "enterprise"
page_title: "Starting - Runs - Terraform Enterprise"
sidebar_current: "docs-enterprise-runs-starting"
description: |-
  How to start runs in Terraform Enterprise.
---


# Starting Terraform Runs

There are a variety of ways to queue a Terraform run in Terraform Enterprise. In addition to
`terraform push`, you can connect your environment
to GitHub and runs based on new commits. You can
also intelligently queue new runs when linked artifacts are uploaded or changed.
Remember from the [previous section about Terraform runs](/docs/enterprise/runs)
that it is safe to trigger many plans without consequence since Terraform plans
do not change infrastructure.


## Terraform Push

Terraform `push` is a [Terraform command](https://terraform.io/docs/commands/push.html)
that packages and uploads a set of Terraform configuration and directory to the platform. This then creates a run
which performs `terraform plan` and `terraform apply` against the uploaded
configuration.

The directory is included in order to run any associated provisioners,
that might use local files. For example, a remote-exec provisioner
that executes a shell script.

By default, everything in your directory is uploaded as part of the push.

However, it's not always the case that the entire directory should be uploaded. Often,
temporary or cache directories and files like `.git`, `.tmp` will be included by default, which
can cause failures at certain sizes and should be avoided. You can
specify [exclusions](https://terraform.io/docs/commands/push.html) to avoid this situation.

Terraform also allows for a [VCS option](https://terraform.io/docs/commands/push.html#_vcs_true)
that will detect your VCS (if there is one) and only upload the files that are tracked by the VCS. This is
useful for automatically excluding ignored files. In a VCS like git, this
basically does a `git ls-files`.


## GitHub Webhooks

Optionally, GitHub can be used to import Terraform configuration. When used
within an organization, this can be extremely valuable for keeping differences
in environments and last mile changes from occurring before an upload.

After you have [connected your GitHub account to Terraform Enterprise](/docs/enterprise/vcs/github.html),
you can connect your environment to the target
GitHub repository. The GitHub repository will be linked to the Terraform Enterprise
configuration, and GitHub will start sending webhooks. Certain
GitHub webhook events, detailed below, will cause the repository to be
automatically ingressed into Terraform and stored, along with references to the
GitHub commits and authorship information.

Currently, an environment must already exist to be connected to GitHub. You can
create the environment with `terraform push`, detailed above, and then link it
to GitHub.

Each ingress will trigger a Terraform plan. If you have auto-apply enabled then
the plan will also be applied.

You can disable an ingress by adding the text `[atlas skip]` or `[ci skip]` to
your commit message.

Supported GitHub webhook events:

- pull_request (on by default)
  - ingress when opened or reopened
  - ingress when synchronized (new commits are pushed to the branch)
- push (on by default)
  - ingress when a tag is created
  - ingress when the default branch is updated
    - note: the default branch is either configured on your configuration's
      integrations tab, or if that is blank it is the GitHub
      repository's default branch
- create (off by default)
  - ingress when a tag is created
  - note: if you want to only run on tag creation, turn on create events and
    turn off push events

## Artifact Uploads

Upon successful completion of a Terraform run, the remote state is parsed and
any [artifacts](/docs/enterprise/artifacts/artifact-provider.html) are detected that
were referenced. When new versions of those referenced artifacts are uploaded, you have the option to automatically queue a new Terraform run.

For example, consider the following Terraform configuration which references an
artifact named "worker":

```hcl
resource "aws_instance" "worker" {
  ami           = "${atlas_artifact.worker.metadata_full.region-us-east-1}"
  instance_type = "m1.small"
}
```

When a new version of the and artifact "worker" is uploaded either manually
or as the output of a [Packer build](/docs/enterprise/packer/builds/starting.html), a Terraform plan can be automatically triggered with this new artifact version.
You can enable this feature on a per-environment basis from the
environment settings page.

Combined with
[Terraform auto apply](/docs/enterprise/runs/automatic-applies.html), you can
continuously deliver infrastructure using Terraform and Terraform Enterprise.

## Terraform Plugins

If you are using a custom [Terraform Plugin](https://www.terraform.io/docs/plugins/index.html)
binary for a provider or provisioner that's not currently in a released
version of Terraform, you can still use this in Terraform Enterprise.

All you need to do is include a Linux AMD64 binary for the plugin in the
directory in which Terraform commands are run from; it will then be used next time you `terraform push` or ingress from GitHub.
