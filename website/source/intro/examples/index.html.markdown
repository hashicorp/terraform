---
layout: "intro"
page_title: "Example Configurations"
sidebar_current: "examples"
description: |-
  These examples are designed to help you understand some of the ways Terraform can be used.
---

# Example Configurations

These examples are designed to help you understand some
of the ways Terraform can be used.

All examples are ready to run as-is. Terraform will
ask for input of things such as variables and API keys. If you want to
continue using the example, you should save those parameters in a
"terraform.tfvars" file or in a `provider` config block.

~> **Warning!** The examples use real providers that launch _real_ resources.
That means they can cost money to experiment with. To avoid unexpected charges,
be sure to understand the price of resources before launching them, and verify
any unneeded resources are cleaned up afterwards.

Experimenting in this way can help you learn how the Terraform lifecycle
works, as well as how to repeatedly create and destroy infrastructure.

If you're completely new to Terraform, we recommend reading the
[getting started guide](/intro/getting-started/install.html) before diving into
the examples. However, due to the intuitive configuration Terraform
uses it isn't required.

## Examples

All of the examples are in the
["examples" directory within the Terraform source code](https://github.com/hashicorp/terraform/tree/master/examples). Each example (as well as the examples
directory) has a README explaining the goal of the example.

To use these examples, Terraform must first be installed on your machine.
You can install Terraform from the [downloads page](/downloads.html).
Once installed, you can use two steps to view and run the examples.

To clone any examples, run `terraform init` with the URL to the example:

```
$ terraform init github.com/hashicorp/terraform/examples/aws-two-tier
...
```

This will put the example files within your working directory. You can then
use your own editor to read and browse the configurations. This command will
not _run_ any code.

~> If you want to browse the files before downloading them, you can [view
them on GitHub](https://github.com/hashicorp/terraform/tree/master/examples/aws-two-tier).

If you want to run the example, just run `terraform apply`:

```
$ terraform apply
...
```

Terraform will interactively ask for variable input and potentially
provider configuration, and will start executing.

When you're done with the example, run `terraform destroy` to clean up.
