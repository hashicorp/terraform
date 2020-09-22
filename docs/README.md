# Terraform Core Codebase Documentation

This directory contains some documentation about the Terraform Core codebase,
aimed at readers who are interested in making code contributions.

If you're looking for information on _using_ Terraform, please instead refer
to [the main Terraform CLI documentation](https://www.terraform.io/docs/cli-index.html).

## Terraform Core Architecture Documents

* [Terraform Core Architecture Summary](./architecture.md): an overview of the
  main components of Terraform Core and how they interact. This is the best
  starting point if you are diving in to this codebase for the first time.

* [Resource Instance Change Lifecycle](./resource-instance-change-lifecycle.md):
  a description of the steps in validating, planning, and applying a change
  to a resource instance, from the perspective of the provider plugin RPC
  operations. This may be useful for understanding the various expectations
  Terraform enforces about provider behavior, either if you intend to make
  changes to those behaviors or if you are implementing a new Terraform plugin
  SDK and so wish to conform to them.

  (If you are planning to write a new provider using the _official_ SDK then
  please refer to [the Extend documentation](https://www.terraform.io/docs/extend/index.html)
  instead; it presents similar information from the perspective of the SDK
  API, rather than the plugin wire protocol.)

* [Plugin Protocol](./plugin-protocol/): gRPC/protobuf definitions for the
  plugin wire protocol and information about its versioning strategy.

  This documentation is for SDK developers, and is not necessary reading for
  those implementing a provider using the official SDK.

## Contribution Guides

* [Maintainer Etiquette](./maintainer-etiquette.md): guidelines and expectations
  for those who serve as Pull Request reviewers, issue triagers, etc.
