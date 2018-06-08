---
layout: "intro"
page_title: "Example Configurations"
sidebar_current: "examples"
description: |-
  These examples are designed to help you understand some of the ways Terraform can be used.
---

# Example Configurations

The examples in this section illustrate some
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

Our examples are distributed across several repos. [This README file in the Terraform repo has links to all of them.](https://github.com/hashicorp/terraform/tree/master/examples)

To use these examples, Terraform must first be installed on your machine.
You can install Terraform from the [downloads page](/downloads.html).
Once installed, you can download, view, and run the examples.

To use an example, clone the repository that contains it and navigate to its directory. For example, to try the AWS two-tier architecture example:

```
git clone https://github.com/terraform-providers/terraform-provider-aws.git
cd terraform-provider-aws/examples/two-tier
```

You can then use your preferred code editor to browse and read the configurations.
To try out an example, run Terraform's init and apply commands while in the example's directory:

```
$ terraform init
...
$ terraform apply
...
```

Terraform will interactively ask for variable input and potentially
provider configuration, and will start executing.

When you're done with the example, run `terraform destroy` to clean up.
