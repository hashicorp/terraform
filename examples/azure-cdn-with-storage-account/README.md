# Create a CDN Profile, a CDN Endpoint with a Storage Account as origin

This Terraform template was based on [this](https://github.com/Azure/azure-quickstart-templates/tree/master/201-cdn-with-storage-account) Azure Quickstart Template. Changes to the ARM template that may have occurred since the creation of this example may not be reflected in this Terraform template.

This template creates a [CDN Profile](https://docs.microsoft.com/en-us/azure/cdn/cdn-overview) and a CDN Endpoint with the origin as a Storage Account. Note that the user needs to create a public container in the Storage Account in order for CDN Endpoint to serve content from the Storage Account.

# Important

The endpoint will not immediately be available for use, as it takes time for the registration to propagate through the CDN. For Azure CDN from Akamai profiles, propagation will usually complete within one minute. For Azure CDN from Verizon profiles, propagation will usually complete within 90 minutes, but in some cases can take longer.

Users who try to use the CDN domain name before the endpoint configuration has propagated to the POPs will receive HTTP 404 response codes. If it has been several hours since you created your endpoint and you're still receiving 404 responses, please see [Troubleshooting CDN endpoints returning 404 statuses](https://docs.microsoft.com/en-us/azure/cdn/cdn-troubleshoot-endpoint).

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
