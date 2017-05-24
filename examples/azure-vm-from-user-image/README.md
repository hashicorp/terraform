# [Create a Virtual Machine from a User Image](https://docs.microsoft.com/en-us/azure/virtual-machines/linux/cli-deploy-templates#create-a-custom-vm-image)

This Terraform template was based on [this](https://github.com/Azure/azure-quickstart-templates/tree/master/101-vm-from-user-image) Azure Quickstart Template. Changes to the ARM template that may have occurred since the creation of this example may not be reflected here.

> Prerequisite - The generalized image VHD should exist, as well as a Storage Account for boot diagnostics

This template allows you to create a Virtual Machine from an unmanaged User image vhd. This template also deploys a Virtual Network, Public IP addresses and a Network Interface.

## main.tf
The `main.tf` file contains the actual resources that will be deployed. It also contains the Azure Resource Group definition and any defined variables.

## outputs.tf
This data is outputted when `terraform apply` is called, and can be queried using the `terraform output` command.

## provider.tf
Azure requires that an application is added to Azure Active Directory to generate the `client_id`, `client_secret`, and `tenant_id` needed by Terraform (`subscription_id` can be recovered from your Azure account details). Please go [here](https://www.terraform.io/docs/providers/azurerm/) for full instructions on how to create this to populate your `provider.tf` file.

## terraform.tfvars
If a `terraform.tfvars` file is present in the current directory, Terraform automatically loads it to populate variables. We don't recommend saving usernames and password to version control, but you can create a local secret variables file and use `-var-file` to load it.

If you are committing this template to source control, please insure that you add this file to your `.gitignore` file.

## variables.tf
The `variables.tf` file contains all of the input parameters that the user can specify when deploying this Terraform template.

![graph](/examples/azure-vm-from-user-image/graph.png)