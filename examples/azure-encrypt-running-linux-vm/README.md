# Enable encryption on a running Linux VM. 

This Terraform template was based on [this](https://github.com/Azure/azure-quickstart-templates/tree/master/201-encrypt-running-linux-vm) Azure Quickstart Template. Changes to the ARM template that may have occurred since the creation of this example may not be reflected in this Terraform template.

This template enables encryption on a running linux vm using AAD client secret. This template assumes that the VM is located in the same region as the resource group. If not, please edit the template to pass appropriate location for the VM sub-resources.

## Prerequisites:
Azure Disk Encryption securely stores the encryption secrets in a specified Azure Key Vault.

Create the Key Vault and assign appropriate access policies. You may use this script to ensure that your vault is properly configured: [AzureDiskEncryptionPreRequisiteSetup.ps1](https://github.com/Azure/azure-powershell/blob/10fc37e9141af3fde6f6f79b9d46339b73cf847d/src/ResourceManager/Compute/Commands.Compute/Extension/AzureDiskEncryption/Scripts/AzureDiskEncryptionPreRequisiteSetup.ps1)

Use the below PS cmdlet for getting the `key_vault_secret_url` and `key_vault_resource_id`.

```
    Get-AzureRmKeyVault -VaultName $KeyVaultName -ResourceGroupName $rgname
```

References:

- [White paper](https://azure.microsoft.com/en-us/documentation/articles/azure-security-disk-encryption/)
- [Explore Azure Disk Encryption with Azure Powershell](https://blogs.msdn.microsoft.com/azuresecurity/2015/11/16/explore-azure-disk-encryption-with-azure-powershell/)
- [Explore Azure Disk Encryption with Azure PowerShell – Part 2](http://blogs.msdn.com/b/azuresecurity/archive/2015/11/21/explore-azure-disk-encryption-with-azure-powershell-part-2.aspx)


## main.tf
The `main.tf` file contains the actual resources that will be deployed. It also contains the Azure Resource Group definition and any defined variables. 

## outputs.tf
This data is outputted when `terraform apply` is called, and can be queried using the `terraform output` command.

## provider.tf
You may leave the provider block in the `main.tf`, as it is in this template, or you can create a file called `provider.tf` and add it to your `.gitignore` file.

Azure requires that an application is added to Azure Active Directory to generate the `client_id`, `client_secret`, and `tenant_id` needed by Terraform (`subscription_id` can be recovered from your Azure account details). Please go [here](https://www.terraform.io/docs/providers/azurerm/) for full instructions on how to create this to populate your `provider.tf` file.

## terraform.tfvars
If a `terraform.tfvars` file is present in the current directory, Terraform automatically loads it to populate variables. We don't recommend saving usernames and password to version control, but you can create a local secret variables file and use `-var-file` to load it.

If you are committing this template to source control, please insure that you add this file to your .gitignore file.

## variables.tf
The `variables.tf` file contains all of the input parameters that the user can specify when deploying this Terraform template.

![graph](/examples/azure-encrypt-running-linux-vm/graph.png)