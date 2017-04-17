# Deploy a simple Linux VM
**ubuntu**

This template allows you to deploy a simple Linux VM using a few different options for the Ubuntu version, using the latest patched version. This will deploy a A1 size VM in the resource group location and return the FQDN of the VM.

This template takes a minimum amount of parameters and deploys a Linux VM, using the latest patched version.

Azure requires that an application is added to Azure Active Directory to generate the client_id, client_secret, and tenant_id needed by Terraform (subscription_id can be recovered from your Azure account details). Please go [here](https://www.terraform.io/docs/providers/azurerm/) for full instructions on how to create this.