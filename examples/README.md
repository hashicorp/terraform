# Terraform Examples

This folder contains a set of Terraform examples. These examples each
have their own README you can read for more details on what the example
does.

To run any example, just run `terraform apply` within that directory
if you have Terraform checked out. Or, you can run it directly from git:

```
$ terraform init github.com/hashicorp/terraform/examples/cross-provider
...

$ terraform apply
...
```

## Provider-specific Examples

Terraform providers each live in their own repository. Some of these
repositories contain documentation specific to their provider:

* [AliCloud Examples](https://github.com/terraform-providers/terraform-provider-alicloud/tree/master/examples)
* [Amazon Web Services Examples](https://github.com/terraform-providers/terraform-provider-aws/tree/master/examples)
* [Azure Examples](https://github.com/terraform-providers/terraform-provider-azurerm/tree/master/examples)
* [CenturyLink Cloud Examples](https://github.com/terraform-providers/terraform-provider-clc/tree/master/examples)
* [Google Cloud Examples](https://github.com/terraform-providers/terraform-provider-google/tree/master/examples)
