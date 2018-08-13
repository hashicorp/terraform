---
layout: "docs"
page_title: "Command: providers"
sidebar_current: "docs-commands-providers"
description: |-
  The "providers" sub-command prints information about the providers used
  in the current configuration.
---

# Command: providers

The `terraform providers` command prints information about the providers
used in the current configuration.

Provider dependencies are created in several different ways:

* Explicit use of a `provider` block in configuration, optionally including
  a version constraint.

* Use of any resource belonging to a particular provider in a `resource` or
  `data` block in configuration.

* Existence of any resource instance belonging to a particular provider in
  the current _state_. For example, if a particular resource is removed
  from configuration, it continues to create a dependency on its provider
  until its instances have been destroyed.

This command gives an overview of all of the current dependencies, as an aid
to understanding why a particular provider is needed.

## Usage

Usage: `terraform providers [config-path]`

Pass an explicit configuration path to override the default of using the
current working directory.
