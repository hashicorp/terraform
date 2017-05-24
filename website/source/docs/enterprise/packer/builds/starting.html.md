---
layout: "enterprise"
page_title: "Starting - Packer Builds - Terraform Enterprise"
sidebar_current: "docs-enterprise-packerbuilds-starting"
description: |-
  Packer builds can be started in Terraform Enterprise in two ways. This post is about how.
---

# Starting Packer Builds in Terraform Enterprise

Packer builds can be started in in two ways: `packer push` to upload the
template and directory or via a GitHub connection that retrieves the contents of
a repository after changes to the default branch (usually master).

### Packer Push

Packer `push` is a
[Packer command](https://packer.io/docs/command-line/push.html) that packages
and uploads a Packer template and directory. This then creates a build which
performs `packer build` against the uploaded template and packaged directory.

The directory is included in order to run any associated provisioners, builds or
post-processors that all might use local files. For example, a shell script or
set of Puppet modules used in a Packer build needs to be part of the upload for
Packer to be run remotely.

By default, everything in your directory is uploaded as part of the push.

However, it's not always the case that the entire directory should be uploaded.
Often, temporary or cache directories and files like `.git`, `.tmp` will be
included by default. This can cause builds to fail at certain sizes and should
be avoided. You can specify
[exclusions](https://packer.io/docs/templates/push.html#exclude) to avoid this
situation.

Packer also allows for a
[VCS option](https://packer.io/docs/templates/push.html#vcs) that will detect
your VCS (if there is one) and only upload the files that are tracked by the
VCS. This is useful for automatically excluding ignored files. In a VCS like
git, this basically does a `git ls-files`.


### GitHub Webhooks

Optionally, GitHub can be used to import Packer templates and configurations.
When used within an organization, this can be extremely valuable for keeping
differences in environments and last mile changes from occurring before an
upload.

After you have [connected your GitHub account](/docs/enterprise/vcs/github.html) to Terraform Enterprise,
you can connect your [Build Configuration](/docs/enterprise/glossary#build-configuration)
to the target GitHub repository. The GitHub repository will be linked to the
Packer configuration, and GitHub will start sending webhooks.
Certain GitHub webhook events, detailed below, will cause the repository to be
automatically ingressed into Terraform Enterprise and stored, along with references to the
GitHub commits and authorship information.

After each ingress the configuration will automatically build.

You can disable an ingress by adding the text `[atlas skip]` or `[ci skip]` to
your commit message.

Supported GitHub webhook events:

- push (on by default)
  - ingress when a tag is created
  - ingress when the default branch is updated
    - note: the default branch is either configured on your configuration's
      integrations tab in Terraform Enterprise, or if that is blank it is the GitHub
      repository's default branch
- create (off by default)
  - ingress when a tag is created
  - note: if you want to only run on tag creation, turn on create events and
    turn off push events
