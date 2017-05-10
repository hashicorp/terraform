# Azure Search service

This Terraform template was based on [this](https://github.com/Azure/azure-quickstart-templates/tree/bf842409eeeeb7c4523add3922b204793eb4d85f/101-azure-search-create) Azure Quickstart Template. Changes to the ARM template that may have occurred since the creation of this example may not be reflected in this Terraform template.

This template creates a new Azure Search Service.

If you are unclear as to what parameters are allowed you can check the [Azure Search Management REST API docs on MSDN](https://msdn.microsoft.com/en-us/library/azure/dn832687.aspx).

## main.tf
The `main.tf` file contains the actual resources that will be deployed. It also contains the Azure Resource Group definition and any defined variables.

## outputs.tf
This data is outputted when `terraform apply` is called, and can be queried using the `terraform output` command.

## provider.tf
You may leave the provider block in the `main.tf`, as it is in this template, or you can create a file called `provider.tf` and add it to your `.gitignore` file.

Azure requires that an application is added to Azure Active Directory to generate the `client_id`, `client_secret`, and `tenant_id` needed by Terraform (`subscription_id` can be recovered from your Azure account details). Please go [here](https://www.terraform.io/docs/providers/azurerm/) for full instructions on how to create this to populate your `provider.tf` file.

## terraform.tfvars
If a `terraform.tfvars` file is present in the current directory, Terraform automatically loads it to populate variables. We don't recommend saving usernames and password to version control, but you can create a local secret variables file and use `-var-file` to load it.

If you are committing this template to source control, please insure that you add this file to your `.gitignore` file.

## variables.tf
The `variables.tf` file contains all of the input parameters that the user can specify when deploying this Terraform template.

![graph](/examples/azure-search-create/graph.png)