# Create a specialized virtual machine in an existing virtual network [![Build Status](https://travis-ci.org/harijayms/terraform.svg?branch=topic-201-vm-specialized-vhd-existing-vnet)](https://travis-ci.org/harijayms/terraform)

This Terraform template was based on [this](https://github.com/Azure/azure-quickstart-templates/tree/master/201-vm-specialized-vhd-existing-vnet) Azure Quickstart Template. Changes to the ARM template that may have occurred since the creation of this example may not be reflected in this Terraform template.

## Prerequisites

- VHD file from which to create a VM that already exists in a storage account
- Name of the existing VNET and subnet to which the new virtual machine will connect
- Name of the Resource Group in which the VNET resides


### NOTE

This template will create an additional Standard_GRS storage account for enabling boot diagnostics each time you execute this template. To avoid running into storage account limits, it is best to delete the storage account when the VM is deleted.

This template creates a VM from a specialized VHD and lets you connect it to an existing VNET that can reside in a different Resource Group from which the virtual machine resides.

_Please note: This deployment template does not create or attach an existing Network Security Group to the virtual machine._

## main.tf
The `main.tf` file contains the actual resources that will be deployed. It also contains the Azure Resource Group definition and any defined variables. 

## outputs.tf
This data is outputted when `terraform apply` is called, and can be queried using the `terraform output` command.

## provider.tf
Azure requires that an application is added to Azure Active Directory to generate the `client_id`, `client_secret`, and `tenant_id` needed by Terraform (`subscription_id` can be recovered from your Azure account details). Please go [here](https://www.terraform.io/docs/providers/azurerm/) for full instructions on how to create this to populate your `provider.tf` file.

## terraform.tfvars
If a `terraform.tfvars` or any `.auto.tfvars` files are present in the current directory, Terraform automatically loads them to populate variables. We don't recommend saving usernames and password to version control, but you can create a local secret variables file and use the `-var-file` flag or the `.auto.tfvars` extension to load it.

## variables.tf
The `variables.tf` file contains all of the input parameters that the user can specify when deploying this Terraform template.

![graph](/examples/azure-vm-specialized-vhd-existing-vnet/graph.png)
