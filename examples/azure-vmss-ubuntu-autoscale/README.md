### Autoscale a Linux VM Scale Set ###

This Terraform template was based on [this](https://github.com/Azure/azure-quickstart-templates/tree/master/201-vmss-ubuntu-autoscale) Azure Quickstart Template. Changes to the ARM template that may have occurred since the creation of this example may not be reflected in this Terraform template.

This template deploys a desired count Linux VM Scale Set integrated with Azure autoscale. Once the VMSS is deployed, the user can deploy an application inside each of the VMs (either by directly logging into the VMs or via a [`remote-exec` provisioner](https://www.terraform.io/docs/provisioners/remote-exec.html)).

The Autoscale rules are configured as follows:
- sample for CPU (\\Processor\\PercentProcessorTime) in each VM every 1 Minute
- if the Percent Processor Time is greater than 50% for 5 Minutes, then the scale out action (add more VM instances one at a time) is triggered
- once the scale out action is completed, the cool down period is 1 Minute

## main.tf
The `main.tf` file contains the actual resources that will be deployed. It also contains the Azure Resource Group definition and any defined variables.

## outputs.tf
This data is outputted when `terraform apply` is called, and can be queried using the `terraform output` command.

## provider.tf
You may leave the provider block in the `main.tf`, as it is in this template, or you can create a file called `provider.tf` and add it to your `.gitignore` file.

Azure requires that an application is added to Azure Active Directory to generate the `client_id`, `client_secret`, and `tenant_id` needed by Terraform (`subscription_id` can be recovered from your Azure account details). Please go [here](https://www.terraform.io/docs/providers/azurerm/) for full instructions on how to create this to populate your `provider.tf` file.

## terraform.tfvars
If a `terraform.tfvars` file is present in the current directory, Terraform automatically loads it to populate variables. We don't recommend saving usernames and password to version control, but you can create a local secret variables file and use `-var-file` to load it.

## variables.tf
The `variables.tf` file contains all of the input parameters that the user can specify when deploying this Terraform template.
