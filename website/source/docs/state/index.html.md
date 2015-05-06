---
layout: "docs"
page_title: "State"
sidebar_current: "docs-state"
description: |-
  Terraform stores state which caches the known state of the world the last time Terraform ran.
---

# State

Terraform stores the state of your managed infrastructure from the last
time Terraform was run. By default this state is stored in a local file
named "terraform.tfstate", but it can also be stored remotely, which works
better in a team environment.

Terraform uses this local state to create plans and make changes to your
infrastructure. Prior to any operation, Terraform does a
[refresh](/docs/commands/refresh.html) to update the state with the
real infrastructure.

-> **Note:** Terraform currently requires the state to exist after Terraform
has been run. Technically,
at some point in the future, Terraform should be able to populate the local
state file with the real infrastructure if the file didn't exist. But currently,
Terraform state is a mixture of both a cache and required configuration and
isn't optional.

## Format

The state is in JSON format and Terraform will promise backwards compatibility
with the state file. The JSON format makes it easy to write tools around the
state if you want or to modify it by hand in the case of a Terraform bug.
The "version" field on the state contents allows us to transparently move
the format forward if we make modifications.

